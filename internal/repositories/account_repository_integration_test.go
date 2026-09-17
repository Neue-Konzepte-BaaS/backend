package repositories_test

import (
	"context"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/repositories"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// seedOrphanAccount writes an account with no subtype row. Nothing in the API
// can produce one -- registration always creates a subtype row in the same
// transaction -- but the derived role has an ELSE branch for it, and the admin
// list is where such a row should become visible.
func seedOrphanAccount(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()

	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO account (first_name, last_name, email, password_hash)
		VALUES ('Orph', 'An', $1, 'irrelevant')
		RETURNING id`, uuid.NewString()+"@example.com").Scan(&id)
	if err != nil {
		t.Fatalf("creating orphan account: %v", err)
	}
	return id
}

func findAccount(t *testing.T, page models.Page[models.AccountListing], id uuid.UUID) models.AccountListing {
	t.Helper()

	for _, account := range page.Items {
		if account.ID == id {
			return account
		}
	}
	t.Fatalf("account %v is not in the listing", id)
	return models.AccountListing{}
}

// Role is not a column; it is which subtype table the account joins to. This
// pins that the listing derives it the same way the login path does.
func TestListAccounts_DerivesRoleFromSubtypeMembership(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	queries := database.New(pool)
	repo := repositories.NewAccountRepository(pool, queries)

	farmer, _, _, _ := seedFarmWithPlots(t, ctx, pool, 1)
	customer := seedCustomer(t, ctx, pool)
	admin, err := repo.CreateAdmin(ctx, models.Account{
		FirstName:    "Ann",
		LastName:     "Admin",
		Email:        uuid.NewString() + "@example.com",
		PasswordHash: "irrelevant",
	})
	if err != nil {
		t.Fatalf("creating admin: %v", err)
	}
	orphan := seedOrphanAccount(t, ctx, pool)

	page, err := repo.ListAccounts(ctx, models.AccountListFilter{Limit: 100})
	if err != nil {
		t.Fatalf("listing accounts: %v", err)
	}

	tests := []struct {
		name string
		id   uuid.UUID
		want models.Role
	}{
		{name: "farmer", id: farmer, want: models.RoleFarmer},
		{name: "customer", id: customer, want: models.RoleCustomer},
		{name: "admin", id: admin.ID, want: models.RoleAdmin},
		// An account with no subtype row has no role, and the listing says so
		// rather than hiding it.
		{name: "orphan", id: orphan, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findAccount(t, page, tt.id).Role; got != tt.want {
				t.Errorf("role = %q, want %q", got, tt.want)
			}
		})
	}

	if page.Total != 4 {
		t.Errorf("total = %d, want 4", page.Total)
	}
}

// A listing must never be able to carry a password hash. models.AccountListing
// has no such field, so this checks the next thing that could leak: that the
// hash is not smuggled into another string field.
func TestListAccounts_CarriesNoPasswordMaterial(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	repo := repositories.NewAccountRepository(pool, database.New(pool))

	seedCustomer(t, ctx, pool)

	page, err := repo.ListAccounts(ctx, models.AccountListFilter{Limit: 100})
	if err != nil {
		t.Fatalf("listing accounts: %v", err)
	}
	if len(page.Items) == 0 {
		t.Fatal("expected at least one account")
	}

	for _, account := range page.Items {
		for _, field := range []string{account.FirstName, account.LastName, account.Email, string(account.Role)} {
			if field == "irrelevant" {
				t.Errorf("account %v carries password material in a listing field", account.ID)
			}
		}
		if account.CreatedAt.IsZero() {
			t.Errorf("account %v has a zero createdAt", account.ID)
		}
	}
}

func TestListAccounts_PagesStablyAndReportsTheFullTotal(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	repo := repositories.NewAccountRepository(pool, database.New(pool))

	// Seeded in one go, so they share a created_at and only the id tiebreaker
	// keeps the order -- and therefore the paging -- deterministic.
	for range 3 {
		seedCustomer(t, ctx, pool)
	}

	seen := map[uuid.UUID]int{}
	for offset := int32(0); offset < 4; offset += 2 {
		page, err := repo.ListAccounts(ctx, models.AccountListFilter{Limit: 2, Offset: offset})
		if err != nil {
			t.Fatalf("listing accounts at offset %d: %v", offset, err)
		}
		if len(page.Items) > 0 && page.Total != 3 {
			t.Errorf("total at offset %d = %d, want 3", offset, page.Total)
		}
		for _, account := range page.Items {
			seen[account.ID]++
		}
	}

	if len(seen) != 3 {
		t.Errorf("saw %d distinct accounts across the pages, want 3", len(seen))
	}
	for id, count := range seen {
		if count != 1 {
			t.Errorf("account %v appeared %d times across the pages, want once", id, count)
		}
	}
}

func TestListAccounts_FiltersNarrowBothItemsAndTotal(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	repo := repositories.NewAccountRepository(pool, database.New(pool))

	seedFarmWithPlots(t, ctx, pool, 1)
	seedFarmWithPlots(t, ctx, pool, 1)
	seedCustomer(t, ctx, pool)

	tests := []struct {
		name   string
		filter models.AccountListFilter
		want   int64
	}{
		{name: "no filter", filter: models.AccountListFilter{Limit: 100}, want: 3},
		{name: "farmers only", filter: models.AccountListFilter{Role: models.RoleFarmer, Limit: 100}, want: 2},
		{name: "customers only", filter: models.AccountListFilter{Role: models.RoleCustomer, Limit: 100}, want: 1},
		{name: "admins only", filter: models.AccountListFilter{Role: models.RoleAdmin, Limit: 100}, want: 0},
		// seedFarmWithPlots names every farmer "Old MacDonald".
		{name: "by name", filter: models.AccountListFilter{Query: "macdonald", Limit: 100}, want: 2},
		{name: "role and search together", filter: models.AccountListFilter{Role: models.RoleCustomer, Query: "macdonald", Limit: 100}, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := repo.ListAccounts(ctx, tt.filter)
			if err != nil {
				t.Fatalf("listing accounts: %v", err)
			}
			if int64(len(page.Items)) != tt.want {
				t.Errorf("items = %d, want %d", len(page.Items), tt.want)
			}
			if tt.want > 0 && page.Total != tt.want {
				t.Errorf("total = %d, want %d", page.Total, tt.want)
			}
		})
	}
}
