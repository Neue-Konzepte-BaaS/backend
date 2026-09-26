package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// fakePaymentRentalService is a RentalService stub that only implements
// RequestRental, the one method PaymentService calls.
type fakePaymentRentalService struct {
	rental       models.Rental
	err          error
	calledStart  time.Time
	calledPlot   uuid.UUID
	calledCrop   uuid.UUID
	calledCustom uuid.UUID
	calls        int
}

func (f *fakePaymentRentalService) RequestRental(_ context.Context, customer, plot, crop uuid.UUID, startAt time.Time, _ string) (models.Rental, error) {
	f.calls++
	f.calledCustom, f.calledPlot, f.calledCrop, f.calledStart = customer, plot, crop, startAt
	if f.err != nil {
		return models.Rental{}, f.err
	}
	return f.rental, nil
}
func (f *fakePaymentRentalService) ApproveRental(context.Context, uuid.UUID, uuid.UUID) (models.Rental, error) {
	return models.Rental{}, nil
}
func (f *fakePaymentRentalService) DeclineRental(context.Context, uuid.UUID, uuid.UUID) (models.Rental, error) {
	return models.Rental{}, nil
}
func (f *fakePaymentRentalService) GetRentals(context.Context, uuid.UUID) ([]models.RentalWithPlot, error) {
	return nil, nil
}
func (f *fakePaymentRentalService) GetRentalsForFarmer(context.Context, uuid.UUID) ([]models.RentalWithPlotAndCustomer, error) {
	return nil, nil
}

// fakePaymentRentalRepo is a RentalRepository stub for what PaymentService
// needs: IsPlotAvailable and GetRentalByID.
type fakePaymentRentalRepo struct {
	available   bool
	availableEr error
	rental      models.Rental
	rentalErr   error
}

func (f *fakePaymentRentalRepo) CreateRentalRequest(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, time.Time, int32, string) (models.Rental, error) {
	return models.Rental{}, nil
}
func (f *fakePaymentRentalRepo) UpdateRentalStatus(context.Context, uuid.UUID, models.RentalStatus) (models.Rental, error) {
	return models.Rental{}, nil
}
func (f *fakePaymentRentalRepo) GetRentalWithFieldByID(context.Context, uuid.UUID) (models.RentalWithField, error) {
	return models.RentalWithField{}, nil
}
func (f *fakePaymentRentalRepo) GetRentalsByCustomer(context.Context, uuid.UUID) ([]models.RentalWithPlot, error) {
	return nil, nil
}
func (f *fakePaymentRentalRepo) GetRentalsByFarm(context.Context, uuid.UUID) ([]models.RentalWithPlotAndCustomer, error) {
	return nil, nil
}
func (f *fakePaymentRentalRepo) GetRentalByID(context.Context, uuid.UUID) (models.Rental, error) {
	if f.rentalErr != nil {
		return models.Rental{}, f.rentalErr
	}
	return f.rental, nil
}
func (f *fakePaymentRentalRepo) IsPlotAvailable(context.Context, uuid.UUID, time.Time, int32) (bool, error) {
	return f.available, f.availableEr
}
func (f *fakePaymentRentalRepo) GetActiveRentalsByCustomer(context.Context, uuid.UUID) ([]models.ActiveRental, error) {
	return nil, nil
}

// fakeCheckoutRepo is an in-memory RentalCheckoutRepository.
type fakeCheckoutRepo struct {
	bySession map[string]models.RentalCheckout
	byRental  map[uuid.UUID]models.RentalCheckout
	created   models.RentalCheckout
	createErr error

	completeCalls int
	failCalls     int
	expireCalls   int
	refundCalls   int
	stateErr      error // returned by Complete/Fail/Expire/MarkRefunded to simulate ErrCheckoutAlreadyProcessed
}

func (f *fakeCheckoutRepo) CreateCheckout(_ context.Context, checkout models.RentalCheckout) (models.RentalCheckout, error) {
	if f.createErr != nil {
		return models.RentalCheckout{}, f.createErr
	}
	f.created = checkout
	return checkout, nil
}
func (f *fakeCheckoutRepo) GetCheckoutBySessionID(_ context.Context, sessionID string) (models.RentalCheckout, error) {
	if c, ok := f.bySession[sessionID]; ok {
		return c, nil
	}
	return models.RentalCheckout{}, ErrNotFound
}
func (f *fakeCheckoutRepo) CompleteCheckout(_ context.Context, id, rental uuid.UUID) (models.RentalCheckout, error) {
	f.completeCalls++
	if f.stateErr != nil {
		return models.RentalCheckout{}, f.stateErr
	}
	c := f.bySession[sessionIDFor(f, id)]
	c.Status = models.CheckoutStatusCompleted
	c.Rental = &rental
	return c, nil
}
func (f *fakeCheckoutRepo) FailCheckout(_ context.Context, id uuid.UUID) (models.RentalCheckout, error) {
	f.failCalls++
	if f.stateErr != nil {
		return models.RentalCheckout{}, f.stateErr
	}
	c := f.bySession[sessionIDFor(f, id)]
	c.Status = models.CheckoutStatusFailed
	return c, nil
}
func (f *fakeCheckoutRepo) ExpireCheckout(_ context.Context, id uuid.UUID) (models.RentalCheckout, error) {
	f.expireCalls++
	if f.stateErr != nil {
		return models.RentalCheckout{}, f.stateErr
	}
	c := f.bySession[sessionIDFor(f, id)]
	c.Status = models.CheckoutStatusExpired
	return c, nil
}
func (f *fakeCheckoutRepo) GetCompletedCheckoutByRental(_ context.Context, rental uuid.UUID) (models.RentalCheckout, error) {
	if c, ok := f.byRental[rental]; ok && c.Status == models.CheckoutStatusCompleted {
		return c, nil
	}
	return models.RentalCheckout{}, ErrNotFound
}
func (f *fakeCheckoutRepo) MarkCheckoutRefunded(_ context.Context, id uuid.UUID) (models.RentalCheckout, error) {
	f.refundCalls++
	if f.stateErr != nil {
		return models.RentalCheckout{}, f.stateErr
	}
	return models.RentalCheckout{ID: id, Status: models.CheckoutStatusRefunded}, nil
}

// sessionIDFor is a small helper so the fake's Complete/Fail/Expire methods
// can find the row they were called on by id, since the fake is keyed by
// session id rather than by id.
func sessionIDFor(f *fakeCheckoutRepo, id uuid.UUID) string {
	for session, c := range f.bySession {
		if c.ID == id {
			return session
		}
	}
	return ""
}

// fakePaymentGateway is an in-memory PaymentGateway.
type fakePaymentGateway struct {
	sessionID, clientSecret string
	createErr               error
	refundErr               error
	refundedSessions        []string
	event                   WebhookEvent
	eventOK                 bool
	parseErr                error
}

func (f *fakePaymentGateway) CreateCheckoutSession(context.Context, int64, string, string) (string, string, error) {
	if f.createErr != nil {
		return "", "", f.createErr
	}
	return f.sessionID, f.clientSecret, nil
}
func (f *fakePaymentGateway) RefundCheckoutSession(_ context.Context, sessionID string) error {
	f.refundedSessions = append(f.refundedSessions, sessionID)
	return f.refundErr
}
func (f *fakePaymentGateway) ParseWebhookEvent([]byte, string) (WebhookEvent, bool, error) {
	if f.parseErr != nil {
		return WebhookEvent{}, false, f.parseErr
	}
	return f.event, f.eventOK, nil
}

// fakePaymentPlotRepo, fakePaymentCropRepo, fakePaymentFieldRepo and
// fakePaymentFarmRepo each implement only what CreateCheckoutSession needs;
// every other method is an unused stub.
type fakePaymentPlotRepo struct {
	plot    models.Plot
	plotErr error
}

func (f *fakePaymentPlotRepo) CreatePlot(context.Context, models.Plot) (models.Plot, error) {
	return models.Plot{}, nil
}
func (f *fakePaymentPlotRepo) GetPlotsByFields(context.Context, []uuid.UUID) ([]models.Plot, error) {
	return nil, nil
}
func (f *fakePaymentPlotRepo) GetNearestPlots(context.Context, float64, float64, *uuid.UUID, int32) ([]models.NearbyPlot, error) {
	return nil, nil
}
func (f *fakePaymentPlotRepo) GetPlotField(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.UUID{}, nil
}
func (f *fakePaymentPlotRepo) GetPlotByID(context.Context, uuid.UUID) (models.Plot, error) {
	if f.plotErr != nil {
		return models.Plot{}, f.plotErr
	}
	return f.plot, nil
}

type fakePaymentCropRepo struct {
	crop           models.Crop
	cropErr        error
	offeredCropIDs []uuid.UUID
}

func (f *fakePaymentCropRepo) CreateCrop(context.Context, string, int32) (models.Crop, error) {
	return models.Crop{}, nil
}
func (f *fakePaymentCropRepo) DeleteCrop(context.Context, uuid.UUID) error { return nil }
func (f *fakePaymentCropRepo) GetAllCrops(context.Context) ([]models.Crop, error) {
	return nil, nil
}
func (f *fakePaymentCropRepo) GetCropByID(_ context.Context, _ uuid.UUID) (models.Crop, error) {
	if f.cropErr != nil {
		return models.Crop{}, f.cropErr
	}
	return f.crop, nil
}
func (f *fakePaymentCropRepo) SetPlotCrops(context.Context, uuid.UUID, int32, []uuid.UUID) error {
	return nil
}
func (f *fakePaymentCropRepo) GetCropsByPlot(_ context.Context, _ uuid.UUID) ([]models.Crop, error) {
	crops := make([]models.Crop, len(f.offeredCropIDs))
	for i, id := range f.offeredCropIDs {
		crops[i] = models.Crop{ID: id}
	}
	return crops, nil
}
func (f *fakePaymentCropRepo) GetCropsByPlots(context.Context, []uuid.UUID) (map[uuid.UUID][]models.Crop, error) {
	return nil, nil
}
func (f *fakePaymentCropRepo) GetPricedCropOfferingsByPlots(context.Context, []uuid.UUID) (map[uuid.UUID][]models.PlotCropOffering, error) {
	return nil, nil
}

type fakePaymentFieldRepo struct {
	farm uuid.UUID
}

func (f *fakePaymentFieldRepo) CreateField(context.Context, models.Field) (uuid.UUID, error) {
	return uuid.UUID{}, nil
}
func (f *fakePaymentFieldRepo) GetFieldFarm(context.Context, uuid.UUID) (uuid.UUID, error) {
	return f.farm, nil
}
func (f *fakePaymentFieldRepo) GetFieldsByFarm(context.Context, uuid.UUID) ([]models.Field, error) {
	return nil, nil
}

type fakePaymentFarmRepo struct {
	rate    int32
	rateErr error
}

func (f *fakePaymentFarmRepo) GetFarmByID(context.Context, uuid.UUID) (models.Farm, error) {
	return models.Farm{}, nil
}
func (f *fakePaymentFarmRepo) GetFarmIDByFarmerID(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.UUID{}, nil
}
func (f *fakePaymentFarmRepo) ListFarms(context.Context, models.FarmListFilter) (models.Page[models.FarmListing], error) {
	return models.Page[models.FarmListing]{}, nil
}
func (f *fakePaymentFarmRepo) GetFarmCropRates(context.Context, uuid.UUID) ([]models.FarmCropRate, error) {
	return nil, nil
}
func (f *fakePaymentFarmRepo) GetFarmCropRate(context.Context, uuid.UUID, uuid.UUID) (int32, error) {
	if f.rateErr != nil {
		return 0, f.rateErr
	}
	return f.rate, nil
}
func (f *fakePaymentFarmRepo) SetFarmCropRates(context.Context, uuid.UUID, []models.FarmCropRate) error {
	return nil
}

func newTestPaymentService(rentalService *fakePaymentRentalService, rentalRepo *fakePaymentRentalRepo, checkoutRepo *fakeCheckoutRepo, gateway *fakePaymentGateway, plotRepo *fakePaymentPlotRepo, cropRepo *fakePaymentCropRepo, fieldRepo *fakePaymentFieldRepo, farmRepo *fakePaymentFarmRepo) PaymentService {
	return NewPaymentService(rentalService, rentalRepo, checkoutRepo, gateway, plotRepo, cropRepo, fieldRepo, farmRepo, "https://frontend.example.com")
}

func validCheckoutInputs() (plotID, cropID uuid.UUID, startAt time.Time, message string) {
	return uuid.New(), uuid.New(), time.Now().Add(10 * 24 * time.Hour), "please and thank you"
}

func TestCreateCheckoutSession_ComputesPriceAndRecordsAPendingCheckout(t *testing.T) {
	plotID, cropID, startAt, message := validCheckoutInputs()
	plot := models.Plot{ID: plotID, Name: "North Field Plot 3", AreaSquareMeters: 10, BasePriceCentsPerSqmPerWeek: int32Ptr(10)}
	crop := models.Crop{ID: cropID, Name: "Tomatoes", DurationMonths: 12} // exactly 52 weeks

	checkoutRepo := &fakeCheckoutRepo{bySession: map[string]models.RentalCheckout{}}
	gateway := &fakePaymentGateway{sessionID: "cs_test_123", clientSecret: "secret_123"}
	svc := newTestPaymentService(
		&fakePaymentRentalService{},
		&fakePaymentRentalRepo{available: true},
		checkoutRepo,
		gateway,
		&fakePaymentPlotRepo{plot: plot},
		&fakePaymentCropRepo{crop: crop, offeredCropIDs: []uuid.UUID{cropID}},
		&fakePaymentFieldRepo{},
		&fakePaymentFarmRepo{rate: 5},
	)

	result, err := svc.CreateCheckoutSession(context.Background(), uuid.New(), plotID, cropID, startAt, message)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantPrice := ComputeRentalPriceCents(10, 5, 10, 12)
	if result.PriceCents != wantPrice {
		t.Errorf("priceCents = %d, want %d", result.PriceCents, wantPrice)
	}
	if result.ClientSecret != "secret_123" || result.PlotName != plot.Name || result.CropName != crop.Name {
		t.Errorf("unexpected result: %+v", result)
	}
	if checkoutRepo.created.StripeCheckoutSessionID != "cs_test_123" {
		t.Errorf("checkout row was not recorded against the stripe session")
	}
	if checkoutRepo.created.AmountCents != wantPrice {
		t.Errorf("recorded amountCents = %d, want %d", checkoutRepo.created.AmountCents, wantPrice)
	}
}

func TestCreateCheckoutSession_CropNotOfferedNeverReachesStripe(t *testing.T) {
	plotID, cropID, startAt, message := validCheckoutInputs()
	gateway := &fakePaymentGateway{sessionID: "cs_test_123"}
	svc := newTestPaymentService(
		&fakePaymentRentalService{},
		&fakePaymentRentalRepo{available: true},
		&fakeCheckoutRepo{bySession: map[string]models.RentalCheckout{}},
		gateway,
		&fakePaymentPlotRepo{plot: models.Plot{ID: plotID, BasePriceCentsPerSqmPerWeek: int32Ptr(10)}},
		&fakePaymentCropRepo{crop: models.Crop{ID: cropID, DurationMonths: 1}, offeredCropIDs: nil}, // not offered
		&fakePaymentFieldRepo{},
		&fakePaymentFarmRepo{rate: 5},
	)

	_, err := svc.CreateCheckoutSession(context.Background(), uuid.New(), plotID, cropID, startAt, message)
	if !errors.Is(err, ErrCropNotOffered) {
		t.Fatalf("error = %v, want ErrCropNotOffered", err)
	}
}

func TestCreateCheckoutSession_MissingFarmRateIsTreatedAsNotOffered(t *testing.T) {
	// A plot can list a crop (plot_crop row exists) before the farm has set
	// its rate for it -- from a customer's perspective that is not a real
	// offer, so it must reuse ErrCropNotOffered, not a distinct sentinel.
	plotID, cropID, startAt, message := validCheckoutInputs()
	svc := newTestPaymentService(
		&fakePaymentRentalService{},
		&fakePaymentRentalRepo{available: true},
		&fakeCheckoutRepo{bySession: map[string]models.RentalCheckout{}},
		&fakePaymentGateway{},
		&fakePaymentPlotRepo{plot: models.Plot{ID: plotID, BasePriceCentsPerSqmPerWeek: int32Ptr(10)}},
		&fakePaymentCropRepo{crop: models.Crop{ID: cropID, DurationMonths: 1}, offeredCropIDs: []uuid.UUID{cropID}},
		&fakePaymentFieldRepo{},
		&fakePaymentFarmRepo{rateErr: ErrNotFound},
	)

	_, err := svc.CreateCheckoutSession(context.Background(), uuid.New(), plotID, cropID, startAt, message)
	if !errors.Is(err, ErrCropNotOffered) {
		t.Fatalf("error = %v, want ErrCropNotOffered", err)
	}
}

func TestCreateCheckoutSession_UnavailablePlotNeverReachesStripe(t *testing.T) {
	plotID, cropID, startAt, message := validCheckoutInputs()
	gateway := &fakePaymentGateway{sessionID: "cs_test_123"}
	svc := newTestPaymentService(
		&fakePaymentRentalService{},
		&fakePaymentRentalRepo{available: false},
		&fakeCheckoutRepo{bySession: map[string]models.RentalCheckout{}},
		gateway,
		&fakePaymentPlotRepo{plot: models.Plot{ID: plotID, BasePriceCentsPerSqmPerWeek: int32Ptr(10)}},
		&fakePaymentCropRepo{crop: models.Crop{ID: cropID, DurationMonths: 1}, offeredCropIDs: []uuid.UUID{cropID}},
		&fakePaymentFieldRepo{},
		&fakePaymentFarmRepo{rate: 5},
	)

	_, err := svc.CreateCheckoutSession(context.Background(), uuid.New(), plotID, cropID, startAt, message)
	if !errors.Is(err, ErrPlotUnavailable) {
		t.Fatalf("error = %v, want ErrPlotUnavailable", err)
	}
	if len(gateway.refundedSessions) != 0 {
		t.Errorf("stripe should never have been called")
	}
}

func TestHandleWebhookEvent_CompletedSessionCreatesTheRental(t *testing.T) {
	checkoutID := uuid.New()
	rentalID := uuid.New()
	checkout := models.RentalCheckout{ID: checkoutID, StripeCheckoutSessionID: "cs_1", Status: models.CheckoutStatusPending, Plot: uuid.New(), Crop: uuid.New(), Customer: uuid.New(), StartAt: time.Now()}
	checkoutRepo := &fakeCheckoutRepo{bySession: map[string]models.RentalCheckout{"cs_1": checkout}}
	rentalService := &fakePaymentRentalService{rental: models.Rental{ID: rentalID}}
	gateway := &fakePaymentGateway{event: WebhookEvent{Type: WebhookEventCheckoutCompleted, CheckoutSessionID: "cs_1"}, eventOK: true}

	svc := newTestPaymentService(rentalService, &fakePaymentRentalRepo{}, checkoutRepo, gateway, &fakePaymentPlotRepo{}, &fakePaymentCropRepo{}, &fakePaymentFieldRepo{}, &fakePaymentFarmRepo{})

	if err := svc.HandleWebhookEvent(context.Background(), []byte("{}"), "sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rentalService.calls != 1 {
		t.Errorf("RequestRental calls = %d, want 1", rentalService.calls)
	}
	if checkoutRepo.completeCalls != 1 {
		t.Errorf("CompleteCheckout calls = %d, want 1", checkoutRepo.completeCalls)
	}
	if len(gateway.refundedSessions) != 0 {
		t.Errorf("a successful booking must not be refunded")
	}
}

func TestHandleWebhookEvent_AlreadyCompletedSessionIsANoop(t *testing.T) {
	// Stripe retries webhook delivery: a second delivery for a session that
	// already produced a rental must not create a second one.
	checkoutID := uuid.New()
	rentalID := uuid.New()
	checkout := models.RentalCheckout{ID: checkoutID, StripeCheckoutSessionID: "cs_1", Status: models.CheckoutStatusCompleted, Rental: &rentalID}
	checkoutRepo := &fakeCheckoutRepo{bySession: map[string]models.RentalCheckout{"cs_1": checkout}}
	rentalService := &fakePaymentRentalService{}
	gateway := &fakePaymentGateway{event: WebhookEvent{Type: WebhookEventCheckoutCompleted, CheckoutSessionID: "cs_1"}, eventOK: true}

	svc := newTestPaymentService(rentalService, &fakePaymentRentalRepo{}, checkoutRepo, gateway, &fakePaymentPlotRepo{}, &fakePaymentCropRepo{}, &fakePaymentFieldRepo{}, &fakePaymentFarmRepo{})

	if err := svc.HandleWebhookEvent(context.Background(), []byte("{}"), "sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rentalService.calls != 0 {
		t.Errorf("RequestRental should not have been called again, got %d calls", rentalService.calls)
	}
	if checkoutRepo.completeCalls != 0 {
		t.Errorf("CompleteCheckout should not have been called again")
	}
}

func TestHandleWebhookEvent_LostAvailabilityRaceRefundsAndFails(t *testing.T) {
	checkout := models.RentalCheckout{ID: uuid.New(), StripeCheckoutSessionID: "cs_1", Status: models.CheckoutStatusPending}
	checkoutRepo := &fakeCheckoutRepo{bySession: map[string]models.RentalCheckout{"cs_1": checkout}}
	rentalService := &fakePaymentRentalService{err: ErrPlotUnavailable}
	gateway := &fakePaymentGateway{event: WebhookEvent{Type: WebhookEventCheckoutCompleted, CheckoutSessionID: "cs_1"}, eventOK: true}

	svc := newTestPaymentService(rentalService, &fakePaymentRentalRepo{}, checkoutRepo, gateway, &fakePaymentPlotRepo{}, &fakePaymentCropRepo{}, &fakePaymentFieldRepo{}, &fakePaymentFarmRepo{})

	if err := svc.HandleWebhookEvent(context.Background(), []byte("{}"), "sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gateway.refundedSessions) != 1 || gateway.refundedSessions[0] != "cs_1" {
		t.Errorf("refundedSessions = %v, want [cs_1]", gateway.refundedSessions)
	}
	if checkoutRepo.failCalls != 1 {
		t.Errorf("FailCheckout calls = %d, want 1", checkoutRepo.failCalls)
	}
	if checkoutRepo.completeCalls != 0 {
		t.Errorf("CompleteCheckout should not have been called")
	}
}

func TestHandleWebhookEvent_ExpiredSessionMarksItExpired(t *testing.T) {
	checkout := models.RentalCheckout{ID: uuid.New(), StripeCheckoutSessionID: "cs_1", Status: models.CheckoutStatusPending}
	checkoutRepo := &fakeCheckoutRepo{bySession: map[string]models.RentalCheckout{"cs_1": checkout}}
	gateway := &fakePaymentGateway{event: WebhookEvent{Type: WebhookEventCheckoutExpired, CheckoutSessionID: "cs_1"}, eventOK: true}

	svc := newTestPaymentService(&fakePaymentRentalService{}, &fakePaymentRentalRepo{}, checkoutRepo, gateway, &fakePaymentPlotRepo{}, &fakePaymentCropRepo{}, &fakePaymentFieldRepo{}, &fakePaymentFarmRepo{})

	if err := svc.HandleWebhookEvent(context.Background(), []byte("{}"), "sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if checkoutRepo.expireCalls != 1 {
		t.Errorf("ExpireCheckout calls = %d, want 1", checkoutRepo.expireCalls)
	}
}

func TestHandleWebhookEvent_UnhandledEventTypeIsANoop(t *testing.T) {
	checkoutRepo := &fakeCheckoutRepo{bySession: map[string]models.RentalCheckout{}}
	gateway := &fakePaymentGateway{eventOK: false} // ParseWebhookEvent reports it does not act on this one

	svc := newTestPaymentService(&fakePaymentRentalService{}, &fakePaymentRentalRepo{}, checkoutRepo, gateway, &fakePaymentPlotRepo{}, &fakePaymentCropRepo{}, &fakePaymentFieldRepo{}, &fakePaymentFarmRepo{})

	if err := svc.HandleWebhookEvent(context.Background(), []byte("{}"), "sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleWebhookEvent_InvalidSignaturePropagates(t *testing.T) {
	gateway := &fakePaymentGateway{parseErr: ErrInvalidWebhookSignature}
	svc := newTestPaymentService(&fakePaymentRentalService{}, &fakePaymentRentalRepo{}, &fakeCheckoutRepo{}, gateway, &fakePaymentPlotRepo{}, &fakePaymentCropRepo{}, &fakePaymentFieldRepo{}, &fakePaymentFarmRepo{})

	err := svc.HandleWebhookEvent(context.Background(), []byte("{}"), "bad-sig")
	if !errors.Is(err, ErrInvalidWebhookSignature) {
		t.Fatalf("error = %v, want ErrInvalidWebhookSignature", err)
	}
}

func TestRefundIfPaid_NoPaymentIsANoop(t *testing.T) {
	checkoutRepo := &fakeCheckoutRepo{byRental: map[uuid.UUID]models.RentalCheckout{}}
	gateway := &fakePaymentGateway{}
	svc := newTestPaymentService(&fakePaymentRentalService{}, &fakePaymentRentalRepo{}, checkoutRepo, gateway, &fakePaymentPlotRepo{}, &fakePaymentCropRepo{}, &fakePaymentFieldRepo{}, &fakePaymentFarmRepo{})

	if err := svc.RefundIfPaid(context.Background(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gateway.refundedSessions) != 0 {
		t.Errorf("stripe should not have been called for an unpaid rental")
	}
}

func TestRefundIfPaid_RefundsAndMarksTheCheckoutRow(t *testing.T) {
	rentalID := uuid.New()
	checkout := models.RentalCheckout{ID: uuid.New(), StripeCheckoutSessionID: "cs_1", Status: models.CheckoutStatusCompleted, Rental: &rentalID}
	checkoutRepo := &fakeCheckoutRepo{byRental: map[uuid.UUID]models.RentalCheckout{rentalID: checkout}}
	gateway := &fakePaymentGateway{}
	svc := newTestPaymentService(&fakePaymentRentalService{}, &fakePaymentRentalRepo{}, checkoutRepo, gateway, &fakePaymentPlotRepo{}, &fakePaymentCropRepo{}, &fakePaymentFieldRepo{}, &fakePaymentFarmRepo{})

	if err := svc.RefundIfPaid(context.Background(), rentalID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gateway.refundedSessions) != 1 || gateway.refundedSessions[0] != "cs_1" {
		t.Errorf("refundedSessions = %v, want [cs_1]", gateway.refundedSessions)
	}
	if checkoutRepo.refundCalls != 1 {
		t.Errorf("MarkCheckoutRefunded calls = %d, want 1", checkoutRepo.refundCalls)
	}
}

func int32Ptr(v int32) *int32 { return &v }
