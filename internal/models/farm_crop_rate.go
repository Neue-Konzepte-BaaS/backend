package models

import "github.com/google/uuid"

// FarmCropRate is one crop's farm-wide rental rate: what a farmer charges
// per square metre per week for that crop, regardless of which of the
// farm's plots grows it. Combined with a plot's own
// Plot.BasePriceCentsPerSqmPerWeek, it forms half of a PlotCropOffering's
// price.
type FarmCropRate struct {
	Crop                    uuid.UUID
	PriceCentsPerSqmPerWeek int32
}
