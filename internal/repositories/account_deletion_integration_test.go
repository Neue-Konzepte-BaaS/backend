package repositories_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/repositories"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// rentInThePast books an approved rental that has already run out.
func rentInThePast(t *testing.T, ctx context.Context, pool *pgxpool.Pool, plot, customer, crop uuid.UUID) uuid.UUID {
	t.Helper()

	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO rental (plot, customer, crop, period, status, message, decided_at)
		VALUES ($1, $2, $3, tstzrange(CURRENT_TIMESTAMP - interval '1 year', CURRENT_TIMESTAMP - interval '6 months'), 'approved', 'please', CURRENT_TIMESTAMP)
		RETURNING id`,
		plot, customer, crop,
	).Scan(&id)
	if err != nil {
		t.Fatalf("inserting past rental: %v", err)
	}
	return id
}

func insertCheckout(t *testing.T, ctx context.Context, pool *pgxpool.Pool, plot, customer, crop uuid.UUID, status string) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		INSERT INTO rental_checkout (customer, plot, crop, start_at, message, stripe_checkout_session_id, status, amount_cents)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP + interval '2 days', 'please', $4, $5, 1000)`,
		customer, plot, crop, "cs_"+uuid.NewString(), status,
	)
	if err != nil {
		t.Fatalf("inserting %s checkout: %v", status, err)
	}
}

func TestGetDeletionBlockers(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	repo := repositories.NewAccountRepository(pool, database.New(pool))

	type party struct{ farmer, customer uuid.UUID }
	seed := func(t *testing.T) (party, uuid.UUID, uuid.UUID) {
		farmer, _, plots, crop := seedFarmWithPlots(t, ctx, pool, 1)
		return party{farmer: farmer, customer: seedCustomer(t, ctx, pool)}, plots[0], crop
	}

	tests := []struct {
		name    string
		arrange func(t *testing.T, p party, plot, crop uuid.UUID)
		want    models.AccountDeletionBlockers
	}{
		{name: "nothing", arrange: func(*testing.T, party, uuid.UUID, uuid.UUID) {}},
		{
			name: "rental that has run out",
			arrange: func(t *testing.T, p party, plot, crop uuid.UUID) {
				rentInThePast(t, ctx, pool, plot, p.customer, crop)
				insertCheckout(t, ctx, pool, plot, p.customer, crop, "completed")
			},
		},
		{
			name: "rental running now",
			arrange: func(t *testing.T, p party, plot, crop uuid.UUID) {
				rentNow(t, ctx, pool, plot, p.customer, crop, 6)
			},
			want: models.AccountDeletionBlockers{OpenRentals: true},
		},
		{
			name: "pending checkout",
			arrange: func(t *testing.T, p party, plot, crop uuid.UUID) {
				insertCheckout(t, ctx, pool, plot, p.customer, crop, "pending")
			},
			want: models.AccountDeletionBlockers{PendingPayment: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, plot, crop := seed(t)
			tt.arrange(t, p, plot, crop)

			// Both sides of a rental are bound by it: the customer who holds
			// it and the farmer whose plot it is on.
			for role, id := range map[string]uuid.UUID{"customer": p.customer, "farmer": p.farmer} {
				got, err := repo.GetDeletionBlockers(ctx, id)
				if err != nil {
					t.Fatalf("%s: %v", role, err)
				}
				if got != tt.want {
					t.Fatalf("%s: blockers = %+v, want %+v", role, got, tt.want)
				}
			}
		})
	}
}

func TestDeleteAccount_Customer(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	queries := database.New(pool)
	repo := repositories.NewAccountRepository(pool, queries)

	_, _, plots, crop := seedFarmWithPlots(t, ctx, pool, 1)
	email := uuid.NewString() + "@example.com"
	customer, err := repo.CreateCustomer(ctx, models.Account{
		FirstName: "Ada", LastName: "Lovelace", Email: email, PasswordHash: "irrelevant",
	}, 76133)
	if err != nil {
		t.Fatalf("creating customer: %v", err)
	}
	rental := rentInThePast(t, ctx, pool, plots[0], customer.ID, crop)

	if err := repo.DeleteAccount(ctx, customer.ID, models.RoleCustomer); err != nil {
		t.Fatalf("deleting account: %v", err)
	}

	if _, err := repo.GetAccountByEmail(ctx, email); !errors.Is(err, services.ErrNotFound) {
		t.Fatalf("login lookup after delete: error = %v, want ErrNotFound", err)
	}
	if _, err := repo.GetAccountByID(ctx, customer.ID); !errors.Is(err, services.ErrNotFound) {
		t.Fatalf("id lookup after delete: error = %v, want ErrNotFound", err)
	}

	var firstName, storedEmail string
	if err := pool.QueryRow(ctx, `SELECT first_name, email FROM account WHERE id = $1`, customer.ID).Scan(&firstName, &storedEmail); err != nil {
		t.Fatalf("reading anonymised row: %v", err)
	}
	if firstName == "Ada" || storedEmail == email {
		t.Fatalf("personal data survived: first_name=%q email=%q", firstName, storedEmail)
	}

	var rentalCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM rental WHERE id = $1`, rental).Scan(&rentalCount); err != nil {
		t.Fatalf("counting rentals: %v", err)
	}
	if rentalCount != 1 {
		t.Fatal("the rental was deleted with the account; it is a payment record and must stay")
	}

	recipients, err := repo.GetAllRecipients(ctx)
	if err != nil {
		t.Fatalf("listing recipients: %v", err)
	}
	for _, r := range recipients {
		if r.AccountID == customer.ID {
			t.Fatal("a deleted account is still a broadcast recipient")
		}
	}
	page, err := repo.ListAccounts(ctx, models.AccountListFilter{Limit: 100})
	if err != nil {
		t.Fatalf("listing accounts: %v", err)
	}
	for _, a := range page.Items {
		if a.ID == customer.ID {
			t.Fatal("a deleted account is still in the admin listing")
		}
	}

	if err := repo.DeleteAccount(ctx, customer.ID, models.RoleCustomer); !errors.Is(err, services.ErrNotFound) {
		t.Fatalf("second delete: error = %v, want ErrNotFound", err)
	}

	// The address is free again for whoever owns it.
	if _, err := repo.CreateCustomer(ctx, models.Account{
		FirstName: "Ada", LastName: "Lovelace", Email: email, PasswordHash: "irrelevant",
	}, 76133); err != nil {
		t.Fatalf("registering the same email again: %v", err)
	}
}

func TestDeleteAccount_Farmer(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	queries := database.New(pool)
	repo := repositories.NewAccountRepository(pool, queries)
	farmRepo := repositories.NewFarmRepository(pool, queries)
	fieldRepo := repositories.NewFieldRepository(queries)
	plotRepo := repositories.NewPlotRepository(queries)
	announcementRepo := repositories.NewAnnouncementRepository(queries)

	farmer, farm, plots, crop := seedFarmWithPlots(t, ctx, pool, 1)
	if err := farmRepo.SetFarmCropRates(ctx, farm, []models.FarmCropRate{{Crop: crop, PriceCentsPerSqmPerWeek: 5}}); err != nil {
		t.Fatalf("pricing crop: %v", err)
	}
	if _, err := announcementRepo.CreateAnnouncement(ctx, farmer, "Hallo", "Erntefest am Samstag", nil, nil); err != nil {
		t.Fatalf("posting announcement: %v", err)
	}
	rentInThePast(t, ctx, pool, plots[0], seedCustomer(t, ctx, pool), crop)

	if err := repo.DeleteAccount(ctx, farmer, models.RoleFarmer); err != nil {
		t.Fatalf("deleting account: %v", err)
	}

	if _, err := farmRepo.GetFarmByID(ctx, farm); !errors.Is(err, services.ErrNotFound) {
		t.Fatalf("public farm page after delete: error = %v, want ErrNotFound", err)
	}
	fields, err := fieldRepo.GetFieldsByFarmWithAvailablePlotStats(ctx, farm)
	if err != nil {
		t.Fatalf("listing public fields: %v", err)
	}
	if len(fields) != 0 {
		t.Fatalf("a deleted farm still shows %d fields", len(fields))
	}
	nearby, err := plotRepo.GetNearestPlots(ctx, 0, 0, nil, 100)
	if err != nil {
		t.Fatalf("searching plots: %v", err)
	}
	for _, p := range nearby {
		if p.Farm == farm {
			t.Fatal("a deleted farm's plot is still offered in the search")
		}
	}
	listing, err := farmRepo.ListFarms(ctx, models.FarmListFilter{Limit: 100})
	if err != nil {
		t.Fatalf("listing farms: %v", err)
	}
	for _, f := range listing.Items {
		if f.ID == farm {
			t.Fatal("a deleted farm is still in the admin listing")
		}
	}

	var name, address string
	if err := pool.QueryRow(ctx, `SELECT name, address FROM farm WHERE id = $1`, farm).Scan(&name, &address); err != nil {
		t.Fatalf("reading anonymised farm: %v", err)
	}
	if name == "Green Acres" || address != "" {
		t.Fatalf("farm data survived: name=%q address=%q", name, address)
	}
	if _, err := farmRepo.GetFarmCropRate(ctx, farm, crop); !errors.Is(err, services.ErrNotFound) {
		t.Fatalf("crop rate after delete: error = %v, want ErrNotFound", err)
	}
	posts, err := announcementRepo.GetAnnouncementsByFarmer(ctx, farmer)
	if err != nil {
		t.Fatalf("listing announcements: %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("%d announcements survived the farmer", len(posts))
	}
}
