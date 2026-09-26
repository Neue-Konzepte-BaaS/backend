package services

import (
	"context"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// EmailSender delivers one message to one recipient. It is a repository in
// the dependency-inversion sense rather than a database one: an outbound port,
// backed by SMTP in production and by a console logger in development.
type EmailSender interface {
	SendMail(email string, displayName string, subject string, message string, isHTML bool, attachments map[string][]byte) error
}

// WebhookEventType is the subset of Stripe event types this codebase acts
// on.
type WebhookEventType string

const (
	WebhookEventCheckoutCompleted WebhookEventType = "checkout.session.completed"
	WebhookEventCheckoutExpired   WebhookEventType = "checkout.session.expired"
)

// WebhookEvent is the subset of a Stripe webhook event this codebase acts
// on, already reduced from the Stripe SDK's own types so that nothing above
// the repositories package needs to import it.
type WebhookEvent struct {
	Type              WebhookEventType
	CheckoutSessionID string
}

// PaymentGateway is an outbound port to Stripe, in the same
// dependency-inversion sense as EmailSender: this package stays free of the
// Stripe SDK, which only the repositories package (its implementation)
// imports.
type PaymentGateway interface {
	// CreateCheckoutSession opens a Stripe Embedded Checkout session for a
	// one-off payment of amountCents (EUR), described by description, that
	// redirects the top-level page to returnURL once paid. Returns the
	// session id, to persist against the local rental_checkout row, and the
	// client secret the frontend's embedded checkout iframe needs.
	CreateCheckoutSession(ctx context.Context, amountCents int64, description, returnURL string) (sessionID, clientSecret string, err error)
	// RefundCheckoutSession refunds the full payment collected by the given
	// Stripe Checkout Session.
	RefundCheckoutSession(ctx context.Context, sessionID string) error
	// ParseWebhookEvent verifies payload against the Stripe-Signature header
	// using the configured webhook secret. ok is false for any event type
	// this codebase does not act on, which the caller should treat as a
	// no-op, not an error. Returns ErrInvalidWebhookSignature if
	// verification fails.
	ParseWebhookEvent(payload []byte, sigHeader string) (event WebhookEvent, ok bool, err error)
}

type AccountRepository interface {
	GetAccountByEmail(ctx context.Context, email string) (models.Account, error)
	GetAccountByID(ctx context.Context, id uuid.UUID) (models.Account, error)
	// CreateAdmin atomically inserts the account and its admin subtype row.
	CreateAdmin(ctx context.Context, account models.Account) (models.Account, error)
	// CreateFarmer atomically inserts the account and its farmer subtype row.
	CreateFarmer(ctx context.Context, account models.Account, farmName string, postalCode int32, address string, description string) (models.Account, error)
	// CreateCustomer atomically inserts the account and its customer subtype row.
	CreateCustomer(ctx context.Context, account models.Account, postalCode int32) (models.Account, error)
	// GetAllRecipients returns every farmer and customer as a notification
	// recipient. Admins are not included: a platform-wide notice is addressed
	// to users, not to the operators sending it.
	GetAllRecipients(ctx context.Context) ([]models.Recipient, error)
	// ListAccounts returns one page of every account on the platform, newest
	// first, together with how many accounts match the filter in total. An
	// empty page is an empty page: this never reports ErrNotFound.
	ListAccounts(ctx context.Context, filter models.AccountListFilter) (models.Page[models.AccountListing], error)
	// GetCustomersOfFarmer returns the customers currently renting one of the
	// farmer's plots, each exactly once however many plots they rent. A
	// customer whose rental has ended is not included: the farmer's licence to
	// mail them is the rental itself.
	GetCustomersOfFarmer(ctx context.Context, farmer uuid.UUID) ([]models.Recipient, error)
	// GetCustomersOfFarmerForField narrows GetCustomersOfFarmer to the
	// customers currently renting a plot of one specific field.
	GetCustomersOfFarmerForField(ctx context.Context, field uuid.UUID) ([]models.Recipient, error)
	// GetCustomersOfFarmerForPlot narrows GetCustomersOfFarmer to the
	// customer currently renting one specific plot.
	GetCustomersOfFarmerForPlot(ctx context.Context, plot uuid.UUID) ([]models.Recipient, error)
	// GetCustomersOfFarmerForFieldAndCrop is the audience for a ripeness
	// notice: customers with an active rental on a plot of the given field,
	// growing the given crop.
	GetCustomersOfFarmerForFieldAndCrop(ctx context.Context, field, crop uuid.UUID) ([]models.Recipient, error)
}

// PendingRegistrationRepository stores registrations awaiting email
// verification, separately from AccountRepository: a pending registration is
// not an account and must never be reachable through the account queries
// (login, listing, notification recipients) until it is verified.
type PendingRegistrationRepository interface {
	// UpsertPendingRegistration stores the registration, replacing any
	// existing pending registration for the same email (refreshed data and
	// expiry) rather than erroring — see the ON CONFLICT in the query.
	UpsertPendingRegistration(ctx context.Context, reg models.PendingRegistration, ttl time.Duration) (uuid.UUID, error)
	// GetPendingRegistrationByID returns ErrNotFound if the id does not
	// exist or has expired.
	GetPendingRegistrationByID(ctx context.Context, id uuid.UUID) (models.PendingRegistration, error)
	DeletePendingRegistration(ctx context.Context, id uuid.UUID) error
}

type BroadcastNotificationRepository interface {
	// CreateBroadcastNotification stores one platform-wide notice.
	CreateBroadcastNotification(ctx context.Context, subject, body string) (models.BroadcastNotification, error)
	// GetAllBroadcastNotifications returns every broadcast, newest first.
	GetAllBroadcastNotifications(ctx context.Context) ([]models.BroadcastNotification, error)
}

type AnnouncementRepository interface {
	// CreateAnnouncement stores one notice by a farmer and returns it with the
	// farm name already resolved. field and plot are the optional scope — at
	// most one is non-nil; both nil reaches every current renter.
	CreateAnnouncement(ctx context.Context, farmer uuid.UUID, subject, body string, field, plot *uuid.UUID) (models.AnnouncementWithFarm, error)
	// GetAnnouncementsByFarmer returns the farmer's own notices, newest first.
	GetAnnouncementsByFarmer(ctx context.Context, farmer uuid.UUID) ([]models.AnnouncementWithFarm, error)
	// GetAnnouncementsForCustomer returns the notices of every farmer the
	// customer currently rents from, newest first, each carrying the farm name
	// it came from. A scoped notice is only included if the customer's active
	// rental actually covers that field/plot.
	GetAnnouncementsForCustomer(ctx context.Context, customer uuid.UUID) ([]models.AnnouncementWithFarm, error)
}

type CareInstructionRepository interface {
	// CreateCareInstruction adds one task to a crop's weekly guide: the
	// default guide when farm is nil, that farm's own guide otherwise, which
	// must already have been started with StartFarmCareGuide. Returns
	// ErrNotFound if no crop has that id.
	CreateCareInstruction(ctx context.Context, crop uuid.UUID, farm *uuid.UUID, week int32, title, body string) (models.CareInstruction, error)
	// UpdateCareInstruction rewrites an instruction's week, title and body.
	// Returns ErrNotFound if no instruction has that id.
	UpdateCareInstruction(ctx context.Context, id uuid.UUID, week int32, title, body string) (models.CareInstruction, error)
	// DeleteCareInstruction removes one instruction, reporting ErrNotFound
	// rather than succeeding silently when the id is unknown.
	DeleteCareInstruction(ctx context.Context, id uuid.UUID) error
	// GetCareInstructionByID returns ErrNotFound if no instruction has that id.
	GetCareInstructionByID(ctx context.Context, id uuid.UUID) (models.CareInstruction, error)
	// GetDefaultCareInstructionsByCrop returns one crop's default guide in
	// week order.
	GetDefaultCareInstructionsByCrop(ctx context.Context, crop uuid.UUID) ([]models.CareInstruction, error)
	// StartFarmCareGuide takes the crop's guide over for the farm, copying
	// the default guide as it stands. Does nothing if the farm already has its
	// own guide for the crop, so every farmer write may call it first. Returns
	// ErrNotFound if the crop does not exist.
	StartFarmCareGuide(ctx context.Context, crop, farm uuid.UUID) error
	// GetFarmCopyOfCareInstruction returns the farm's copy of a default
	// instruction, or ErrNotFound if the farm's guide has none.
	GetFarmCopyOfCareInstruction(ctx context.Context, farm, basedOn uuid.UUID) (models.CareInstruction, error)
	// DeleteFarmCareGuide drops the farm's own guide for the crop, so its
	// tenants read the default again. Returns ErrNotFound if the farm has no
	// guide of its own for the crop.
	DeleteFarmCareGuide(ctx context.Context, crop, farm uuid.UUID) error
	// HasFarmCareGuide reports whether the farm has its own guide for the crop.
	HasFarmCareGuide(ctx context.Context, crop, farm uuid.UUID) (bool, error)
	// GetEffectiveCareInstructions returns, for each crop grown on a farm, the
	// guide that farm's tenants read — the farm's own, or else the default —
	// each in week order. A guide with no instructions is absent from the map
	// rather than mapping to an empty slice.
	GetEffectiveCareInstructions(ctx context.Context, guides []models.CropAtFarm) (map[models.CropAtFarm][]models.CareInstruction, error)
}

type RipenessNoticeRepository interface {
	// CreateRipenessNotice stores one notice by a farmer and returns it with
	// the farm, field and crop names already resolved.
	CreateRipenessNotice(ctx context.Context, farmer, field, crop uuid.UUID) (models.RipenessNoticeWithDetails, error)
	// GetRipenessNoticesForCustomer returns the notices for fields the
	// customer currently rents a matching plot on, newest first.
	GetRipenessNoticesForCustomer(ctx context.Context, customer uuid.UUID) ([]models.RipenessNoticeWithDetails, error)
}

type FarmRepository interface {
	// GetFarmByID returns ErrNotFound if no farm has that id.
	GetFarmByID(ctx context.Context, farmID uuid.UUID) (models.Farm, error)
	// GetFarmIDByFarmerID resolves a farmer's own farm id. Returns
	// ErrNotFound if the account is not a farmer.
	GetFarmIDByFarmerID(ctx context.Context, farmerID uuid.UUID) (uuid.UUID, error)
	// ListFarms returns one page of every farm on the platform, together with
	// how many match the filter in total. A farm that owns nothing comes back
	// with zeros rather than being left out.
	ListFarms(ctx context.Context, filter models.FarmListFilter) (models.Page[models.FarmListing], error)
	// GetFarmCropRates returns the farm's crop rates, one row per crop that
	// has been priced.
	GetFarmCropRates(ctx context.Context, farm uuid.UUID) ([]models.FarmCropRate, error)
	// GetFarmCropRate returns one crop's rate on the farm. Returns
	// ErrNotFound if the farm has not priced that crop.
	GetFarmCropRate(ctx context.Context, farm, crop uuid.UUID) (int32, error)
	// SetFarmCropRates fully replaces the farm's crop rates. Returns
	// ErrNotFound if any crop id does not exist.
	SetFarmCropRates(ctx context.Context, farm uuid.UUID, rates []models.FarmCropRate) error
}

type FieldRepository interface {
	CreateField(ctx context.Context, field models.Field) (uuid.UUID, error)
	// GetFieldFarm returns the id of the farm a field belongs to.
	GetFieldFarm(ctx context.Context, id uuid.UUID) (uuid.UUID, error)
	GetFieldsByFarm(ctx context.Context, farm uuid.UUID) ([]models.Field, error)
}

type PlotRepository interface {
	// CreatePlot returns the created plot, including its computed area.
	CreatePlot(ctx context.Context, plot models.Plot) (models.Plot, error)
	GetPlotsByFields(ctx context.Context, fields []uuid.UUID) ([]models.Plot, error)
	// GetNearestPlots returns up to limit plots ordered by distance from the
	// given point (lon, lat), nearest first — only farm's plots when farm is set.
	GetNearestPlots(ctx context.Context, lon, lat float64, farm *uuid.UUID, limit int32) ([]models.NearbyPlot, error)
	// GetPlotField returns the id of the field a plot belongs to. Returns
	// ErrNotFound if the plot does not exist.
	GetPlotField(ctx context.Context, plot uuid.UUID) (uuid.UUID, error)
	// GetPlotByID returns the plot, including its computed area and its own
	// base price. Returns ErrNotFound if the plot does not exist.
	GetPlotByID(ctx context.Context, plot uuid.UUID) (models.Plot, error)
}

type RentalRepository interface {
	// CreateRentalRequest records the customer's request to book the plot
	// starting at startAt and running for durationMonths, in the Requested
	// state. Returns ErrPlotUnavailable if an existing non-declined rental
	// overlaps that period, and ErrNotFound if the plot, crop, or customer
	// does not exist.
	CreateRentalRequest(ctx context.Context, plot, customer, crop uuid.UUID, startAt time.Time, durationMonths int32, message string) (models.Rental, error)
	// UpdateRentalStatus decides a still-Requested rental into status.
	// Returns ErrRentalAlreadyDecided if the rental is not in the Requested
	// state (including if the id does not exist).
	UpdateRentalStatus(ctx context.Context, id uuid.UUID, status models.RentalStatus) (models.Rental, error)
	// GetRentalWithFieldByID returns the rental together with the id of the
	// field its plot belongs to, so callers can check field ownership before
	// deciding it. Returns ErrNotFound if the id does not exist.
	GetRentalWithFieldByID(ctx context.Context, id uuid.UUID) (models.RentalWithField, error)
	// GetRentalsByCustomer returns the customer's rentals, newest first,
	// each with the plot and crop it books.
	GetRentalsByCustomer(ctx context.Context, customer uuid.UUID) ([]models.RentalWithPlot, error)
	// GetActiveRentalsByCustomer returns only the customer's rentals covering
	// right now, each with the plot's and field's names, the crop, and which
	// week of the rental today falls in.
	GetActiveRentalsByCustomer(ctx context.Context, customer uuid.UUID) ([]models.ActiveRental, error)
	// GetRentalsByFarm returns every rental on the farm's own plots,
	// active and historic, newest first, each with its plot, field name, and
	// customer.
	GetRentalsByFarm(ctx context.Context, farm uuid.UUID) ([]models.RentalWithPlotAndCustomer, error)
	// GetRentalByID returns ErrNotFound if the id does not exist.
	GetRentalByID(ctx context.Context, id uuid.UUID) (models.Rental, error)
	// IsPlotAvailable is a fast-fail check for whether the plot is free for
	// the given period. It is advisory only: rental_no_overlap on the rental
	// table remains the actual concurrency guard at insert time.
	IsPlotAvailable(ctx context.Context, plot uuid.UUID, startAt time.Time, durationMonths int32) (bool, error)
}

type CropRepository interface {
	// CreateCrop adds a new crop to the catalog.
	CreateCrop(ctx context.Context, name string, durationMonths int32) (models.Crop, error)
	// DeleteCrop removes a crop from the catalog. Crops referenced by active
	// rentals cannot be removed (the DB enforces the FK).
	DeleteCrop(ctx context.Context, id uuid.UUID) error
	// GetAllCrops returns the full crop catalog, ordered by name.
	GetAllCrops(ctx context.Context) ([]models.Crop, error)
	// GetCropByID returns ErrNotFound if the crop does not exist.
	GetCropByID(ctx context.Context, id uuid.UUID) (models.Crop, error)
	// SetPlotCrops replaces the plot's base price and the set of crops it
	// offers, in one transaction.
	SetPlotCrops(ctx context.Context, plot uuid.UUID, basePriceCentsPerSqmPerWeek int32, crops []uuid.UUID) error
	// GetCropsByPlot returns the crops offered by a single plot, ordered by name.
	GetCropsByPlot(ctx context.Context, plot uuid.UUID) ([]models.Crop, error)
	// GetCropsByPlots returns the crops offered by each of the given plots,
	// keyed by plot id.
	GetCropsByPlots(ctx context.Context, plots []uuid.UUID) (map[uuid.UUID][]models.Crop, error)
	// GetPricedCropOfferingsByPlots returns, for each of the given plots,
	// only the crops that are actually rentable -- both the plot's own base
	// rate and that crop's farm rate are set -- each with its computed
	// total price, keyed by plot id.
	GetPricedCropOfferingsByPlots(ctx context.Context, plots []uuid.UUID) (map[uuid.UUID][]models.PlotCropOffering, error)
}

// RentalCheckoutRepository tracks a Stripe Checkout Session's lifecycle
// from creation until a rental request exists from it, or the payment
// fails to produce one. It deliberately never touches the rental table
// itself -- see RentalRepository.
type RentalCheckoutRepository interface {
	// CreateCheckout records a new Stripe Checkout Session in the Pending
	// state.
	CreateCheckout(ctx context.Context, checkout models.RentalCheckout) (models.RentalCheckout, error)
	// GetCheckoutBySessionID returns ErrNotFound if no checkout has that
	// Stripe session id.
	GetCheckoutBySessionID(ctx context.Context, sessionID string) (models.RentalCheckout, error)
	// CompleteCheckout marks a still-Pending checkout Completed and links
	// the rental created from it. Returns ErrCheckoutAlreadyProcessed if the
	// checkout is not Pending, so a retried webhook delivery cannot
	// double-process it.
	CompleteCheckout(ctx context.Context, id, rental uuid.UUID) (models.RentalCheckout, error)
	// FailCheckout marks a still-Pending checkout Failed. Same idempotency
	// guard as CompleteCheckout.
	FailCheckout(ctx context.Context, id uuid.UUID) (models.RentalCheckout, error)
	// ExpireCheckout marks a still-Pending checkout Expired. Same
	// idempotency guard as CompleteCheckout.
	ExpireCheckout(ctx context.Context, id uuid.UUID) (models.RentalCheckout, error)
	// GetCompletedCheckoutByRental returns the Completed checkout that
	// produced the given rental, so a later farmer decline can be paired
	// back to the payment that must now be refunded. Returns ErrNotFound if
	// the rental has no completed checkout, e.g. it predates this feature.
	GetCompletedCheckoutByRental(ctx context.Context, rental uuid.UUID) (models.RentalCheckout, error)
	// MarkCheckoutRefunded marks a still-Completed checkout Refunded. Same
	// idempotency guard as CompleteCheckout.
	MarkCheckoutRefunded(ctx context.Context, id uuid.UUID) (models.RentalCheckout, error)
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
