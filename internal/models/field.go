package models

import (
	"github.com/google/uuid"
	geom "github.com/twpayne/go-geom"
)

type Field struct {
	ID          uuid.UUID
	Name        string
	Farmer      uuid.UUID
	Coordinates *geom.Polygon
}

type Plot struct {
	ID               uuid.UUID
	Name             string
	Field            uuid.UUID
	Coordinates      *geom.Polygon
	AreaSquareMeters float64
}

type PlotWithCrops struct {
	Plot
	// Crops this plot currently offers.
	Crops []Crop
}

type FieldWithPlots struct {
	Field
	Plots []PlotWithCrops
}

type NearbyPlot struct {
	Plot
	DistanceMeters float64
	// Crops this plot currently offers.
	Crops []Crop
}
