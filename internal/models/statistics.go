package models

import "time"

// StatisticsScope says what a Statistics payload covers: one farmer's own
// holdings, or the whole platform. It is derived from the caller's role, never
// chosen by the caller.
type StatisticsScope string

const (
	ScopeFarm     StatisticsScope = "farm"
	ScopePlatform StatisticsScope = "platform"
)

// Statistics is the whole dashboard payload. A new statistic is a CTE in
// sql/queries/statistics.sql plus a field here; nothing in between changes.
type Statistics struct {
	Scope       StatisticsScope
	GeneratedAt time.Time
	Fields      FieldStatistics
	Plots       PlotStatistics
	Rentals     RentalStatistics
	// Accounts is platform scope only: a farmer has no business seeing how
	// many customers exist. Nil for ScopeFarm.
	Accounts *AccountStatistics
}

type FieldStatistics struct {
	Total            int64
	AreaSquareMeters float64
}

// Available and OccupancyRate are derived by the service rather than queried;
// see withDerivedStatistics. OccupancyRate is a ratio in [0, 1].
type PlotStatistics struct {
	Total            int64
	Rented           int64 // plots with a rental covering the present moment
	Available        int64
	AreaSquareMeters float64
	OccupancyRate    float64
}

type RentalStatistics struct {
	Total      int64
	Active     int64
	Last30Days int64
}

type AccountStatistics struct {
	Total                int64
	Farmers              int64
	Customers            int64
	RegisteredLast30Days int64
}
