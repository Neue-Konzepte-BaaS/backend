package services

import (
	"context"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// EmailSender delivers one message to one recipient. It is a repository in
// the dependency-inversion sense rather than a database one: an outbound port,
// backed by SMTP in production and by a console logger in development.
type EmailSender interface {
	SendMail(email string, displayName string, subject string, message string, isHTML bool, attachments map[string][]byte) error
}

type AccountRepository interface {
	GetAccountByEmail(ctx context.Context, email string) (models.Account, error)
	GetAccountByID(ctx context.Context, id uuid.UUID) (models.Account, error)
	// CreateFarmer atomically inserts the account and its farmer subtype row.
	CreateFarmer(ctx context.Context, account models.Account, farmName string, postalCode int32) (models.Account, error)
	// CreateCustomer atomically inserts the account and its customer subtype row.
	CreateCustomer(ctx context.Context, account models.Account, postalCode int32) (models.Account, error)
	// GetAllRecipients returns every farmer and customer as a notification
	// recipient. Admins are not included: a platform-wide notice is addressed
	// to users, not to the operators sending it.
	GetAllRecipients(ctx context.Context) ([]models.Recipient, error)
	// GetCustomersOfFarmer returns the customers currently renting one of the
	// farmer's plots, each exactly once however many plots they rent. A
	// customer whose rental has ended is not included: the farmer's licence to
	// mail them is the rental itself.
	GetCustomersOfFarmer(ctx context.Context, farmer uuid.UUID) ([]models.Recipient, error)
}

type AnnouncementRepository interface {
	// CreateAnnouncement stores one notice by a farmer and returns it with the
	// farm name already resolved.
	CreateAnnouncement(ctx context.Context, farmer uuid.UUID, subject, body string) (models.AnnouncementWithFarm, error)
	// GetAnnouncementsByFarmer returns the farmer's own notices, newest first.
	GetAnnouncementsByFarmer(ctx context.Context, farmer uuid.UUID) ([]models.AnnouncementWithFarm, error)
	// GetAnnouncementsForCustomer returns the notices of every farmer the
	// customer currently rents from, newest first, each carrying the farm name
	// it came from.
	GetAnnouncementsForCustomer(ctx context.Context, customer uuid.UUID) ([]models.AnnouncementWithFarm, error)
}

type FieldRepository interface {
	CreateField(ctx context.Context, field models.Field) (uuid.UUID, error)
	GetFieldOwner(ctx context.Context, id uuid.UUID) (uuid.UUID, error)
	GetFieldsByFarmer(ctx context.Context, farmer uuid.UUID) ([]models.Field, error)
}

type PlotRepository interface {
	CreatePlot(ctx context.Context, plot models.Plot) (uuid.UUID, error)
	GetPlotsByFields(ctx context.Context, fields []uuid.UUID) ([]models.Plot, error)
	// GetNearestPlots returns up to limit plots ordered by distance from the
	// given point (lon, lat), nearest first.
	GetNearestPlots(ctx context.Context, lon, lat float64, limit int32) ([]models.NearbyPlot, error)
	// GetPlotField returns the id of the field a plot belongs to. Returns
	// ErrNotFound if the plot does not exist.
	GetPlotField(ctx context.Context, plot uuid.UUID) (uuid.UUID, error)
}

type RentalRepository interface {
	// CreateRental books the plot for the customer, starting at the database's
	// current time and running for durationMonths. Returns ErrPlotUnavailable
	// if an existing rental overlaps that period, and ErrNotFound if the plot,
	// crop, or customer does not exist.
	CreateRental(ctx context.Context, plot, customer, crop uuid.UUID, durationMonths int32) (models.Rental, error)
	// GetRentalsByCustomer returns the customer's rentals, newest first,
	// each with the plot and crop it books.
	GetRentalsByCustomer(ctx context.Context, customer uuid.UUID) ([]models.RentalWithPlot, error)
}

type CropRepository interface {
	// CreateCrop adds a new crop to the catalog.
	CreateCrop(ctx context.Context, name string, durationMonths int32) (models.Crop, error)
	// GetAllCrops returns the full crop catalog, ordered by name.
	GetAllCrops(ctx context.Context) ([]models.Crop, error)
	// GetCropByID returns ErrNotFound if the crop does not exist.
	GetCropByID(ctx context.Context, id uuid.UUID) (models.Crop, error)
	// SetPlotCrops replaces the set of crops a plot offers.
	SetPlotCrops(ctx context.Context, plot uuid.UUID, crops []uuid.UUID) error
	// GetCropsByPlot returns the crops offered by a single plot, ordered by name.
	GetCropsByPlot(ctx context.Context, plot uuid.UUID) ([]models.Crop, error)
	// GetCropsByPlots returns the crops offered by each of the given plots,
	// keyed by plot id.
	GetCropsByPlots(ctx context.Context, plots []uuid.UUID) (map[uuid.UUID][]models.Crop, error)
}

type PostalCodeRepository interface {
	// FindCoordinates resolves a German postal code or city name to a
	// lon/lat point. Exactly one of postalCode/city should be set. Returns
	// ErrNotFound if nothing matches.
	FindCoordinates(ctx context.Context, postalCode, city string) (lon, lat float64, err error)
}

type StatisticsRepository interface {
	// GetFarmStatistics aggregates one farmer's own fields, plots and
	// rentals. A farmer who owns nothing gets zeros, never ErrNotFound: the
	// query always returns exactly one row.
	GetFarmStatistics(ctx context.Context, farmer uuid.UUID) (models.Statistics, error)
	// GetPlatformStatistics aggregates every farmer's data and, unlike the
	// farm variant, fills Accounts.
	GetPlatformStatistics(ctx context.Context) (models.Statistics, error)
}
