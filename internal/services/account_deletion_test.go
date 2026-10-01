package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/credentials"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// fakeAccountDeletionRepo answers only what DeleteAccount asks of an
// AccountRepository; everything else panics through the embedded listing fake.
type fakeAccountDeletionRepo struct {
	*fakeAccountListRepo
	hash        string
	hashErr     error
	blockers    models.AccountDeletionBlockers
	deleteErr   error
	deletedID   uuid.UUID
	deletedRole models.Role
}

func (f *fakeAccountDeletionRepo) GetPasswordHash(context.Context, uuid.UUID) (string, error) {
	return f.hash, f.hashErr
}

func (f *fakeAccountDeletionRepo) GetDeletionBlockers(context.Context, uuid.UUID) (models.AccountDeletionBlockers, error) {
	return f.blockers, nil
}

func (f *fakeAccountDeletionRepo) DeleteAccount(_ context.Context, id uuid.UUID, role models.Role) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deletedID = id
	f.deletedRole = role
	return nil
}

// fakeDeletionSubRepo implements the two FarmerSubscriptionRepository methods
// DeleteAccount uses; the embedded nil interface makes any other call panic.
type fakeDeletionSubRepo struct {
	FarmerSubscriptionRepository
	active      *models.FarmerSubscription
	canceledIDs []string
}

func (f *fakeDeletionSubRepo) GetActiveSubscriptionByFarmer(context.Context, uuid.UUID) (models.FarmerSubscription, error) {
	if f.active == nil {
		return models.FarmerSubscription{}, ErrNotFound
	}
	return *f.active, nil
}

func (f *fakeDeletionSubRepo) CancelSubscription(_ context.Context, stripeSubscriptionID string) (models.FarmerSubscription, error) {
	f.canceledIDs = append(f.canceledIDs, stripeSubscriptionID)
	return models.FarmerSubscription{}, nil
}

var deletionTestHash = func() string {
	hash, err := credentials.HashPassword("correct horse")
	if err != nil {
		panic(err)
	}
	return hash
}()

func TestDeleteAccount(t *testing.T) {
	stripeID := "sub_123"

	tests := []struct {
		name         string
		role         models.Role
		password     string
		hashErr      error
		blockers     models.AccountDeletionBlockers
		subscription *models.FarmerSubscription
		cancelErr    error
		wantErr      error
		wantDeleted  bool
		wantCanceled bool
	}{
		{name: "customer", role: models.RoleCustomer, password: "correct horse", wantDeleted: true},
		{name: "farmer without subscription", role: models.RoleFarmer, password: "correct horse", wantDeleted: true},
		{
			name: "farmer with subscription", role: models.RoleFarmer, password: "correct horse",
			subscription: &models.FarmerSubscription{StripeSubscriptionID: &stripeID},
			wantDeleted:  true, wantCanceled: true,
		},
		{name: "admin", role: models.RoleAdmin, password: "correct horse", wantErr: ErrForbidden},
		{name: "wrong password", role: models.RoleCustomer, password: "wrong", wantErr: ErrInvalidCredentials},
		{name: "already deleted", role: models.RoleCustomer, password: "correct horse", hashErr: ErrNotFound, wantErr: ErrNotFound},
		{
			name: "open rentals", role: models.RoleCustomer, password: "correct horse",
			blockers: models.AccountDeletionBlockers{OpenRentals: true}, wantErr: ErrAccountHasOpenRentals,
		},
		{
			name: "pending payment", role: models.RoleFarmer, password: "correct horse",
			blockers:     models.AccountDeletionBlockers{PendingPayment: true},
			subscription: &models.FarmerSubscription{StripeSubscriptionID: &stripeID},
			wantErr:      ErrAccountHasPendingPayment,
		},
		{
			// Stripe refusing must keep the account: deleting it anyway would
			// leave a farmer nobody can reach still being billed.
			name: "stripe cancel fails", role: models.RoleFarmer, password: "correct horse",
			subscription: &models.FarmerSubscription{StripeSubscriptionID: &stripeID},
			cancelErr:    errors.New("stripe down"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAccountDeletionRepo{hash: deletionTestHash, hashErr: tt.hashErr, blockers: tt.blockers}
			subRepo := &fakeDeletionSubRepo{active: tt.subscription}
			gateway := &fakePaymentGateway{cancelErr: tt.cancelErr}
			svc := NewAccountService(repo, subRepo, gateway)
			id := uuid.New()

			err := svc.DeleteAccount(context.Background(), id, tt.role, tt.password)

			switch {
			case tt.wantErr != nil && !errors.Is(err, tt.wantErr):
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			case tt.cancelErr != nil && !errors.Is(err, tt.cancelErr):
				t.Fatalf("error = %v, want %v", err, tt.cancelErr)
			case tt.wantErr == nil && tt.cancelErr == nil && err != nil:
				t.Fatalf("unexpected error: %v", err)
			}

			if deleted := repo.deletedID == id; deleted != tt.wantDeleted {
				t.Fatalf("account deleted = %v, want %v", deleted, tt.wantDeleted)
			}
			if tt.wantDeleted && repo.deletedRole != tt.role {
				t.Fatalf("deleted with role %q, want %q", repo.deletedRole, tt.role)
			}

			canceled := len(gateway.canceledSubscriptions) == 1 && len(subRepo.canceledIDs) == 1
			if canceled != tt.wantCanceled {
				t.Fatalf("subscription canceled = %v (stripe %v, local %v), want %v",
					canceled, gateway.canceledSubscriptions, subRepo.canceledIDs, tt.wantCanceled)
			}
		})
	}
}
