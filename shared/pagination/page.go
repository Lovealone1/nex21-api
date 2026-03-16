// Package pagination provides shared HTTP utilities for cursor-less
// offset/limit pagination. It is designed to be imported by any HTTP
// transport layer that needs to parse incoming query parameters and
// send back a consistent envelope response.
//
// The package is intentionally domain-agnostic: it does NOT define any
// global list of sortable column names. Each caller is responsible for
// passing its own allowlist of valid sort fields to ParseRequest.
package pagination

import (
	"net/http"
	"strconv"

	"github.com/Lovealone1/nex21-api/internal/core/store"
)

const (
	defaultLimit     = 20
	maxLimit         = 100
	defaultSortField = "created_at"
)

// PagedResponse is the standard JSON envelope returned for every list endpoint.
// Items is typed via generics so it can carry any domain object.
type PagedResponse[T any] struct {
	// Items contains the slice of results for the current page.
	Items []T `json:"items"`
	// Total is the total number of records across all pages.
	Total int64 `json:"total"`
	// Page is the current page number (1-based).
	Page int `json:"page"`
	// Limit is the number of items per page.
	Limit int `json:"limit"`
	// TotalPages is the total number of pages available.
	TotalPages int64 `json:"total_pages"`
}

// ParseRequest extracts page, limit, sort_by and sort_dir from query parameters
// and returns a validated store.Page ready to use in repository queries.
//
// allowedFields is the caller-defined allowlist of columns that may be used for
// ORDER BY. If sort_by does not appear in allowedFields it is silently ignored
// and the default sort (created_at DESC) is applied.
//
// Query parameters:
//
//	page      int    — 1-based page number (default: 1)
//	limit     int    — records per page (default: 20, max: 100)
//	sort_by   string — must be in allowedFields
//	sort_dir  string — ASC | DESC (default: DESC)
func ParseRequest(r *http.Request, allowedFields ...string) store.Page {
	q := r.URL.Query()

	// ─ page ─────────────────────────────────────────────────────────────────
	page := 1
	if v := q.Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}

	// ─ limit ─────────────────────────────────────────────────────────────────
	limit := defaultLimit
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > maxLimit {
				n = maxLimit
			}
			limit = n
		}
	}

	offset := (page - 1) * limit

	// ─ sort ──────────────────────────────────────────────────────────────────
	// Build allowlist from caller-provided fields — the shared package knows
	// nothing about domain column names.
	allowed := make(map[string]bool, len(allowedFields))
	for _, f := range allowedFields {
		allowed[f] = true
	}

	var sorts []store.Sort
	if sortBy := q.Get("sort_by"); sortBy != "" && allowed[sortBy] {
		dir := store.SortDesc
		if sd := q.Get("sort_dir"); sd == "ASC" || sd == "asc" {
			dir = store.SortAsc
		}
		sorts = append(sorts, store.Sort{Field: sortBy, Direction: dir})
	}

	// Default: newest first.
	if len(sorts) == 0 {
		sorts = []store.Sort{{Field: defaultSortField, Direction: store.SortDesc}}
	}

	return store.Page{
		Limit:  limit,
		Offset: offset,
		Sorts:  sorts,
	}
}

// NewResponse wraps a ResultList into the standard PagedResponse envelope.
func NewResponse[T any](result store.ResultList[T], page store.Page) PagedResponse[T] {
	currentPage := 1
	if page.Limit > 0 {
		currentPage = (page.Offset / page.Limit) + 1
	}

	totalPages := int64(1)
	if page.Limit > 0 && result.Total > 0 {
		totalPages = (result.Total + int64(page.Limit) - 1) / int64(page.Limit)
	}

	items := result.Items
	if items == nil {
		items = []T{} // always return an array, never null
	}

	return PagedResponse[T]{
		Items:      items,
		Total:      result.Total,
		Page:       currentPage,
		Limit:      page.Limit,
		TotalPages: totalPages,
	}
}
