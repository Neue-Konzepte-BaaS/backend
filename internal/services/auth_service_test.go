package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/credentials"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// fakeAccountRepo is an in-memory AccountRepository for exercising the service
// layer without a database.
type fakeAccountRepo struct {
	createFarmerErr   error
	createCustomerErr error
	lastFarmName      string
	lastPostalCode    int32
	createdRole       models.Role
}

func (f *fakeAccountRepo) GetAccountByEmail(context.Context, string) (models.Account, error) {
	return models.Account{}, ErrNotFound
}

func (f *fakeAccountRepo) GetAccountByID(context.Context, uuid.UUID) (models.Account, error) {
	return models.Account{}, ErrNotFound
}

func (f *fakeAccountRepo) GetAllRecipients(context.Context) ([]models.Recipient, error) {
	panic("auth service does not send notifications")
}

func (f *fakeAccountRepo) GetCustomersOfFarmer(context.Context, uuid.UUID) ([]models.Recipient, error) {
	panic("auth service does not send notifications")
}

func (f *fakeAccountRepo) CreateFarmer(_ context.Context, account models.Account, farmName string, postalCode int32) (models.Account, error) {
	if f.createFarmerErr != nil {
		return models.Account{}, f.createFarmerErr
	}
	f.lastFarmName = farmName
	f.lastPostalCode = postalCode
	f.createdRole = models.RoleFarmer
	account.ID = uuid.New()
	account.Role = models.RoleFarmer
	return account, nil
}

func (f *fakeAccountRepo) CreateCustomer(_ context.Context, account models.Account, postalCode int32) (models.Account, error) {
	if f.createCustomerErr != nil {
		return models.Account{}, f.createCustomerErr
	}
	f.lastPostalCode = postalCode
	f.createdRole = models.RoleCustomer
	account.ID = uuid.New()
	account.Role = models.RoleCustomer
	return account, nil
}

func newTestService(repo AccountRepository) AuthService {
	// A 32+ char secret satisfies the issuer; the value is irrelevant to tests.
	return NewAuthService(repo, credentials.NewIssuer("test-secret-test-secret-test-secret"))
}

func validCustomerInput() RegisterInput {
	return RegisterInput{
		FirstName:  "Ada",
		LastName:   "Lovelace",
		Email:      "ada@example.com",
		Password:   "correct-horse-battery-staple",
		Role:       models.RoleCustomer,
		PostalCode: 76133,
	}
}

func TestRegister_Customer(t *testing.T) {
	repo := &fakeAccountRepo{}
	svc := newTestService(repo)

	account, pair, err := svc.Register(context.Background(), validCustomerInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if account.Role != models.RoleCustomer {
		t.Errorf("role = %q, want customer", account.Role)
	}
	if account.PasswordHash != "" {
		t.Error("password hash must not be returned to the caller")
	}
	if pair.Access == "" || pair.Refresh == "" {
		t.Error("expected a token pair to be issued")
	}
	if repo.lastPostalCode != 76133 {
		t.Errorf("postal code = %d, want 76133", repo.lastPostalCode)
	}
}

func TestRegister_Farmer_RequiresFarmName(t *testing.T) {
	repo := &fakeAccountRepo{}
	svc := newTestService(repo)

	in := validCustomerInput()
	in.Role = models.RoleFarmer
	in.FarmName = ""

	_, _, err := svc.Register(context.Background(), in)
	if err == nil {
		t.Fatal("expected an error for a farmer without a farm name")
	}
	if repo.createdRole != "" {
		t.Error("repository should not be called when validation fails")
	}
}

func TestRegister_Farmer_OK(t *testing.T) {
	repo := &fakeAccountRepo{}
	svc := newTestService(repo)

	in := validCustomerInput()
	in.Role = models.RoleFarmer
	in.FarmName = "Green Acres"

	account, _, err := svc.Register(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if account.Role != models.RoleFarmer {
		t.Errorf("role = %q, want farmer", account.Role)
	}
	if repo.lastFarmName != "Green Acres" {
		t.Errorf("farm name = %q, want Green Acres", repo.lastFarmName)
	}
}

func TestRegister_RejectsAdmin(t *testing.T) {
	repo := &fakeAccountRepo{}
	svc := newTestService(repo)

	in := validCustomerInput()
	in.Role = models.RoleAdmin

	_, _, err := svc.Register(context.Background(), in)
	if err == nil {
		t.Fatal("expected admin self-registration to be rejected")
	}
	if repo.createdRole != "" {
		t.Error("repository should not be called for a rejected role")
	}
}

func TestRegister_MissingFields(t *testing.T) {
	svc := newTestService(&fakeAccountRepo{})

	in := validCustomerInput()
	in.Email = "  " // trimmed to empty

	_, _, err := svc.Register(context.Background(), in)
	if err == nil {
		t.Fatal("expected an error for a missing email")
	}
}

func TestRegister_EmailTakenPropagates(t *testing.T) {
	repo := &fakeAccountRepo{createCustomerErr: ErrEmailTaken}
	svc := newTestService(repo)

	_, _, err := svc.Register(context.Background(), validCustomerInput())
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("error = %v, want ErrEmailTaken", err)
	}
}
