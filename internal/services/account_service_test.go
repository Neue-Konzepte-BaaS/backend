package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// fakeAccountListRepo is an AccountRepository that only answers listings; it
// records the filter it was handed so tests can prove what the service passed
// down rather than only what it returned.
type fakeAccountListRepo struct {
	page   models.Page[models.AccountListing]
	err    error
	called bool
	filter models.AccountListFilter
}

func (f *fakeAccountListRepo) ListAccounts(_ context.Context, filter models.AccountListFilter) (models.Page[models.AccountListing], error) {
	f.called = true
	f.filter = filter
	if f.err != nil {
		return models.Page[models.AccountListing]{}, f.err
	}
	return f.page, nil
}

func (f *fakeAccountListRepo) GetAccountByEmail(context.Context, string) (models.Account, error) {
	panic("the account listing does not look up by email")
}

func (f *fakeAccountListRepo) GetAccountByID(context.Context, uuid.UUID) (models.Account, error) {
	panic("the account listing does not look up by id")
}

func (f *fakeAccountListRepo) CreateAdmin(context.Context, models.Account) (models.Account, error) {
	panic("the account listing does not create accounts")
}

func (f *fakeAccountListRepo) CreateFarmer(context.Context, models.Account, string, int32, string, string) (models.Account, error) {
	panic("the account listing does not create accounts")
}

func (f *fakeAccountListRepo) CreateCustomer(context.Context, models.Account, int32) (models.Account, error) {
	panic("the account listing does not create accounts")
}

func (f *fakeAccountListRepo) GetAllRecipients(context.Context) ([]models.Recipient, error) {
	panic("the account listing does not send notifications")
}

func (f *fakeAccountListRepo) GetCustomersOfFarmer(context.Context, uuid.UUID) ([]models.Recipient, error) {
	panic("the account listing does not send notifications")
}

func (f *fakeAccountListRepo) GetCustomersOfFarmerForField(context.Context, uuid.UUID) ([]models.Recipient, error) {
	panic("the account listing does not send notifications")
}

func (f *fakeAccountListRepo) GetCustomersOfFarmerForPlot(context.Context, uuid.UUID) ([]models.Recipient, error) {
	panic("the account listing does not send notifications")
}

func (f *fakeAccountListRepo) GetCustomersOfFarmerForPlotAndCrop(context.Context, uuid.UUID, uuid.UUID) ([]models.Recipient, error) {
	panic("the account listing does not send notifications")
}

func (f *fakeAccountListRepo) SoftDeleteAccount(context.Context, uuid.UUID, string, string, string, string) error {
	panic("the account listing does not delete accounts")
}

func (f *fakeAccountListRepo) HasActiveRentalAsCustomer(context.Context, uuid.UUID) (bool, error) {
	panic("the account listing does not check rentals")
}

func (f *fakeAccountListRepo) HasActiveRentalAsFarmer(context.Context, uuid.UUID) (bool, error) {
	panic("the account listing does not check rentals")
}

func (f *fakeAccountListRepo) GetCustomerNotificationPreferences(context.Context, uuid.UUID) (models.CustomerNotificationPreferences, error) {
	panic("the account listing does not read notification preferences")
}

func (f *fakeAccountListRepo) UpdateCustomerNotificationPreferences(context.Context, uuid.UUID, models.CustomerNotificationPreferences) error {
	panic("the account listing does not update notification preferences")
}

// fakeDeletionFarmerSubRepo is a FarmerSubscriptionRepository stub for
// DeleteMyAccount tests that never reach the farmer-subscription branch.
type fakeDeletionFarmerSubRepo struct{ FarmerSubscriptionRepository }

// fakeDeletionPaymentGateway is a PaymentGateway stub for the same reason.
type fakeDeletionPaymentGateway struct{ PaymentGateway }

func TestListAccounts_OnlyAdminReachesTheRepository(t *testing.T) {
	tests := []struct {
		name      string
		role      models.Role
		wantErr   error
		wantCalls bool
	}{
		{name: "admin", role: models.RoleAdmin, wantCalls: true},
		{name: "farmer", role: models.RoleFarmer, wantErr: ErrForbidden},
		{name: "customer", role: models.RoleCustomer, wantErr: ErrForbidden},
		{name: "no role at all", role: "", wantErr: ErrForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAccountListRepo{}
			svc := NewAccountService(repo, &fakeDeletionFarmerSubRepo{}, &fakeDeletionPaymentGateway{})

			_, err := svc.ListAccounts(context.Background(), tt.role, models.AccountListFilter{})

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if repo.called != tt.wantCalls {
				t.Errorf("repository called = %v, want %v", repo.called, tt.wantCalls)
			}
		})
	}
}

func TestListAccounts_RejectsUnknownRoleFilter(t *testing.T) {
	tests := []struct {
		name    string
		filter  models.Role
		wantErr error
	}{
		{name: "empty means any", filter: ""},
		{name: "admin", filter: models.RoleAdmin},
		{name: "farmer", filter: models.RoleFarmer},
		{name: "customer", filter: models.RoleCustomer},
		{name: "typo", filter: "framer", wantErr: ErrInvalidFilter},
		{name: "wrong case", filter: "Admin", wantErr: ErrInvalidFilter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAccountListRepo{}
			svc := NewAccountService(repo, &fakeDeletionFarmerSubRepo{}, &fakeDeletionPaymentGateway{})

			_, err := svc.ListAccounts(context.Background(), models.RoleAdmin, models.AccountListFilter{Role: tt.filter})

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			// An unknown role must not reach the database at all: coming back
			// with an empty page would read as "no such accounts" rather than
			// "no such role".
			if tt.wantErr != nil && repo.called {
				t.Error("an invalid role filter must not reach the repository")
			}
		})
	}
}

func TestListAccounts_ClampsPagination(t *testing.T) {
	tests := []struct {
		name                  string
		limit, offset         int32
		wantLimit, wantOffset int32
	}{
		{name: "unset limit becomes the default", limit: 0, wantLimit: DefaultPageLimit},
		{name: "negative limit becomes the default", limit: -5, wantLimit: DefaultPageLimit},
		{name: "oversized limit is capped", limit: 5000, wantLimit: MaxPageLimit},
		{name: "limit at the cap is kept", limit: MaxPageLimit, wantLimit: MaxPageLimit},
		{name: "ordinary limit is kept", limit: 7, wantLimit: 7},
		{name: "negative offset becomes zero", limit: 7, offset: -1, wantLimit: 7, wantOffset: 0},
		{name: "offset is otherwise kept", limit: 7, offset: 40, wantLimit: 7, wantOffset: 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAccountListRepo{}
			svc := NewAccountService(repo, &fakeDeletionFarmerSubRepo{}, &fakeDeletionPaymentGateway{})

			_, err := svc.ListAccounts(context.Background(), models.RoleAdmin, models.AccountListFilter{
				Limit:  tt.limit,
				Offset: tt.offset,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if repo.filter.Limit != tt.wantLimit {
				t.Errorf("limit = %d, want %d", repo.filter.Limit, tt.wantLimit)
			}
			if repo.filter.Offset != tt.wantOffset {
				t.Errorf("offset = %d, want %d", repo.filter.Offset, tt.wantOffset)
			}
		})
	}
}

func TestListAccounts_TrimsTheSearchTerm(t *testing.T) {
	repo := &fakeAccountListRepo{}
	svc := NewAccountService(repo, &fakeDeletionFarmerSubRepo{}, &fakeDeletionPaymentGateway{})

	if _, err := svc.ListAccounts(context.Background(), models.RoleAdmin, models.AccountListFilter{Query: "  ada  "}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.filter.Query != "ada" {
		t.Errorf("query = %q, want %q", repo.filter.Query, "ada")
	}
}

func TestListAccounts_PassesThePageThrough(t *testing.T) {
	want := models.Page[models.AccountListing]{
		Items: []models.AccountListing{{Email: "ada@example.com", Role: models.RoleCustomer}},
		Total: 137,
	}
	repo := &fakeAccountListRepo{page: want}
	svc := NewAccountService(repo, &fakeDeletionFarmerSubRepo{}, &fakeDeletionPaymentGateway{})

	got, err := svc.ListAccounts(context.Background(), models.RoleAdmin, models.AccountListFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Total != want.Total {
		t.Errorf("total = %d, want %d", got.Total, want.Total)
	}
	if len(got.Items) != 1 || got.Items[0].Email != want.Items[0].Email {
		t.Errorf("items = %v, want %v", got.Items, want.Items)
	}
}

func TestListAccounts_WrapsRepositoryErrors(t *testing.T) {
	sentinel := errors.New("connection refused")
	repo := &fakeAccountListRepo{err: sentinel}
	svc := NewAccountService(repo, &fakeDeletionFarmerSubRepo{}, &fakeDeletionPaymentGateway{})

	_, err := svc.ListAccounts(context.Background(), models.RoleAdmin, models.AccountListFilter{})
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want it to wrap %v", err, sentinel)
	}
}

// fakeNotificationPrefsRepo is an AccountRepository that only answers
// notification-preference reads/writes.
type fakeNotificationPrefsRepo struct {
	prefs     models.CustomerNotificationPreferences
	getErr    error
	updateErr error
	updated   bool
	gotPrefs  models.CustomerNotificationPreferences
}

func (f *fakeNotificationPrefsRepo) GetCustomerNotificationPreferences(context.Context, uuid.UUID) (models.CustomerNotificationPreferences, error) {
	if f.getErr != nil {
		return models.CustomerNotificationPreferences{}, f.getErr
	}
	return f.prefs, nil
}

func (f *fakeNotificationPrefsRepo) UpdateCustomerNotificationPreferences(_ context.Context, _ uuid.UUID, prefs models.CustomerNotificationPreferences) error {
	f.updated = true
	f.gotPrefs = prefs
	if f.updateErr != nil {
		return f.updateErr
	}
	return nil
}

func (f *fakeNotificationPrefsRepo) GetAccountByEmail(context.Context, string) (models.Account, error) {
	panic("not exercised by notification preference tests")
}

func (f *fakeNotificationPrefsRepo) GetAccountByID(context.Context, uuid.UUID) (models.Account, error) {
	panic("not exercised by notification preference tests")
}

func (f *fakeNotificationPrefsRepo) CreateAdmin(context.Context, models.Account) (models.Account, error) {
	panic("not exercised by notification preference tests")
}

func (f *fakeNotificationPrefsRepo) CreateFarmer(context.Context, models.Account, string, int32, string, string) (models.Account, error) {
	panic("not exercised by notification preference tests")
}

func (f *fakeNotificationPrefsRepo) CreateCustomer(context.Context, models.Account, int32) (models.Account, error) {
	panic("not exercised by notification preference tests")
}

func (f *fakeNotificationPrefsRepo) GetAllRecipients(context.Context) ([]models.Recipient, error) {
	panic("not exercised by notification preference tests")
}

func (f *fakeNotificationPrefsRepo) ListAccounts(context.Context, models.AccountListFilter) (models.Page[models.AccountListing], error) {
	panic("not exercised by notification preference tests")
}

func (f *fakeNotificationPrefsRepo) GetCustomersOfFarmer(context.Context, uuid.UUID) ([]models.Recipient, error) {
	panic("not exercised by notification preference tests")
}

func (f *fakeNotificationPrefsRepo) GetCustomersOfFarmerForField(context.Context, uuid.UUID) ([]models.Recipient, error) {
	panic("not exercised by notification preference tests")
}

func (f *fakeNotificationPrefsRepo) GetCustomersOfFarmerForPlot(context.Context, uuid.UUID) ([]models.Recipient, error) {
	panic("not exercised by notification preference tests")
}

func (f *fakeNotificationPrefsRepo) GetCustomersOfFarmerForPlotAndCrop(context.Context, uuid.UUID, uuid.UUID) ([]models.Recipient, error) {
	panic("not exercised by notification preference tests")
}

func (f *fakeNotificationPrefsRepo) SoftDeleteAccount(context.Context, uuid.UUID, string, string, string, string) error {
	panic("not exercised by notification preference tests")
}

func (f *fakeNotificationPrefsRepo) HasActiveRentalAsCustomer(context.Context, uuid.UUID) (bool, error) {
	panic("not exercised by notification preference tests")
}

func (f *fakeNotificationPrefsRepo) HasActiveRentalAsFarmer(context.Context, uuid.UUID) (bool, error) {
	panic("not exercised by notification preference tests")
}

func TestGetMyNotificationPreferences_OnlyCustomerReachesTheRepository(t *testing.T) {
	tests := []struct {
		name    string
		role    models.Role
		wantErr error
	}{
		{name: "customer", role: models.RoleCustomer},
		{name: "farmer", role: models.RoleFarmer, wantErr: ErrForbidden},
		{name: "admin", role: models.RoleAdmin, wantErr: ErrForbidden},
		{name: "no role at all", role: "", wantErr: ErrForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeNotificationPrefsRepo{prefs: models.CustomerNotificationPreferences{NotifyMessagesByEmail: true}}
			svc := NewAccountService(repo, &fakeDeletionFarmerSubRepo{}, &fakeDeletionPaymentGateway{})

			_, err := svc.GetMyNotificationPreferences(context.Background(), uuid.New(), tt.role)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateMyNotificationPreferences_OnlyCustomerReachesTheRepository(t *testing.T) {
	tests := []struct {
		name    string
		role    models.Role
		wantErr error
	}{
		{name: "customer", role: models.RoleCustomer},
		{name: "farmer", role: models.RoleFarmer, wantErr: ErrForbidden},
		{name: "admin", role: models.RoleAdmin, wantErr: ErrForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeNotificationPrefsRepo{}
			svc := NewAccountService(repo, &fakeDeletionFarmerSubRepo{}, &fakeDeletionPaymentGateway{})

			_, err := svc.UpdateMyNotificationPreferences(context.Background(), uuid.New(), tt.role, models.CustomerNotificationPreferences{NotifyMessagesByEmail: false})

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if repo.updated != (tt.wantErr == nil) {
				t.Errorf("repository updated = %v, want %v", repo.updated, tt.wantErr == nil)
			}
		})
	}
}

func TestUpdateMyNotificationPreferences_PassesPreferencesThrough(t *testing.T) {
	repo := &fakeNotificationPrefsRepo{}
	svc := NewAccountService(repo, &fakeDeletionFarmerSubRepo{}, &fakeDeletionPaymentGateway{})

	got, err := svc.UpdateMyNotificationPreferences(context.Background(), uuid.New(), models.RoleCustomer, models.CustomerNotificationPreferences{NotifyMessagesByEmail: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.NotifyMessagesByEmail {
		t.Errorf("NotifyMessagesByEmail = true, want false")
	}
	if repo.gotPrefs.NotifyMessagesByEmail {
		t.Errorf("repository received NotifyMessagesByEmail = true, want false")
	}
}

func TestGetMyNotificationPreferences_WrapsRepositoryErrors(t *testing.T) {
	sentinel := errors.New("connection refused")
	repo := &fakeNotificationPrefsRepo{getErr: sentinel}
	svc := NewAccountService(repo, &fakeDeletionFarmerSubRepo{}, &fakeDeletionPaymentGateway{})

	_, err := svc.GetMyNotificationPreferences(context.Background(), uuid.New(), models.RoleCustomer)
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want it to wrap %v", err, sentinel)
	}
}
