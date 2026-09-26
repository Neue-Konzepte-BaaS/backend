package services

import (
	"math"
	"testing"
)

func TestComputeRentalPriceCents(t *testing.T) {
	tests := []struct {
		name                           string
		basePriceCentsPerSqmPerWeek    int32
		farmCropRateCentsPerSqmPerWeek int32
		areaSquareMeters               float64
		durationMonths                 int32
		want                           int32
	}{
		{
			name:                           "one sqm, one week-equivalent month, 1 cent rates",
			basePriceCentsPerSqmPerWeek:    1,
			farmCropRateCentsPerSqmPerWeek: 1,
			areaSquareMeters:               1,
			durationMonths:                 12, // 52 weeks exactly
			want:                           2 * 52,
		},
		{
			name:                           "rounds to the nearest cent",
			basePriceCentsPerSqmPerWeek:    10,
			farmCropRateCentsPerSqmPerWeek: 5,
			areaSquareMeters:               10,
			durationMonths:                 1, // 52/12 weeks
			want:                           int32(math.Round(15 * 10 * (52.0 / 12.0))),
		},
		{
			name:                           "zero area is zero price",
			basePriceCentsPerSqmPerWeek:    100,
			farmCropRateCentsPerSqmPerWeek: 50,
			areaSquareMeters:               0,
			durationMonths:                 6,
			want:                           0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeRentalPriceCents(tt.basePriceCentsPerSqmPerWeek, tt.farmCropRateCentsPerSqmPerWeek, tt.areaSquareMeters, tt.durationMonths)
			if got != tt.want {
				t.Errorf("got = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestComputeRentalPriceCents_MatchesTheDocumentedFormula(t *testing.T) {
	// A round-trip against the formula spelled out in the doc comment:
	// (base + farm rate) x area x (months x 52/12), independently
	// computed here rather than reusing the implementation.
	const base, farmRate int32 = 250, 100
	const area = 37.5
	const months int32 = 4

	weeks := float64(months) * 52.0 / 12.0
	expected := int32(math.Round(float64(base+farmRate) * area * weeks))

	got := ComputeRentalPriceCents(base, farmRate, area, months)
	if got != expected {
		t.Errorf("got = %d, want %d", got, expected)
	}
}
