package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/repositories"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// rentNowInStatus writes a rental covering right now in the given status. A
// request the farmer has not decided on, and one he declined, both keep their
// row with a period covering now — which is exactly what an audience filtering
// on the period alone mistakes for a tenant.
func rentNowInStatus(t *testing.T, ctx context.Context, pool *pgxpool.Pool, plot, customer, crop uuid.UUID, status string) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		INSERT INTO rental (plot, customer, crop, period, status, message, decided_at)
		VALUES ($1, $2, $3, tstzrange(CURRENT_TIMESTAMP - interval '1 day', CURRENT_TIMESTAMP + interval '3 months'), $4, 'please',
		        CASE WHEN $4 = 'requested' THEN NULL ELSE CURRENT_TIMESTAMP END)`,
		plot, customer, crop, status)
	if err != nil {
		t.Fatalf("inserting %s rental: %v", status, err)
	}
}

// rentApprovedNow books the plot through the real request flow and approves
// it, as a farmer would — the only way a customer becomes a tenant.
func rentApprovedNow(t *testing.T, ctx context.Context, rentalRepo services.RentalRepository, plot, customer, crop uuid.UUID) {
	t.Helper()

	rental, err := rentalRepo.CreateRentalRequest(ctx, plot, customer, crop, time.Now(), 6, "")
	if err != nil {
		t.Fatalf("requesting rental: %v", err)
	}
	if _, err := rentalRepo.UpdateRentalStatus(ctx, rental.ID, models.RentalStatusApproved); err != nil {
		t.Fatalf("approving rental: %v", err)
	}
}

// approvedAudience seeds one field with three plots and three customers, one
// per rental status, all covering right now and all growing the same crop.
type approvedAudience struct {
	farmer, field, crop           uuid.UUID
	plots                         []uuid.UUID
	approved, requested, declined uuid.UUID
}

func seedApprovedAudience(t *testing.T, ctx context.Context, pool *pgxpool.Pool) approvedAudience {
	t.Helper()

	farmer, _, plots, crop := seedFarmerWithPlots(t, ctx, pool, 3)
	field, err := repositories.NewPlotRepository(database.New(pool)).GetPlotField(ctx, plots[0])
	if err != nil {
		t.Fatalf("looking up field: %v", err)
	}

	a := approvedAudience{
		farmer:    farmer,
		field:     field,
		crop:      crop,
		plots:     plots,
		approved:  seedCustomer(t, ctx, pool),
		requested: seedCustomer(t, ctx, pool),
		declined:  seedCustomer(t, ctx, pool),
	}
	rentNowInStatus(t, ctx, pool, plots[0], a.approved, crop, "approved")
	rentNowInStatus(t, ctx, pool, plots[1], a.requested, crop, "requested")
	rentNowInStatus(t, ctx, pool, plots[2], a.declined, crop, "declined")
	return a
}

// TestScopedAudiencesReachOnlyApprovedRentals pins the audience rule every
// notice shares: a customer is addressable once the farmer has approved their
// rental, not from the moment they asked for it, and never after a decline.
// GetCustomersOfFarmer and GetAnnouncementsForCustomer already hold to it;
// these are the queries that did not.
func TestScopedAudiencesReachOnlyApprovedRentals(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	accountRepo := repositories.NewAccountRepository(pool, queries)
	a := seedApprovedAudience(t, ctx, pool)

	t.Run("field-scoped announcement", func(t *testing.T) {
		recipients, err := accountRepo.GetCustomersOfFarmerForField(ctx, a.field)
		if err != nil {
			t.Fatalf("getting recipients: %v", err)
		}
		if len(recipients) != 1 || recipients[0].AccountID != a.approved {
			t.Errorf("recipients = %+v, want only the approved tenant %v", recipients, a.approved)
		}
	})

	t.Run("plot-scoped announcement", func(t *testing.T) {
		for i, want := range []uuid.UUID{a.approved, uuid.Nil, uuid.Nil} {
			recipients, err := accountRepo.GetCustomersOfFarmerForPlot(ctx, a.plots[i])
			if err != nil {
				t.Fatalf("getting recipients for plot %d: %v", i, err)
			}
			if want == uuid.Nil && len(recipients) != 0 {
				t.Errorf("plot %d reaches %+v, want nobody (its rental is not approved)", i, recipients)
			}
			if want != uuid.Nil && (len(recipients) != 1 || recipients[0].AccountID != want) {
				t.Errorf("plot %d reaches %+v, want only %v", i, recipients, want)
			}
		}
	})

	t.Run("ripeness notice mail", func(t *testing.T) {
		recipients, err := accountRepo.GetCustomersOfFarmerForFieldAndCrop(ctx, a.field, a.crop)
		if err != nil {
			t.Fatalf("getting recipients: %v", err)
		}
		if len(recipients) != 1 || recipients[0].AccountID != a.approved {
			t.Errorf("recipients = %+v, want only the approved tenant %v", recipients, a.approved)
		}
	})

	t.Run("ripeness notice in the inbox", func(t *testing.T) {
		ripenessRepo := repositories.NewRipenessNoticeRepository(queries)
		if _, err := ripenessRepo.CreateRipenessNotice(ctx, a.farmer, a.field, a.crop); err != nil {
			t.Fatalf("creating ripeness notice: %v", err)
		}

		for customer, want := range map[uuid.UUID]int{a.approved: 1, a.requested: 0, a.declined: 0} {
			notices, err := ripenessRepo.GetRipenessNoticesForCustomer(ctx, customer)
			if err != nil {
				t.Fatalf("getting notices: %v", err)
			}
			if len(notices) != want {
				t.Errorf("customer %v reads %d ripeness notices, want %d", customer, len(notices), want)
			}
		}
	})
}
