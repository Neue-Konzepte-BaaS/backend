package services

import "math"

// weeksPerMonth converts a crop's duration in months to weeks, matching how
// a rental's total price is computed below.
const weeksPerMonth = 52.0 / 12.0

// ComputeRentalPriceCents is the one formula for what renting a plot with a
// crop costs, in EUR cents: (the plot's own base rate + the farm's rate for
// that crop) x the plot's area x the crop's duration in weeks. Both
// PlotSearchService (to show a customer what a listing costs) and
// PaymentService (to charge that exact amount) call this, so the two never
// drift apart.
func ComputeRentalPriceCents(basePriceCentsPerSqmPerWeek, farmCropRateCentsPerSqmPerWeek int32, areaSquareMeters float64, durationMonths int32) int32 {
	ratePerSqmPerWeek := float64(basePriceCentsPerSqmPerWeek + farmCropRateCentsPerSqmPerWeek)
	weeks := float64(durationMonths) * weeksPerMonth
	return int32(math.Round(ratePerSqmPerWeek * areaSquareMeters * weeks))
}
