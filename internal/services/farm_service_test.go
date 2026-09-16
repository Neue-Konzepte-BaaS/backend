package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// fakeFarmRepo is an in-memory FarmRepository for exercising the service
// layer without a database. farmIDByFarmer maps a farmer's account id to
// their farm id, for GetFarmIDByFarmerID.
type fakeFarmRepo struct {
	farm           models.Farm
	farmErr        error
	farmIDByFarmer map[uuid.UUID]uuid.UUID
	farmIDErr      error
}

func (f *fakeFarmRepo) GetFarmByID(context.Context, uuid.UUID) (models.Farm, error) {
	return f.farm, f.farmErr
}

func (f *fakeFarmRepo) GetFarmIDByFarmerID(_ context.Context, farmerID uuid.UUID) (uuid.UUID, error) {
	if f.farmIDErr != nil {
		return uuid.UUID{}, f.farmIDErr
	}
	if id, ok := f.farmIDByFarmer[farmerID]; ok {
		return id, nil
	}
	return uuid.UUID{}, ErrNotFound
}

func TestFarmService_GetFarm_OK(t *testing.T) {
	farmID := uuid.New()
	want := models.Farm{
		ID:                farmID,
		FarmerID:          uuid.New(),
		Name:              "Green Acres",
		Address:           "1 Farm Lane",
		Description:       "A small family farm",
		TotalSquareMeters: 1234.5,
	}
	svc := NewFarmService(&fakeFarmRepo{farm: want})

	got, err := svc.GetFarm(context.Background(), farmID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("farm = %+v, want %+v", got, want)
	}
}

func TestFarmService_GetFarm_NotFoundPropagates(t *testing.T) {
	svc := NewFarmService(&fakeFarmRepo{farmErr: ErrNotFound})

	_, err := svc.GetFarm(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
