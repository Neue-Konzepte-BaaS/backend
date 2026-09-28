package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// GetFieldsByFarm is the public, customer-facing counterpart to
// GetFieldsWithPlots: no farmer lookup, just a farm id straight through to
// the repository — a customer browsing a farm's plots may call this for any
// farm, not only their own.
func TestFieldService_GetFieldsByFarm_OK(t *testing.T) {
	farmID := uuid.New()
	want := []models.FieldWithPlotStats{
		{
			Field:            models.Field{ID: uuid.New(), Name: "North Field", Farm: farmID},
			PlotCount:        3,
			AreaSquareMeters: 900,
		},
		{
			// A field with zero currently-available plots still gets a row,
			// not silence — see the query's own doc comment.
			Field:            models.Field{ID: uuid.New(), Name: "South Field", Farm: farmID},
			PlotCount:        0,
			AreaSquareMeters: 0,
		},
	}
	repo := &fakeFieldRepo{fieldsWithStats: want}
	svc := NewFieldService(&fakeFarmRepo{}, repo, &fakePlotRepo{}, &fakeCropRepo{})

	got, err := svc.GetFieldsByFarm(context.Background(), farmID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d fields, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("field[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestFieldService_GetFieldsByFarm_UnknownFarmYieldsNoFields(t *testing.T) {
	repo := &fakeFieldRepo{fieldsWithStats: nil}
	svc := NewFieldService(&fakeFarmRepo{}, repo, &fakePlotRepo{}, &fakeCropRepo{})

	got, err := svc.GetFieldsByFarm(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d fields, want 0", len(got))
	}
}

func TestFieldService_GetFieldsByFarm_WrapsRepositoryErrors(t *testing.T) {
	sentinel := errors.New("connection refused")
	repo := &fakeFieldRepo{fieldsWithStatsErr: sentinel}
	svc := NewFieldService(&fakeFarmRepo{}, repo, &fakePlotRepo{}, &fakeCropRepo{})

	_, err := svc.GetFieldsByFarm(context.Background(), uuid.New())
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want it to wrap %v", err, sentinel)
	}
}
