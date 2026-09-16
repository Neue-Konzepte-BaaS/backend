package services

// Pagination bounds for every listing endpoint. The handler rejects a limit
// outside the range outright, so clampPagination is the second line of defence
// for callers that reach a service some other way -- and the one place that
// decides what an unset limit means.
const (
	DefaultPageLimit = 20
	MaxPageLimit     = 100
)

// clampPagination turns whatever it is given into a usable window: an unset or
// nonsensical limit becomes the default, an oversized one is capped, and a
// negative offset becomes zero. Clamping rather than erroring keeps a service
// call from failing on a bound the caller never set.
func clampPagination(limit, offset int32) (int32, int32) {
	if limit <= 0 {
		limit = DefaultPageLimit
	}
	if limit > MaxPageLimit {
		limit = MaxPageLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
