package models

// Page is one window onto a longer list, together with the total number of
// rows the filter matched. Total is what the window is a window onto, so it
// counts every match, not just the ones on this page.
//
// Total comes from the same query -- and therefore the same snapshot -- as
// Items, so the two can never disagree; see the COUNT(*) OVER () in the admin
// listing queries.
type Page[T any] struct {
	Items  []T
	Total  int64
	Limit  int32
	Offset int32
}
