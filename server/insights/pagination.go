package insights

// Paginate trims a result set that was queried with a `LIMIT perPage+1` to the
// requested per-page size, returning a `hasNext` flag indicating whether more
// rows exist beyond the returned slice.
//
// The query layer asks for one row past the requested page so callers can
// distinguish "exactly per_page rows" (no next page) from "per_page rows and at
// least one more" (next page exists) without a separate COUNT query.
func Paginate[T any](items []T, perPage int) (page []T, hasNext bool) {
	if perPage <= 0 || len(items) <= perPage {
		return items, false
	}
	return items[:perPage], true
}
