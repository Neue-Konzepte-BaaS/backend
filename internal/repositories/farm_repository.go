package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type farmRepository struct {
	pool    *pgxpool.Pool
	queries *database.Queries
}

func NewFarmRepository(pool *pgxpool.Pool, queries *database.Queries) services.FarmRepository {
	return &farmRepository{pool: pool, queries: queries}
}

func (r *farmRepository) GetFarmByID(ctx context.Context, farmID uuid.UUID) (models.Farm, error) {
	row, err := r.queries.GetFarmByID(ctx, farmID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Farm{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return models.Farm{}, err
	}

	farm := models.Farm{
		ID:                row.ID,
		FarmerID:          row.FarmerID,
		Name:              row.Name,
		Address:           row.Address,
		Description:       row.Description,
		TotalSquareMeters: row.TotalSquareMeters,
	}
	if row.FoundedAt.Valid {
		founded := row.FoundedAt.Time
		farm.FoundedAt = &founded
	}
	return farm, nil
}

// GetFarmIDByFarmerID returns ErrNotFound if the account is not a farmer.
func (r *farmRepository) GetFarmIDByFarmerID(ctx context.Context, farmerID uuid.UUID) (uuid.UUID, error) {
	id, err := r.queries.GetFarmIDByFarmerID(ctx, farmerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.UUID{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return uuid.UUID{}, err
	}
	return id, nil
}

func (r *farmRepository) ListFarms(ctx context.Context, filter models.FarmListFilter) (models.Page[models.FarmListing], error) {
	postalCode := pgtype.Int4{}
	if filter.PostalCode != nil {
		postalCode = pgtype.Int4{Int32: *filter.PostalCode, Valid: true}
	}

	rows, err := r.queries.ListFarms(ctx, database.ListFarmsParams{
		Search:       filter.Query,
		PostalCode:   postalCode,
		ResultLimit:  filter.Limit,
		ResultOffset: filter.Offset,
	})
	if err != nil {
		return models.Page[models.FarmListing]{}, err
	}

	page := models.Page[models.FarmListing]{
		Items:  make([]models.FarmListing, len(rows)),
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}
	for i, row := range rows {
		page.Items[i] = toModelFarmListing(row)
	}
	// Every row carries the same window-function count, so the first one
	// answers for all of them. No rows means the page is past the end, and
	// Total stays zero -- the client already learned the real total from the
	// page it got there from.
	if len(rows) > 0 {
		page.Total = rows[0].TotalCount
	}
	return page, nil
}

func (r *farmRepository) GetFarmCropRates(ctx context.Context, farm uuid.UUID) ([]models.FarmCropRate, error) {
	rows, err := r.queries.GetFarmCropRates(ctx, farm)
	if err != nil {
		return nil, err
	}

	rates := make([]models.FarmCropRate, len(rows))
	for i, row := range rows {
		rates[i] = models.FarmCropRate{Crop: row.Crop, PriceCentsPerSqmPerWeek: row.PriceCentsPerSqmPerWeek}
	}
	return rates, nil
}

func (r *farmRepository) GetFarmCropRate(ctx context.Context, farm, crop uuid.UUID) (int32, error) {
	rate, err := r.queries.GetFarmCropRate(ctx, database.GetFarmCropRateParams{Farm: farm, Crop: crop})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return 0, err
	}
	return rate, nil
}

// SetFarmCropRates replaces the farm's crop rates in a single transaction,
// so a caller never observes a partially-updated set -- mirrors
// cropRepository.SetPlotCrops.
func (r *farmRepository) SetFarmCropRates(ctx context.Context, farm uuid.UUID, rates []models.FarmCropRate) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit

	qtx := r.queries.WithTx(tx)

	if err := qtx.DeleteFarmCropRates(ctx, farm); err != nil {
		return fmt.Errorf("clearing farm crop rates: %w", err)
	}

	for _, rate := range rates {
		if err := qtx.InsertFarmCropRate(ctx, database.InsertFarmCropRateParams{
			Farm:                    farm,
			Crop:                    rate.Crop,
			PriceCentsPerSqmPerWeek: rate.PriceCentsPerSqmPerWeek,
		}); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation {
				return fmt.Errorf("%s: %w", pgErr.Message, services.ErrNotFound)
			}
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// toModelFarmListing leaves Plots.Available and Plots.OccupancyRate zero: they
// are derived, and deriving them is the service's job.
func toModelFarmListing(row database.ListFarmsRow) models.FarmListing {
	return models.FarmListing{
		ID:         row.ID,
		FarmerID:   row.FarmerID,
		Name:       row.Name,
		Address:    row.Address,
		PostalCode: row.PostalCode,
		FirstName:  row.FirstName,
		LastName:   row.LastName,
		Email:      row.Email,
		CreatedAt:  row.CreatedAt.Time,
		Fields: models.FieldStatistics{
			Total:            row.FieldCount,
			AreaSquareMeters: row.FieldAreaSquareMeters,
		},
		Plots: models.PlotStatistics{
			Total:            row.PlotCount,
			Rented:           row.RentedPlotCount,
			AreaSquareMeters: row.PlotAreaSquareMeters,
		},
		ActiveRentals: row.ActiveRentalCount,
	}
}
