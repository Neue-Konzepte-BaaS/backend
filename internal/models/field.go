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
	ID          uuid.UUID
	Name        string
	Field       uuid.UUID
	Coordinates *geom.Polygon
}

type FieldWithPlots struct {
	Field
	Plots []Plot
}
