package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// fakeUpgradeFarmerSubRepo implements only what UpgradeSubscription needs
// from FarmerSubscriptionRepository.
type fakeUpgradeFarmerSubRepo struct {
	FarmerSubscriptionRepository
	activeSub    models.FarmerSubscription
	activeSubErr error

	updatedPlan uuid.UUID
	updateErr   error
	updatedSub  models.FarmerSubscription
}

func (f *fakeUpgradeFarmerSubRepo) GetActiveSubscriptionByFarmer(context.Context, uuid.UUID) (models.FarmerSubscription, error) {
	if f.activeSubErr != nil {
		return models.FarmerSubscription{}, f.activeSubErr
	}
	return f.activeSub, nil
}

func (f *fakeUpgradeFarmerSubRepo) UpdateSubscriptionPlan(_ context.Context, id, plan uuid.UUID, currentPeriodEnd time.Time) (models.FarmerSubscription, error) {
	if f.updateErr != nil {
		return models.FarmerSubscription{}, f.updateErr
	}
	f.updatedPlan = plan
	f.updatedSub = models.FarmerSubscription{
		ID:               id,
		Plan:             plan,
		Status:           models.FarmerSubscriptionActive,
		CurrentPeriodEnd: &currentPeriodEnd,
	}
	return f.updatedSub, nil
}

// fakeUpgradePlanRepo implements only what UpgradeSubscription needs from
// SubscriptionPlanRepository.
type fakeUpgradePlanRepo struct {
	SubscriptionPlanRepository
	plans map[uuid.UUID]models.SubscriptionPlan
}

func (f *fakeUpgradePlanRepo) GetSubscriptionPlanByID(_ context.Context, id uuid.UUID) (models.SubscriptionPlan, error) {
	plan, ok := f.plans[id]
	if !ok {
		return models.SubscriptionPlan{}, ErrNotFound
	}
	return plan, nil
}

// fakeUpgradeGateway implements only what UpgradeSubscription needs from
// PaymentGateway.
type fakeUpgradeGateway struct {
	PaymentGateway
	gotStripeSubscriptionID, gotStripePriceID string
	currentPeriodEnd                          time.Time
	updateErr                                 error
}

func (f *fakeUpgradeGateway) UpdateSubscriptionPrice(_ context.Context, stripeSubscriptionID, newStripePriceID string) (time.Time, error) {
	f.gotStripeSubscriptionID = stripeSubscriptionID
	f.gotStripePriceID = newStripePriceID
	if f.updateErr != nil {
		return time.Time{}, f.updateErr
	}
	return f.currentPeriodEnd, nil
}

func newTestUpgradeService(farmerSubRepo *fakeUpgradeFarmerSubRepo, planRepo *fakeUpgradePlanRepo, gateway *fakeUpgradeGateway) SubscriptionService {
	return NewSubscriptionService(farmerSubRepo, planRepo, gateway, nil, "https://example.test")
}

func TestSubscriptionService_UpgradeSubscription_OK(t *testing.T) {
	subID := uuid.New()
	cheapPlanID := uuid.New()
	expensivePlanID := uuid.New()
	stripeSubID := "sub_123"
	expensiveStripePriceID := "price_expensive"
	newPeriodEnd := time.Now().Add(30 * 24 * time.Hour)

	farmerSubRepo := &fakeUpgradeFarmerSubRepo{
		activeSub: models.FarmerSubscription{
			ID:                   subID,
			Plan:                 cheapPlanID,
			Status:               models.FarmerSubscriptionActive,
			StripeSubscriptionID: &stripeSubID,
		},
	}
	planRepo := &fakeUpgradePlanRepo{plans: map[uuid.UUID]models.SubscriptionPlan{
		cheapPlanID:     {ID: cheapPlanID, Code: "cheap", PriceCents: 990, IsActive: true, StripePriceID: "price_cheap"},
		expensivePlanID: {ID: expensivePlanID, Code: "expensive", PriceCents: 9900, IsActive: true, StripePriceID: expensiveStripePriceID},
	}}
	gateway := &fakeUpgradeGateway{currentPeriodEnd: newPeriodEnd}

	svc := newTestUpgradeService(farmerSubRepo, planRepo, gateway)

	got, err := svc.UpgradeSubscription(context.Background(), uuid.New(), expensivePlanID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gateway.gotStripeSubscriptionID != stripeSubID {
		t.Errorf("stripe subscription id = %q, want %q", gateway.gotStripeSubscriptionID, stripeSubID)
	}
	if gateway.gotStripePriceID != expensiveStripePriceID {
		t.Errorf("stripe price id = %q, want %q", gateway.gotStripePriceID, expensiveStripePriceID)
	}
	if farmerSubRepo.updatedPlan != expensivePlanID {
		t.Errorf("updated plan = %v, want %v", farmerSubRepo.updatedPlan, expensivePlanID)
	}
	if got.Plan != expensivePlanID {
		t.Errorf("returned plan = %v, want %v", got.Plan, expensivePlanID)
	}
	if got.CurrentPeriodEnd == nil || !got.CurrentPeriodEnd.Equal(newPeriodEnd) {
		t.Errorf("current period end = %v, want %v", got.CurrentPeriodEnd, newPeriodEnd)
	}
}

func TestSubscriptionService_UpgradeSubscription_NoActiveSubscription(t *testing.T) {
	farmerSubRepo := &fakeUpgradeFarmerSubRepo{activeSubErr: ErrNotFound}
	planRepo := &fakeUpgradePlanRepo{plans: map[uuid.UUID]models.SubscriptionPlan{}}
	gateway := &fakeUpgradeGateway{}

	svc := newTestUpgradeService(farmerSubRepo, planRepo, gateway)

	_, err := svc.UpgradeSubscription(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestSubscriptionService_UpgradeSubscription_TargetPlanNotPricedHigher(t *testing.T) {
	cheapPlanID := uuid.New()
	modestPlanID := uuid.New()
	stripeSubID := "sub_123"

	farmerSubRepo := &fakeUpgradeFarmerSubRepo{
		activeSub: models.FarmerSubscription{
			ID:                   uuid.New(),
			Plan:                 modestPlanID,
			Status:               models.FarmerSubscriptionActive,
			StripeSubscriptionID: &stripeSubID,
		},
	}
	planRepo := &fakeUpgradePlanRepo{plans: map[uuid.UUID]models.SubscriptionPlan{
		cheapPlanID:  {ID: cheapPlanID, Code: "cheap", PriceCents: 990, IsActive: true, StripePriceID: "price_cheap"},
		modestPlanID: {ID: modestPlanID, Code: "modest", PriceCents: 2900, IsActive: true, StripePriceID: "price_modest"},
	}}
	gateway := &fakeUpgradeGateway{}

	svc := newTestUpgradeService(farmerSubRepo, planRepo, gateway)

	// Same tier.
	if _, err := svc.UpgradeSubscription(context.Background(), uuid.New(), modestPlanID); !errors.Is(err, ErrNotAnUpgrade) {
		t.Fatalf("same-tier error = %v, want ErrNotAnUpgrade", err)
	}
	// Lower tier.
	if _, err := svc.UpgradeSubscription(context.Background(), uuid.New(), cheapPlanID); !errors.Is(err, ErrNotAnUpgrade) {
		t.Fatalf("downgrade error = %v, want ErrNotAnUpgrade", err)
	}
	if gateway.gotStripeSubscriptionID != "" {
		t.Error("stripe gateway should not have been called")
	}
}

func TestSubscriptionService_UpgradeSubscription_TargetPlanInactive(t *testing.T) {
	cheapPlanID := uuid.New()
	retiredPlanID := uuid.New()
	stripeSubID := "sub_123"

	farmerSubRepo := &fakeUpgradeFarmerSubRepo{
		activeSub: models.FarmerSubscription{
			ID:                   uuid.New(),
			Plan:                 cheapPlanID,
			Status:               models.FarmerSubscriptionActive,
			StripeSubscriptionID: &stripeSubID,
		},
	}
	planRepo := &fakeUpgradePlanRepo{plans: map[uuid.UUID]models.SubscriptionPlan{
		cheapPlanID:   {ID: cheapPlanID, Code: "cheap", PriceCents: 990, IsActive: true, StripePriceID: "price_cheap"},
		retiredPlanID: {ID: retiredPlanID, Code: "expensive", PriceCents: 9900, IsActive: false, StripePriceID: "price_expensive"},
	}}
	gateway := &fakeUpgradeGateway{}

	svc := newTestUpgradeService(farmerSubRepo, planRepo, gateway)

	_, err := svc.UpgradeSubscription(context.Background(), uuid.New(), retiredPlanID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestSubscriptionService_UpgradeSubscription_TargetPlanDoesNotExist(t *testing.T) {
	cheapPlanID := uuid.New()
	stripeSubID := "sub_123"

	farmerSubRepo := &fakeUpgradeFarmerSubRepo{
		activeSub: models.FarmerSubscription{
			ID:                   uuid.New(),
			Plan:                 cheapPlanID,
			Status:               models.FarmerSubscriptionActive,
			StripeSubscriptionID: &stripeSubID,
		},
	}
	planRepo := &fakeUpgradePlanRepo{plans: map[uuid.UUID]models.SubscriptionPlan{
		cheapPlanID: {ID: cheapPlanID, Code: "cheap", PriceCents: 990, IsActive: true, StripePriceID: "price_cheap"},
	}}
	gateway := &fakeUpgradeGateway{}

	svc := newTestUpgradeService(farmerSubRepo, planRepo, gateway)

	_, err := svc.UpgradeSubscription(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}
