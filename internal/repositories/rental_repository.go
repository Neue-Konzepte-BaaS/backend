package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type rentalRepository struct {
	queries *database.Queries
}

func NewRentalRepository(queries *database.Queries) services.RentalRepository {
	return &rentalRepository{queries: queries}
}

func (r *rentalRepository) CreateRental(ctx context.Context, plot, customer, crop uuid.UUID, durationMonths int32) (models.Rental, error) {
	row, err := r.queries.InsertRental(ctx, database.InsertRentalParams{
		Plot:           plot,
		Customer:       customer,
		Crop:           crop,
		DurationMonths: durationMonths,
	})
	if err != nil {
		return models.Rental{}, mapRentalError(err)
	}

	return models.Rental{
		ID:       row.ID,
		PlotID:   plot,
		CropID:   crop,
		Customer: customer,
		StartAt:  row.StartAt.Time,
		EndAt:    row.EndAt.Time,
	}, nil
}

func (r *rentalRepository) GetRentalsByCustomer(ctx context.Context, customer uuid.UUID) ([]models.RentalWithPlot, error) {
	rows, err := r.queries.GetRentalsByCustomer(ctx, customer)
	if err != nil {
		return nil, err
	}

	rentals := make([]models.RentalWithPlot, len(rows))
	for i, row := range rows {
		rentals[i] = models.RentalWithPlot{
			Rental: models.Rental{
				ID:       row.ID,
				PlotID:   row.Plot,
				CropID:   row.Crop,
				Customer: row.Customer,
				StartAt:  row.StartAt.Time,
				EndAt:    row.EndAt.Time,
			},
			Plot: models.Plot{
				ID:               row.Plot,
				Name:             row.PlotName,
				Field:            row.Field,
				Coordinates:      row.Coordinates,
				AreaSquareMeters: row.PlotAreaSquareMeters,
			},
			Crop: models.Crop{
				ID:             row.Crop,
				Name:           row.CropName,
				DurationMonths: row.CropDurationMonths,
			},
		}
	}
	return rentals, nil
}

func (r *rentalRepository) GetActiveRentalsByCustomer(ctx context.Context, customer uuid.UUID) ([]models.ActiveRental, error) {
	rows, err := r.queries.GetActiveRentalsByCustomer(ctx, customer)
	if err != nil {
		return nil, err
	}

	rentals := make([]models.ActiveRental, len(rows))
	for i, row := range rows {
		rentals[i] = models.ActiveRental{
			Rental: models.Rental{
				ID:       row.ID,
				PlotID:   row.Plot,
				CropID:   row.Crop,
				Customer: row.Customer,
				StartAt:  row.StartAt.Time,
				EndAt:    row.EndAt.Time,
			},
			PlotName:  row.PlotName,
			FieldName: row.FieldName,
			Crop: models.Crop{
				ID:             row.Crop,
				Name:           row.CropName,
				DurationMonths: row.CropDurationMonths,
			},
			CurrentWeek: row.CurrentWeek,
			TotalWeeks:  row.TotalWeeks,
		}
	}
	return rentals, nil
}

func (r *rentalRepository) GetRentalsByFarm(ctx context.Context, farm uuid.UUID) ([]models.RentalWithPlotAndCustomer, error) {
	rows, err := r.queries.GetRentalsByFarm(ctx, farm)
	if err != nil {
		return nil, err
	}

	rentals := make([]models.RentalWithPlotAndCustomer, len(rows))
	for i, row := range rows {
		rentals[i] = models.RentalWithPlotAndCustomer{
			Rental: models.Rental{
				ID:       row.ID,
				PlotID:   row.Plot,
				CropID:   row.Crop,
				Customer: row.Customer,
				StartAt:  row.StartAt.Time,
				EndAt:    row.EndAt.Time,
			},
			Plot: models.Plot{
				ID:               row.Plot,
				Name:             row.PlotName,
				Field:            row.Field,
				Coordinates:      row.Coordinates,
				AreaSquareMeters: row.PlotAreaSquareMeters,
			},
			FieldName: row.FieldName,
			Customer: models.Recipient{
				AccountID: row.CustomerID,
				Email:     row.CustomerEmail,
				FirstName: row.CustomerFirstName,
				LastName:  row.CustomerLastName,
			},
		}
	}
	return rentals, nil
}

// mapRentalError turns the Postgres errors produced by the rental table's
// constraints into service sentinels, so handlers can report them as a 409 or
// 404 instead of leaking a raw SQL error as a 500.
func mapRentalError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case exclusionViolation:
		return fmt.Errorf("%s: %w", pgErr.Message, services.ErrPlotUnavailable)
	case foreignKeyViolation:
		// Either the plot id does not exist, or the account has no customer row.
		return fmt.Errorf("%s: %w", pgErr.Message, services.ErrNotFound)
	}
	return err
}
