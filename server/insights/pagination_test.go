package insights

import (
	"testing"
)

// Adapted from server/public/model/insights_test.go in commit 26617fcbdc; the
// generic form replaces the per-type GetTop*ListWithPagination helpers.

func TestPaginate_hasOnePage(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6}
	page, hasNext := Paginate(items, len(items))
	if hasNext {
		t.Fatalf("hasNext = true; want false when len(items) <= perPage")
	}
	if len(page) != len(items) {
		t.Fatalf("page length = %d; want %d", len(page), len(items))
	}
}

func TestPaginate_hasMoreThanOnePage(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7}
	page, hasNext := Paginate(items, len(items)-1)
	if !hasNext {
		t.Fatalf("hasNext = false; want true when len(items) > perPage")
	}
	if len(page) != len(items)-1 {
		t.Fatalf("page length = %d; want %d", len(page), len(items)-1)
	}
}

func TestPaginate_emptyOrZeroPerPage(t *testing.T) {
	page, hasNext := Paginate([]int{}, 5)
	if hasNext || len(page) != 0 {
		t.Fatalf("empty input: page=%v hasNext=%v", page, hasNext)
	}

	page, hasNext = Paginate([]int{1, 2, 3}, 0)
	if hasNext {
		t.Fatalf("perPage=0 should not signal next page")
	}
	if len(page) != 3 {
		t.Fatalf("perPage=0 should return all rows untouched")
	}
}
