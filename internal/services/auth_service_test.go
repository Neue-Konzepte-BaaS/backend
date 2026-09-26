package services

import (
	"context"
	"errors"
	"testing"
	"time"

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

	// existingEmail, when set, makes GetAccountByEmail report an already
	// verified account for that address -- simulating an email that is
	// truly taken, as opposed to merely pending.
	existingEmail string

	// byID backs GetAccountByID for TestMe_* below; nil means "not found".
	byID map[uuid.UUID]models.Account
}

func (f *fakeAccountRepo) GetAccountByEmail(_ context.Context, email string) (models.Account, error) {
	if f.existingEmail != "" && email == f.existingEmail {
		return models.Account{Email: email}, nil
	}
	return models.Account{}, ErrNotFound
}

func (f *fakeAccountRepo) GetAccountByID(_ context.Context, id uuid.UUID) (models.Account, error) {
	account, ok := f.byID[id]
	if !ok {
		return models.Account{}, ErrNotFound
	}
	return account, nil
}

func (f *fakeAccountRepo) GetAllRecipients(context.Context) ([]models.Recipient, error) {
	panic("auth service does not send notifications")
}

func (f *fakeAccountRepo) ListAccounts(context.Context, models.AccountListFilter) (models.Page[models.AccountListing], error) {
	panic("auth service does not list accounts")
}

func (f *fakeAccountRepo) GetCustomersOfFarmer(context.Context, uuid.UUID) ([]models.Recipient, error) {
	panic("auth service does not send notifications")
}

func (f *fakeAccountRepo) GetCustomersOfFarmerForField(context.Context, uuid.UUID) ([]models.Recipient, error) {
	panic("auth service does not send notifications")
}

func (f *fakeAccountRepo) GetCustomersOfFarmerForPlot(context.Context, uuid.UUID) ([]models.Recipient, error) {
	panic("auth service does not send notifications")
}

func (f *fakeAccountRepo) GetCustomersOfFarmerForFieldAndCrop(context.Context, uuid.UUID, uuid.UUID) ([]models.Recipient, error) {
	panic("auth service does not send notifications")
}

func (f *fakeAccountRepo) CreateFarmer(_ context.Context, account models.Account, farmName string, postalCode int32, address string, description string) (models.Account, error) {
	if f.createFarmerErr != nil {
		return models.Account{}, f.createFarmerErr
	}
	f.lastFarmName = farmName
	f.lastPostalCode = postalCode
	f.createdRole = models.RoleFarmer
	account.ID = uuid.New()
	account.Role = models.RoleFarmer
	account.PostalCode = postalCode
	return account, nil
}

func (f *fakeAccountRepo) CreateAdmin(_ context.Context, account models.Account) (models.Account, error) {
	panic("auth service does not create admin accounts")
}

func (f *fakeAccountRepo) CreateCustomer(_ context.Context, account models.Account, postalCode int32) (models.Account, error) {
	if f.createCustomerErr != nil {
		return models.Account{}, f.createCustomerErr
	}
	f.lastPostalCode = postalCode
	f.createdRole = models.RoleCustomer
	account.ID = uuid.New()
	account.Role = models.RoleCustomer
	account.PostalCode = postalCode
	return account, nil
}

// fakePendingRegistrationRepo is an in-memory PendingRegistrationRepository.
type fakePendingRegistrationRepo struct {
	byID       map[uuid.UUID]models.PendingRegistration
	byEmail    map[string]uuid.UUID
	deletedIDs []uuid.UUID
}

func newFakePendingRegistrationRepo() *fakePendingRegistrationRepo {
	return &fakePendingRegistrationRepo{
		byID:    map[uuid.UUID]models.PendingRegistration{},
		byEmail: map[string]uuid.UUID{},
	}
}

func (f *fakePendingRegistrationRepo) UpsertPendingRegistration(_ context.Context, reg models.PendingRegistration, _ time.Duration) (uuid.UUID, error) {
	if existing, ok := f.byEmail[reg.Email]; ok {
		reg.ID = existing
	} else {
		reg.ID = uuid.New()
	}
	f.byID[reg.ID] = reg
	f.byEmail[reg.Email] = reg.ID
	return reg.ID, nil
}

func (f *fakePendingRegistrationRepo) GetPendingRegistrationByID(_ context.Context, id uuid.UUID) (models.PendingRegistration, error) {
	reg, ok := f.byID[id]
	if !ok {
		return models.PendingRegistration{}, ErrNotFound
	}
	return reg, nil
}

func (f *fakePendingRegistrationRepo) DeletePendingRegistration(_ context.Context, id uuid.UUID) error {
	f.deletedIDs = append(f.deletedIDs, id)
	delete(f.byID, id)
	return nil
}

// fakeNotificationService is a spy NotificationService: it records the last
// SendMailFromTemplate call so tests can assert a verification email was
// sent, without any real template rendering or delivery.
type fakeNotificationService struct {
	sendErr        error
	lastEmail      string
	lastTemplate   string
	lastData       any
	sendMailCalled int
}

func (f *fakeNotificationService) SendMail(string, string, string, string, bool, map[string][]byte) error {
	panic("auth service uses SendMailFromTemplate, not SendMail")
}

func (f *fakeNotificationService) SendMailFromTemplate(email, _ string, _ string, templateName string, _ map[string][]byte, data any) error {
	f.sendMailCalled++
	f.lastEmail = email
	f.lastTemplate = templateName
	f.lastData = data
	return f.sendErr
}

func (f *fakeNotificationService) SendMailFromTemplateToMany([]models.Recipient, string, string, any) (int, error) {
	panic("auth service does not mail many recipients")
}

func (f *fakeNotificationService) NotifyAllUsers(context.Context, string, string) (int, error) {
	panic("auth service does not send broadcasts")
}

func (f *fakeNotificationService) NotifyFarmerCustomers(context.Context, uuid.UUID, *uuid.UUID, *uuid.UUID, string, string, string) (int, error) {
	panic("auth service does not notify farmer customers")
}

func (f *fakeNotificationService) NotifyRipeness(context.Context, uuid.UUID, uuid.UUID, string, string, string) (int, error) {
	panic("auth service does not send ripeness notices")
}

// newTestService wires an AuthService whose Register/VerifyEmail can be
// exercised end to end: the pending registration and notification spy are
// returned alongside it so tests can inspect what was stored and sent.
func newTestService(repo AccountRepository) (AuthService, *fakePendingRegistrationRepo, *fakeNotificationService) {
	pending := newFakePendingRegistrationRepo()
	notifier := &fakeNotificationService{}
	// Concurrency 1 is enough; Dispatcher.Wait below makes the background
	// send synchronous from the test's point of view regardless.
	dispatcher := NewDispatcher(1)
	// A 32+ char secret satisfies the issuer; the value is irrelevant to tests.
	svc := NewAuthService(repo, pending, credentials.NewIssuer("test-secret-test-secret-test-secret"), notifier, dispatcher, "https://example.com")
	return svc, pending, notifier
}

// registerAndWait calls Register and blocks until its background
// verification email has been (attempted to be) sent, so the test can then
// inspect the notifier spy deterministically.
func registerAndWait(t *testing.T, svc AuthService, pending *fakePendingRegistrationRepo, input RegisterInput) (uuid.UUID, error) {
	t.Helper()
	err := svc.Register(context.Background(), input)
	if impl, ok := svc.(*authService); ok {
		if waitErr := impl.dispatcher.Wait(context.Background()); waitErr != nil {
			t.Fatalf("dispatcher.Wait: %v", waitErr)
		}
	}
	id, ok := pending.byEmail[input.Email]
	if !ok {
		return uuid.UUID{}, err
	}
	return id, err
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
	svc, pending, notifier := newTestService(repo)

	id, err := registerAndWait(t, svc, pending, validCustomerInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.createdRole != "" {
		t.Error("Register must not create the account -- only VerifyEmail does")
	}
	reg, ok := pending.byID[id]
	if !ok {
		t.Fatal("expected a pending registration to be stored")
	}
	if reg.Role != models.RoleCustomer || reg.PostalCode != 76133 {
		t.Errorf("pending registration = %+v, want customer/76133", reg)
	}
	if reg.PasswordHash == "" || reg.PasswordHash == "correct-horse-battery-staple" {
		t.Error("password must be hashed before being stored")
	}
	if notifier.sendMailCalled != 1 {
		t.Fatalf("SendMailFromTemplate called %d times, want 1", notifier.sendMailCalled)
	}
	if notifier.lastEmail != "ada@example.com" {
		t.Errorf("verification email sent to %q, want ada@example.com", notifier.lastEmail)
	}
	if notifier.lastTemplate != verifyEmailTemplate {
		t.Errorf("template = %q, want %q", notifier.lastTemplate, verifyEmailTemplate)
	}
}

func TestRegister_Farmer_RequiresFarmName(t *testing.T) {
	repo := &fakeAccountRepo{}
	svc, pending, notifier := newTestService(repo)

	in := validCustomerInput()
	in.Role = models.RoleFarmer
	in.FarmName = ""

	_, err := registerAndWait(t, svc, pending, in)
	if err == nil {
		t.Fatal("expected an error for a farmer without a farm name")
	}
	if len(pending.byID) != 0 {
		t.Error("no pending registration should be stored when validation fails")
	}
	if notifier.sendMailCalled != 0 {
		t.Error("no email should be sent when validation fails")
	}
}

func TestRegister_Farmer_RequiresAddress(t *testing.T) {
	repo := &fakeAccountRepo{}
	svc, pending, _ := newTestService(repo)

	in := validCustomerInput()
	in.Role = models.RoleFarmer
	in.FarmName = "Green Acres"
	in.Address = ""

	_, err := registerAndWait(t, svc, pending, in)
	if err == nil {
		t.Fatal("expected an error for a farmer without an address")
	}
	if len(pending.byID) != 0 {
		t.Error("no pending registration should be stored when validation fails")
	}
}

func TestRegister_Farmer_OK(t *testing.T) {
	repo := &fakeAccountRepo{}
	svc, pending, _ := newTestService(repo)

	in := validCustomerInput()
	in.Role = models.RoleFarmer
	in.FarmName = "Green Acres"
	in.Address = "1 Farm Lane"
	in.Description = "A small family farm"

	id, err := registerAndWait(t, svc, pending, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	reg := pending.byID[id]
	if reg.Role != models.RoleFarmer {
		t.Errorf("role = %q, want farmer", reg.Role)
	}
	if reg.FarmName != "Green Acres" {
		t.Errorf("farm name = %q, want Green Acres", reg.FarmName)
	}
}

func TestRegister_RejectsAdmin(t *testing.T) {
	repo := &fakeAccountRepo{}
	svc, pending, _ := newTestService(repo)

	in := validCustomerInput()
	in.Role = models.RoleAdmin

	_, err := registerAndWait(t, svc, pending, in)
	if err == nil {
		t.Fatal("expected admin self-registration to be rejected")
	}
	if len(pending.byID) != 0 {
		t.Error("no pending registration should be stored for a rejected role")
	}
}

func TestRegister_MissingFields(t *testing.T) {
	svc, pending, _ := newTestService(&fakeAccountRepo{})

	in := validCustomerInput()
	in.Email = "  " // trimmed to empty

	_, err := registerAndWait(t, svc, pending, in)
	if err == nil {
		t.Fatal("expected an error for a missing email")
	}
}

func TestRegister_EmailTakenPropagates(t *testing.T) {
	repo := &fakeAccountRepo{existingEmail: "ada@example.com"}
	svc, pending, notifier := newTestService(repo)

	_, err := registerAndWait(t, svc, pending, validCustomerInput())
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("error = %v, want ErrEmailTaken", err)
	}
	if notifier.sendMailCalled != 0 {
		t.Error("no email should be sent when the address is already taken")
	}
}

func TestRegister_Repeated_RefreshesPendingRegistration(t *testing.T) {
	repo := &fakeAccountRepo{}
	svc, pending, notifier := newTestService(repo)

	in := validCustomerInput()
	firstID, err := registerAndWait(t, svc, pending, in)
	if err != nil {
		t.Fatalf("unexpected error on first register: %v", err)
	}

	in.FirstName = "Ada-Updated"
	secondID, err := registerAndWait(t, svc, pending, in)
	if err != nil {
		t.Fatalf("unexpected error on second register: %v", err)
	}

	if secondID != firstID {
		t.Errorf("re-registering the same pending email should refresh the same row, got a new id")
	}
	if len(pending.byID) != 1 {
		t.Errorf("expected exactly one pending registration, got %d", len(pending.byID))
	}
	if pending.byID[secondID].FirstName != "Ada-Updated" {
		t.Error("re-registering should refresh the stored data")
	}
	if notifier.sendMailCalled != 2 {
		t.Errorf("expected the verification email to be resent, got %d sends", notifier.sendMailCalled)
	}
}

func TestVerifyEmail_Customer_CreatesAccount(t *testing.T) {
	repo := &fakeAccountRepo{}
	svc, pending, _ := newTestService(repo)

	id, err := registerAndWait(t, svc, pending, validCustomerInput())
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	impl := svc.(*authService)
	token, err := impl.issuer.Issue(id, "", credentials.TypeEmailVerify, credentials.EmailVerifyTTL)
	if err != nil {
		t.Fatalf("issuing token: %v", err)
	}

	account, pair, err := svc.VerifyEmail(context.Background(), token)
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
	if repo.createdRole != models.RoleCustomer {
		t.Error("VerifyEmail should create the account")
	}
	if _, stillPending := pending.byID[id]; stillPending {
		t.Error("the pending registration should be consumed after verification")
	}
}

func TestVerifyEmail_Farmer_CreatesAccount(t *testing.T) {
	repo := &fakeAccountRepo{}
	svc, pending, _ := newTestService(repo)

	in := validCustomerInput()
	in.Role = models.RoleFarmer
	in.FarmName = "Green Acres"
	in.Address = "1 Farm Lane"

	id, err := registerAndWait(t, svc, pending, in)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	impl := svc.(*authService)
	token, err := impl.issuer.Issue(id, "", credentials.TypeEmailVerify, credentials.EmailVerifyTTL)
	if err != nil {
		t.Fatalf("issuing token: %v", err)
	}

	account, _, err := svc.VerifyEmail(context.Background(), token)
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

func TestVerifyEmail_InvalidToken(t *testing.T) {
	svc, _, _ := newTestService(&fakeAccountRepo{})

	_, _, err := svc.VerifyEmail(context.Background(), "not-a-real-token")
	if !errors.Is(err, ErrInvalidVerificationToken) {
		t.Fatalf("error = %v, want ErrInvalidVerificationToken", err)
	}
}

func TestVerifyEmail_WrongTokenType(t *testing.T) {
	repo := &fakeAccountRepo{}
	svc, _, _ := newTestService(repo)
	impl := svc.(*authService)

	// An access token, not an email-verify token, must be rejected the same
	// as a forged one.
	accessToken, err := impl.issuer.Issue(uuid.New(), models.RoleCustomer, credentials.TypeAccess, credentials.AccessTTL)
	if err != nil {
		t.Fatalf("issuing token: %v", err)
	}

	_, _, err = svc.VerifyEmail(context.Background(), accessToken)
	if !errors.Is(err, ErrInvalidVerificationToken) {
		t.Fatalf("error = %v, want ErrInvalidVerificationToken", err)
	}
}

func TestVerifyEmail_UnknownPendingRegistration(t *testing.T) {
	repo := &fakeAccountRepo{}
	svc, _, _ := newTestService(repo)
	impl := svc.(*authService)

	// A well-formed token for a pending registration that was never stored
	// (or already consumed/expired).
	token, err := impl.issuer.Issue(uuid.New(), "", credentials.TypeEmailVerify, credentials.EmailVerifyTTL)
	if err != nil {
		t.Fatalf("issuing token: %v", err)
	}

	_, _, err = svc.VerifyEmail(context.Background(), token)
	if !errors.Is(err, ErrInvalidVerificationToken) {
		t.Fatalf("error = %v, want ErrInvalidVerificationToken", err)
	}
}

func TestVerifyEmail_EmailTakenSinceRegistration(t *testing.T) {
	repo := &fakeAccountRepo{}
	svc, pending, _ := newTestService(repo)

	id, err := registerAndWait(t, svc, pending, validCustomerInput())
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Simulate a second, unrelated registration for the same email having
	// been verified first.
	repo.existingEmail = "ada@example.com"

	impl := svc.(*authService)
	token, err := impl.issuer.Issue(id, "", credentials.TypeEmailVerify, credentials.EmailVerifyTTL)
	if err != nil {
		t.Fatalf("issuing token: %v", err)
	}

	_, _, err = svc.VerifyEmail(context.Background(), token)
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("error = %v, want ErrEmailTaken", err)
	}
}

func TestMe_OK(t *testing.T) {
	id := uuid.New()
	repo := &fakeAccountRepo{byID: map[uuid.UUID]models.Account{
		id: {
			ID:           id,
			FirstName:    "Ada",
			LastName:     "Lovelace",
			Email:        "ada@example.com",
			PasswordHash: "should-never-leave-this-function",
			Role:         models.RoleCustomer,
			PostalCode:   76133,
		},
	}}
	svc, _, _ := newTestService(repo)

	account, err := svc.Me(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if account.FirstName != "Ada" || account.LastName != "Lovelace" {
		t.Errorf("name = %q %q, want Ada Lovelace", account.FirstName, account.LastName)
	}
	if account.PostalCode != 76133 {
		t.Errorf("postal code = %d, want 76133", account.PostalCode)
	}
	if account.PasswordHash != "" {
		t.Error("password hash must not be returned to the caller")
	}
}

func TestMe_NotFound(t *testing.T) {
	svc, _, _ := newTestService(&fakeAccountRepo{})

	_, err := svc.Me(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}
