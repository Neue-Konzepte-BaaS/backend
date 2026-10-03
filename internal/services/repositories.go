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
	// Subscription-family events. checkout.session.completed/expired for a
	// subscription-mode session share the same Stripe event *type* as the
	// payment-mode ones above -- ParseWebhookEvent tells them apart by the
	// session's mode field and maps to these distinct WebhookEventTypes
	// instead, so callers never need to inspect mode themselves.
	WebhookEventSubscriptionCheckoutCompleted WebhookEventType = "subscription_checkout.session.completed"
	WebhookEventSubscriptionCheckoutExpired   WebhookEventType = "subscription_checkout.session.expired"
	WebhookEventSubscriptionDeleted           WebhookEventType = "customer.subscription.deleted"
	WebhookEventInvoicePaymentFailed          WebhookEventType = "invoice.payment_failed"
	WebhookEventInvoicePaymentSucceeded       WebhookEventType = "invoice.payment_succeeded"
)

// WebhookEvent is the subset of a Stripe webhook event this codebase acts
// on, already reduced from the Stripe SDK's own types so that nothing above
// the repositories package needs to import it.
type WebhookEvent struct {
	Type WebhookEventType
	// CheckoutSessionID is set for the checkout.session.* event family
	// (both payment and subscription mode).
	CheckoutSessionID string
	// StripeSubscriptionID is set for the subscription/invoice event family:
	// customer.subscription.deleted, invoice.payment_failed,
	// invoice.payment_succeeded.
	StripeSubscriptionID string
	// CurrentPeriodEnd is set for invoice.payment_succeeded, the renewed
	// subscription's next billing date.
	CurrentPeriodEnd *time.Time
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

	// CreateSubscriptionPrice creates a new Stripe Product+Price pair with
	// monthly recurring billing, for a plan being created or repriced by an
	// admin. Products/Prices are never mutated once created -- a repriced
	// plan gets a brand new one, see subscription_plan's migration comment.
	CreateSubscriptionPrice(ctx context.Context, displayName string, amountCents int64) (stripePriceID string, err error)
	// CreateSubscriptionCheckoutSession opens a Stripe Embedded Checkout
	// session in subscription mode for the given Stripe Price. If
	// existingStripeCustomerID is nil, Stripe creates a new Customer for
	// customerEmail; otherwise the existing customer is reused so a farmer
	// resubscribing after a cancellation is billed under the same Customer.
	// Returns the session id, client secret, and the Stripe Customer id (new
	// or reused) to persist against the local farmer_subscription row.
	CreateSubscriptionCheckoutSession(ctx context.Context, stripePriceID, customerEmail string, existingStripeCustomerID *string, returnURL string) (sessionID, clientSecret, stripeCustomerID string, err error)
	// UpdateSubscriptionPrice swaps a live Stripe Subscription onto a new
	// Price, prorating the difference for the remainder of the current
	// billing period. Returns the subscription's current_period_end after
	// the change.
	UpdateSubscriptionPrice(ctx context.Context, stripeSubscriptionID, newStripePriceID string) (currentPeriodEnd time.Time, err error)
	// CancelSubscription cancels a live Stripe Subscription immediately, e.g.
	// when the farmer behind it deletes their account.
	CancelSubscription(ctx context.Context, stripeSubscriptionID string) error
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
	// SoftDeleteAccount scrubs the account's personal data and marks it
	// deleted, so it can no longer log in, be messaged, or appear in any
	// audience. Returns ErrNotFound if the id does not exist or is already
	// deleted.
	SoftDeleteAccount(ctx context.Context, id uuid.UUID, firstName, lastName, scrubbedEmail, scrubbedPasswordHash string) error
	// HasActiveRentalAsCustomer reports whether the customer currently has a
	// rental covering right now.
	HasActiveRentalAsCustomer(ctx context.Context, accountID uuid.UUID) (bool, error)
	// HasActiveRentalAsFarmer reports whether any plot on the farmer's farm
	// currently has a rental covering right now.
	HasActiveRentalAsFarmer(ctx context.Context, farmerID uuid.UUID) (bool, error)
	// GetCustomerNotificationPreferences returns the customer's current email
	// notification preference. Returns ErrNotFound if the account id has no
	// customer row.
	GetCustomerNotificationPreferences(ctx context.Context, accountID uuid.UUID) (models.CustomerNotificationPreferences, error)
	// UpdateCustomerNotificationPreferences overwrites the customer's email
	// notification preference. Returns ErrNotFound if the account id has no
	// customer row.
	UpdateCustomerNotificationPreferences(ctx context.Context, accountID uuid.UUID, prefs models.CustomerNotificationPreferences) error
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

// RefreshTokenRepository persists issued refresh tokens so a single session
// can be revoked server-side (rotation, logout, account deletion) without
// waiting for the JWT itself to expire.
type RefreshTokenRepository interface {
	InsertRefreshToken(ctx context.Context, accountID uuid.UUID, tokenHash string, expiresAt time.Time) error
	// GetActiveRefreshTokenByHash returns ErrNotFound if the hash is
	// unknown, expired, or already revoked.
	GetActiveRefreshTokenByHash(ctx context.Context, tokenHash string) (models.RefreshToken, error)
	// RevokeRefreshTokenByHash marks the token revoked. A hash that is
	// unknown or already revoked is not an error -- revocation is
	// idempotent, since both Logout and a failed refresh attempt may try to
	// revoke the same token.
	RevokeRefreshTokenByHash(ctx context.Context, tokenHash string) error
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

type SeasonRepository interface {
	// CreateSeason adds a season to the default set when farm is nil, or that
	// farm's own set otherwise.
	CreateSeason(ctx context.Context, farm *uuid.UUID, name string, startMonth, startDay, endMonth, endDay int32) (models.Season, error)
	// UpdateSeason rewrites a season's name and date range. Returns
	// ErrNotFound if no season has that id.
	UpdateSeason(ctx context.Context, id uuid.UUID, name string, startMonth, startDay, endMonth, endDay int32) (models.Season, error)
	// DeleteSeason removes one season, reporting ErrNotFound rather than
	// succeeding silently when the id is unknown.
	DeleteSeason(ctx context.Context, id uuid.UUID) error
	// GetSeasonByID returns ErrNotFound if no season has that id.
	GetSeasonByID(ctx context.Context, id uuid.UUID) (models.Season, error)
	// GetDefaultSeasons returns the global default set, ordered by start date.
	GetDefaultSeasons(ctx context.Context) ([]models.Season, error)
	// GetFarmSeasons returns the farm's own seasons, ordered by start date.
	GetFarmSeasons(ctx context.Context, farm uuid.UUID) ([]models.Season, error)
	// CreateCropSeasonRule ties a crop to a season: the default rule when
	// farm is nil, that farm's own rule otherwise. Returns
	// ErrCropSeasonRuleExists if a rule already exists for that crop in that
	// set (the defaults, or that farm's own).
	CreateCropSeasonRule(ctx context.Context, crop uuid.UUID, farm *uuid.UUID, season uuid.UUID) (models.CropSeasonRule, error)
	// UpdateCropSeasonRule repoints an existing rule at a different season.
	// Returns ErrNotFound if no rule has that id.
	UpdateCropSeasonRule(ctx context.Context, id uuid.UUID, season uuid.UUID) (models.CropSeasonRule, error)
	// DeleteCropSeasonRule removes one rule, reporting ErrNotFound rather
	// than succeeding silently when the id is unknown.
	DeleteCropSeasonRule(ctx context.Context, id uuid.UUID) error
	// GetCropSeasonRuleByID returns ErrNotFound if no rule has that id.
	GetCropSeasonRuleByID(ctx context.Context, id uuid.UUID) (models.CropSeasonRule, error)
	// GetCropSeasonRuleForCrop returns the rule for a crop in exactly one
	// set: the default rule when farm is nil, that farm's own rule
	// otherwise. Unlike GetEffectiveSeasonForCrop, it never falls back from a
	// farm's set to the defaults. Returns ErrNotFound if no such rule exists.
	GetCropSeasonRuleForCrop(ctx context.Context, crop uuid.UUID, farm *uuid.UUID) (models.CropSeasonRule, error)
	// GetEffectiveSeasonForCrop returns the season a crop is checked against
	// for a given farm: that farm's own rule if it has one, the default rule
	// otherwise. The bool is false if neither exists, meaning the crop is
	// unrestricted for that farm.
	GetEffectiveSeasonForCrop(ctx context.Context, crop, farm uuid.UUID) (models.Season, bool, error)
	// GetEffectiveSeasonsForCrops is GetEffectiveSeasonForCrop batched over
	// several (crop, farm) pairs. A pair absent from the returned map has no
	// rule and is unrestricted.
	GetEffectiveSeasonsForCrops(ctx context.Context, pairs []models.CropAtFarm) (map[models.CropAtFarm]models.Season, error)
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
	// UpdateFarmByFarmer overwrites the editable fields of the farmer's own
	// farm and returns its id. Returns ErrNotFound if the account owns no farm.
	UpdateFarmByFarmer(ctx context.Context, farmerID uuid.UUID, update models.FarmUpdate) (uuid.UUID, error)
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
	// GetFieldsByFarmWithAvailablePlotStats is the public, customer-facing
	// counterpart to GetFieldsByFarm: each field's plot count/area only
	// counts plots available to rent right now — see the query's own doc
	// comment for why that must match GetNearestPlots's own predicate.
	GetFieldsByFarmWithAvailablePlotStats(ctx context.Context, farm uuid.UUID) ([]models.FieldWithPlotStats, error)
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
	// CountPlotsByFarm returns how many plots the farm currently offers,
	// regardless of rental status. Used by the subscription plot-count cap.
	CountPlotsByFarm(ctx context.Context, farm uuid.UUID) (int64, error)
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
	CreateCrop(ctx context.Context, nameDe, nameEn string, durationMonths int32) (models.Crop, error)
	// UpdateCrop overwrites a crop's editable fields. Returns ErrNotFound if
	// the crop does not exist.
	UpdateCrop(ctx context.Context, id uuid.UUID, nameDe, nameEn string, durationMonths int32) (models.Crop, error)
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

// SubscriptionPlanRepository manages the fixed catalog of subscription
// tiers a farmer may choose from.
type SubscriptionPlanRepository interface {
	// CreateSubscriptionPlan seeds one of the three fixed tiers. Used only
	// at startup (see seedSubscriptionPlans in main.go) -- there is no
	// route to create an arbitrary new tier at runtime.
	CreateSubscriptionPlan(ctx context.Context, code models.SubscriptionPlanCode, displayName string, maxPlots *int32, priceCents int32, stripePriceID string) error
	// GetActiveSubscriptionPlans returns the plans currently open to new
	// subscriptions, cheapest first.
	GetActiveSubscriptionPlans(ctx context.Context) ([]models.SubscriptionPlan, error)
	GetSubscriptionPlanByID(ctx context.Context, id uuid.UUID) (models.SubscriptionPlan, error)
	GetSubscriptionPlanByCode(ctx context.Context, code models.SubscriptionPlanCode) (models.SubscriptionPlan, error)
	// ListSubscriptionPlans is the admin view: every plan, active or
	// retired.
	ListSubscriptionPlans(ctx context.Context) ([]models.SubscriptionPlan, error)
	// UpdateSubscriptionPlanPrice repoints a plan at a newly created Stripe
	// Price. Existing subscribers keep paying whatever Price their own
	// subscription already references.
	UpdateSubscriptionPlanPrice(ctx context.Context, id uuid.UUID, priceCents int32, stripePriceID string) (models.SubscriptionPlan, error)
	// SetSubscriptionPlanActive retires or reactivates a tier without
	// deleting it, since existing subscriptions still reference it.
	SetSubscriptionPlanActive(ctx context.Context, id uuid.UUID, active bool) error
}

// FarmerSubscriptionRepository tracks a farmer's subscription lifecycle, in
// the same relationship to Stripe's own subscription object that
// RentalCheckoutRepository has to a rental: this is the ledger, Stripe is
// the source of truth for billing state, kept in sync via webhook.
type FarmerSubscriptionRepository interface {
	// CreateSubscription records a new Stripe subscription Checkout Session
	// in the Pending state.
	CreateSubscription(ctx context.Context, sub models.FarmerSubscription) (models.FarmerSubscription, error)
	GetSubscriptionBySessionID(ctx context.Context, sessionID string) (models.FarmerSubscription, error)
	GetSubscriptionByStripeSubscriptionID(ctx context.Context, stripeSubscriptionID string) (models.FarmerSubscription, error)
	// GetActiveSubscriptionByFarmer returns the farmer's current
	// active-or-past_due subscription. Returns ErrNotFound if the farmer has
	// never subscribed, or their only subscription is pending or canceled.
	GetActiveSubscriptionByFarmer(ctx context.Context, farmer uuid.UUID) (models.FarmerSubscription, error)
	// GetSubscriptionByFarmer returns the farmer's current non-terminal
	// subscription regardless of status, including Pending, so a farmer
	// mid-checkout can poll their own status.
	GetSubscriptionByFarmer(ctx context.Context, farmer uuid.UUID) (models.FarmerSubscription, error)
	// ActivateSubscription marks a still-Pending subscription Active.
	// Returns ErrCheckoutAlreadyProcessed if it is not Pending, so a
	// retried webhook delivery cannot double-process it.
	ActivateSubscription(ctx context.Context, id uuid.UUID, stripeSubscriptionID string, currentPeriodEnd time.Time) (models.FarmerSubscription, error)
	// ExpireSubscription marks a still-Pending subscription Canceled (its
	// checkout session expired unpaid). Same idempotency guard as
	// ActivateSubscription.
	ExpireSubscription(ctx context.Context, id uuid.UUID) (models.FarmerSubscription, error)
	// MarkSubscriptionPastDue marks a still-Active subscription PastDue.
	MarkSubscriptionPastDue(ctx context.Context, stripeSubscriptionID string) (models.FarmerSubscription, error)
	// ReactivateSubscription marks an Active-or-PastDue subscription Active
	// and refreshes its current period end, covering both a routine renewal
	// and recovery from PastDue.
	ReactivateSubscription(ctx context.Context, stripeSubscriptionID string, currentPeriodEnd time.Time) (models.FarmerSubscription, error)
	// CancelSubscription marks a non-terminal subscription Canceled.
	CancelSubscription(ctx context.Context, stripeSubscriptionID string) (models.FarmerSubscription, error)
	// UpdateSubscriptionPlan repoints an Active-or-PastDue subscription at a
	// new plan and refreshes its current period end from Stripe's proration
	// response. Returns ErrNotFound if the subscription is not Active or
	// PastDue.
	UpdateSubscriptionPlan(ctx context.Context, id, plan uuid.UUID, currentPeriodEnd time.Time) (models.FarmerSubscription, error)
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
