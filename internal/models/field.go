package models

import (
	"github.com/google/uuid"
	geom "github.com/twpayne/go-geom"
)

type Field struct {
	ID          uuid.UUID
	Name        string
	Farm        uuid.UUID
	Coordinates *geom.Polygon
}

type Plot struct {
	ID               uuid.UUID
	Name             string
	Field            uuid.UUID
	Coordinates      *geom.Polygon
	AreaSquareMeters float64
	// BasePriceCentsPerSqmPerWeek is the farmer's own per-plot rate, nil
	// until set. It is one half of a crop's rental price -- see
	// PlotCropOffering -- the other half is the farm-wide FarmCropRate.
	BasePriceCentsPerSqmPerWeek *int32
}

type PlotWithCrops struct {
	Plot
	// Crops this plot currently offers, priced or not: the farmer needs to
	// see an unpriced one too, to know it is still missing a rate.
	Crops []Crop
}

type FieldWithPlots struct {
	Field
	Plots []PlotWithCrops
}

// PlotCropOffering is a crop as a customer sees it on a specific plot: what
// it is, plus the total price for renting that plot with it for its full
// duration. It only ever represents a crop that is actually priced -- see
// GetPricedCropOfferingsByPlots.
type PlotCropOffering struct {
	Crop
	PriceCents int32
}

type NearbyPlot struct {
	Plot
	Farm           uuid.UUID
	DistanceMeters float64
	// Crops this plot offers to a customer. Unlike PlotWithCrops.Crops, an
	// unpriced crop is left out entirely: from a customer's perspective it
	// is not an offer at all.
	Crops []PlotCropOffering
}
