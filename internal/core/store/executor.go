package store

import "context"

// RowScanner isolates scanning to free us from infrastructure frameworks.
type RowScanner interface {
	Scan(dest ...any) error
}

// RowsScanner extends RowScanner for collections requiring cursors.
type RowsScanner interface {
	RowScanner
	Next() bool
	Close() error
	Err() error
}

// Result abstracts insert/update/delete results.
type Result interface {
	RowsAffected() (int64, error)
}

// Querier provides the basic unified methods for interactions.
type Querier interface {
	Exec(ctx context.Context, query string, args ...any) (Result, error)
	Query(ctx context.Context, query string, args ...any) (RowsScanner, error)
	QueryRow(ctx context.Context, query string, args ...any) RowScanner
}
