package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// fakeAnnouncementRepo records what was stored and can fail on demand.
type fakeAnnouncementRepo struct {
	created      []models.AnnouncementWithFarm
	byFarmer     []models.AnnouncementWithFarm
	forCustomer  []models.AnnouncementWithFarm
	createErr    error
	listErr      error
	farmName     string
	createdCalls int
	lastField    *uuid.UUID
	lastPlot     *uuid.UUID
}

func (f *fakeAnnouncementRepo) CreateAnnouncement(_ context.Context, farmer uuid.UUID, subject, body string, field, plot *uuid.UUID) (models.AnnouncementWithFarm, error) {
	f.createdCalls++
	f.lastField, f.lastPlot = field, plot
	if f.createErr != nil {
		return models.AnnouncementWithFarm{}, f.createErr
	}
	announcement := models.AnnouncementWithFarm{
		Announcement: models.Announcement{
			ID:        uuid.New(),
			Farmer:    farmer,
			Subject:   subject,
			Body:      body,
			CreatedAt: time.Now(),
			Field:     field,
			Plot:      plot,
		},
		FarmName: f.farmName,
	}
	f.created = append(f.created, announcement)
	return announcement, nil
}

func (f *fakeAnnouncementRepo) GetAnnouncementsByFarmer(context.Context, uuid.UUID) ([]models.AnnouncementWithFarm, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.byFarmer, nil
}

func (f *fakeAnnouncementRepo) GetAnnouncementsForCustomer(context.Context, uuid.UUID) ([]models.AnnouncementWithFarm, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.forCustomer, nil
}

// fakeFieldRepo is an in-memory FieldRepository for exercising ownership
// checks without a database. farmByField maps a field id to the farm it
// belongs to; a field missing from the map behaves as not found.
type fakeFieldRepo struct {
	farmByField map[uuid.UUID]uuid.UUID
}

func (f *fakeFieldRepo) CreateField(context.Context, models.Field) (uuid.UUID, error) {
	panic("not used by these tests")
}

func (f *fakeFieldRepo) GetFieldFarm(_ context.Context, id uuid.UUID) (uuid.UUID, error) {
	farm, ok := f.farmByField[id]
	if !ok {
		return uuid.UUID{}, ErrNotFound
	}
	return farm, nil
}

func (f *fakeFieldRepo) GetFieldsByFarm(context.Context, uuid.UUID) ([]models.Field, error) {
	panic("not used by these tests")
}

// fakePlotRepo is an in-memory PlotRepository for exercising ownership checks
// without a database. fieldByPlot maps a plot id to the field it belongs to;
// a plot missing from the map behaves as not found.
type fakePlotRepo struct {
	fieldByPlot map[uuid.UUID]uuid.UUID
}

func (f *fakePlotRepo) CreatePlot(context.Context, models.Plot) (models.Plot, error) {
	panic("not used by these tests")
}

func (f *fakePlotRepo) GetPlotsByFields(context.Context, []uuid.UUID) ([]models.Plot, error) {
	panic("not used by these tests")
}

func (f *fakePlotRepo) GetNearestPlots(context.Context, float64, float64, int32) ([]models.NearbyPlot, error) {
	panic("not used by these tests")
}

func (f *fakePlotRepo) GetPlotField(_ context.Context, plot uuid.UUID) (uuid.UUID, error) {
	field, ok := f.fieldByPlot[plot]
	if !ok {
		return uuid.UUID{}, ErrNotFound
	}
	return field, nil
}

// fakeNotifier stands in for the notification provider. Only the fan-out the
// board uses is reachable from this service.
type fakeNotifier struct {
	NotificationService

	queued     int
	err        error
	calls      int
	gotFarmer  uuid.UUID
	gotField   *uuid.UUID
	gotPlot    *uuid.UUID
	gotFarm    string
	gotSubject string
	gotBody    string
}

func (f *fakeNotifier) NotifyFarmerCustomers(_ context.Context, farmer uuid.UUID, field, plot *uuid.UUID, farmName, subject, body string) (int, error) {
	f.calls++
	f.gotFarmer, f.gotField, f.gotPlot, f.gotFarm, f.gotSubject, f.gotBody = farmer, field, plot, farmName, subject, body
	if f.err != nil {
		return 0, f.err
	}
	return f.queued, nil
}

// newTestAnnouncementService wires an AnnouncementService whose ownership
// dependencies are empty fakes — fine for any test that never sets a scope,
// since checkScopeOwnership is only reached when field or plot is non-nil.
func newTestAnnouncementService(repo AnnouncementRepository, notifier NotificationService) AnnouncementService {
	return NewAnnouncementService(&fakeFarmRepo{}, &fakeFieldRepo{}, &fakePlotRepo{}, repo, notifier)
}

func TestCreateAnnouncement_StoresThenMailsTheFarmersCustomers(t *testing.T) {
	farmer := uuid.New()
	repo := &fakeAnnouncementRepo{farmName: "Hof Grünwald"}
	notifier := &fakeNotifier{queued: 3}
	svc := newTestAnnouncementService(repo, notifier)

	announcement, recipients, err := svc.CreateAnnouncement(context.Background(), farmer, "Ernte", "Samstag um 9", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.created) != 1 {
		t.Fatalf("stored %d announcements, want 1", len(repo.created))
	}
	if announcement.Subject != "Ernte" || announcement.Body != "Samstag um 9" {
		t.Errorf("announcement = %+v, want the posted subject and body", announcement)
	}
	if announcement.FarmName != "Hof Grünwald" {
		t.Errorf("farm name = %q, want it resolved by the repository", announcement.FarmName)
	}
	if recipients != 3 {
		t.Errorf("recipients = %d, want 3", recipients)
	}

	// The fan-out must be told which farm is writing, or the mail cannot say.
	if notifier.calls != 1 {
		t.Fatalf("notifier called %d times, want 1", notifier.calls)
	}
	if notifier.gotFarmer != farmer || notifier.gotFarm != "Hof Grünwald" {
		t.Errorf("notified farmer %v / farm %q, want %v / %q", notifier.gotFarmer, notifier.gotFarm, farmer, "Hof Grünwald")
	}
	if notifier.gotSubject != "Ernte" || notifier.gotBody != "Samstag um 9" {
		t.Errorf("notified with %q / %q, want the posted subject and body", notifier.gotSubject, notifier.gotBody)
	}
	if notifier.gotField != nil || notifier.gotPlot != nil {
		t.Errorf("notified scope = field:%v plot:%v, want unscoped (nil, nil)", notifier.gotField, notifier.gotPlot)
	}
}

func TestCreateAnnouncement_StoreFailureSendsNothing(t *testing.T) {
	boom := errors.New("db exploded")
	repo := &fakeAnnouncementRepo{createErr: boom}
	notifier := &fakeNotifier{}
	svc := newTestAnnouncementService(repo, notifier)

	if _, _, err := svc.CreateAnnouncement(context.Background(), uuid.New(), "Ernte", "Samstag", nil, nil); !errors.Is(err, boom) {
		t.Errorf("error = %v, want it to wrap the repository failure", err)
	}
	if notifier.calls != 0 {
		t.Errorf("notifier called %d times, want 0 when nothing was stored", notifier.calls)
	}
}

func TestCreateAnnouncement_MailFailureStillKeepsTheAnnouncement(t *testing.T) {
	// The board is what a customer who misses the mail reads instead, so a
	// delivery failure must not undo the post or report it as failed.
	repo := &fakeAnnouncementRepo{farmName: "Hof Grünwald"}
	notifier := &fakeNotifier{err: errors.New("relay refused")}
	svc := newTestAnnouncementService(repo, notifier)

	announcement, recipients, err := svc.CreateAnnouncement(context.Background(), uuid.New(), "Ernte", "Samstag", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if announcement.Subject != "Ernte" {
		t.Errorf("announcement = %+v, want it returned despite the mail failure", announcement)
	}
	if recipients != 0 {
		t.Errorf("recipients = %d, want 0 when nothing could be queued", recipients)
	}
	if len(repo.created) != 1 {
		t.Errorf("stored %d announcements, want the post to survive", len(repo.created))
	}
}

func TestCreateAnnouncement_ScopedToOwnField_PassesScopeThrough(t *testing.T) {
	farmer := uuid.New()
	farm := uuid.New()
	field := uuid.New()
	repo := &fakeAnnouncementRepo{farmName: "Hof Grünwald"}
	notifier := &fakeNotifier{queued: 1}
	farmRepo := &fakeFarmRepo{farmIDByFarmer: map[uuid.UUID]uuid.UUID{farmer: farm}}
	fieldRepo := &fakeFieldRepo{farmByField: map[uuid.UUID]uuid.UUID{field: farm}}
	svc := NewAnnouncementService(farmRepo, fieldRepo, &fakePlotRepo{}, repo, notifier)

	_, _, err := svc.CreateAnnouncement(context.Background(), farmer, "Ernte", "Samstag", &field, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.lastField == nil || *repo.lastField != field {
		t.Errorf("stored field = %v, want %v", repo.lastField, field)
	}
	if repo.lastPlot != nil {
		t.Errorf("stored plot = %v, want nil", repo.lastPlot)
	}
	if notifier.gotField == nil || *notifier.gotField != field {
		t.Errorf("notified field = %v, want %v", notifier.gotField, field)
	}
}

func TestCreateAnnouncement_ScopedToOwnPlot_ResolvesOwnershipThroughItsField(t *testing.T) {
	farmer := uuid.New()
	farm := uuid.New()
	field := uuid.New()
	plot := uuid.New()
	repo := &fakeAnnouncementRepo{farmName: "Hof Grünwald"}
	notifier := &fakeNotifier{queued: 1}
	farmRepo := &fakeFarmRepo{farmIDByFarmer: map[uuid.UUID]uuid.UUID{farmer: farm}}
	fieldRepo := &fakeFieldRepo{farmByField: map[uuid.UUID]uuid.UUID{field: farm}}
	plotRepo := &fakePlotRepo{fieldByPlot: map[uuid.UUID]uuid.UUID{plot: field}}
	svc := NewAnnouncementService(farmRepo, fieldRepo, plotRepo, repo, notifier)

	_, _, err := svc.CreateAnnouncement(context.Background(), farmer, "Ernte", "Samstag", nil, &plot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.lastPlot == nil || *repo.lastPlot != plot {
		t.Errorf("stored plot = %v, want %v", repo.lastPlot, plot)
	}
	if notifier.gotPlot == nil || *notifier.gotPlot != plot {
		t.Errorf("notified plot = %v, want %v", notifier.gotPlot, plot)
	}
}

func TestCreateAnnouncement_BothFieldAndPlotIsRejected(t *testing.T) {
	// The handler already rejects this before calling the service — this
	// proves the service does not merely trust that, since checkScopeOwnership
	// silently skips plot ownership whenever field is also set.
	field := uuid.New()
	plot := uuid.New()
	repo := &fakeAnnouncementRepo{}
	notifier := &fakeNotifier{}
	svc := newTestAnnouncementService(repo, notifier)

	_, _, err := svc.CreateAnnouncement(context.Background(), uuid.New(), "Ernte", "Samstag", &field, &plot)
	if !errors.Is(err, ErrInvalidFilter) {
		t.Errorf("error = %v, want ErrInvalidFilter", err)
	}
	if repo.createdCalls != 0 {
		t.Errorf("stored %d announcements, want 0 when both scopes are set", repo.createdCalls)
	}
	if notifier.calls != 0 {
		t.Errorf("notifier called %d times, want 0 when both scopes are set", notifier.calls)
	}
}

func TestCreateAnnouncement_ScopeOwnedByAnotherFarmerIsForbidden(t *testing.T) {
	farmer := uuid.New()
	field := uuid.New()
	farmRepo := &fakeFarmRepo{farmIDByFarmer: map[uuid.UUID]uuid.UUID{farmer: uuid.New()}}
	fieldRepo := &fakeFieldRepo{farmByField: map[uuid.UUID]uuid.UUID{field: uuid.New()}} // different farm
	repo := &fakeAnnouncementRepo{}
	notifier := &fakeNotifier{}
	svc := NewAnnouncementService(farmRepo, fieldRepo, &fakePlotRepo{}, repo, notifier)

	_, _, err := svc.CreateAnnouncement(context.Background(), farmer, "Ernte", "Samstag", &field, nil)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("error = %v, want ErrForbidden", err)
	}
	if repo.createdCalls != 0 {
		t.Errorf("stored %d announcements, want 0 when ownership fails", repo.createdCalls)
	}
	if notifier.calls != 0 {
		t.Errorf("notifier called %d times, want 0 when ownership fails", notifier.calls)
	}
}

func TestCreateAnnouncement_ScopeFieldNotFound(t *testing.T) {
	farmer := uuid.New()
	field := uuid.New()
	farmRepo := &fakeFarmRepo{farmIDByFarmer: map[uuid.UUID]uuid.UUID{farmer: uuid.New()}}
	svc := NewAnnouncementService(farmRepo, &fakeFieldRepo{}, &fakePlotRepo{}, &fakeAnnouncementRepo{}, &fakeNotifier{})

	_, _, err := svc.CreateAnnouncement(context.Background(), farmer, "Ernte", "Samstag", &field, nil)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestGetAnnouncements_FarmerAndCustomerReadDifferentBoards(t *testing.T) {
	mine := models.AnnouncementWithFarm{
		Announcement: models.Announcement{Subject: "meine"},
		FarmName:     "Hof Grünwald",
	}
	theirs := models.AnnouncementWithFarm{
		Announcement: models.Announcement{Subject: "vom Hof"},
		FarmName:     "Hof Klein",
	}
	repo := &fakeAnnouncementRepo{byFarmer: []models.AnnouncementWithFarm{mine}, forCustomer: []models.AnnouncementWithFarm{theirs}}
	svc := newTestAnnouncementService(repo, &fakeNotifier{})

	farmerBoard, err := svc.GetAnnouncementsForFarmer(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(farmerBoard) != 1 || farmerBoard[0].Subject != "meine" {
		t.Errorf("farmer board = %+v, want only the farmer's own notices", farmerBoard)
	}

	customerBoard, err := svc.GetAnnouncementsForCustomer(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(customerBoard) != 1 || customerBoard[0].Subject != "vom Hof" {
		t.Errorf("customer board = %+v, want the notices of farmers rented from", customerBoard)
	}
	if customerBoard[0].FarmName != "Hof Klein" {
		t.Errorf("farm name = %q, want each notice to say which farm it came from", customerBoard[0].FarmName)
	}
}

func TestGetAnnouncements_RepositoryFailureIsReturned(t *testing.T) {
	boom := errors.New("db exploded")
	svc := newTestAnnouncementService(&fakeAnnouncementRepo{listErr: boom}, &fakeNotifier{})

	if _, err := svc.GetAnnouncementsForFarmer(context.Background(), uuid.New()); !errors.Is(err, boom) {
		t.Errorf("farmer board error = %v, want it to wrap the repository failure", err)
	}
	if _, err := svc.GetAnnouncementsForCustomer(context.Background(), uuid.New()); !errors.Is(err, boom) {
		t.Errorf("customer board error = %v, want it to wrap the repository failure", err)
	}
}
