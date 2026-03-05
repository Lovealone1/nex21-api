package store

type SortDirection string

const (
	SortAsc  SortDirection = "ASC"
	SortDesc SortDirection = "DESC"
)

// Sort dictates order evading raw strings to prevent injections.
type Sort struct {
	Field     string
	Direction SortDirection
}

type Page struct {
	Limit  int
	Offset int
	Sorts  []Sort
}

type ResultList[T any] struct {
	Items []T
	Total int64
	Page  Page
}
