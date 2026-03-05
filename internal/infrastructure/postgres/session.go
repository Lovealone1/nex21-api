package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Lovealone1/nex21-api/internal/core/errors"
	"github.com/Lovealone1/nex21-api/internal/core/store"
)

// sqlQuerier wraps standard database/sql to comply with our abstract store.Querier interface
type sqlQuerier struct {
	q dbExecutor // interface satisfied by *sql.DB and *sql.Tx
}

type dbExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (sq *sqlQuerier) Exec(ctx context.Context, query string, args ...any) (store.Result, error) {
	res, err := sq.q.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, mapError(err)
	}
	return res, nil // sql.Result conforms to store.Result abstracting RowsAffected
}

func (sq *sqlQuerier) Query(ctx context.Context, query string, args ...any) (store.RowsScanner, error) {
	rows, err := sq.q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, mapError(err)
	}
	return rows, nil // *sql.Rows conforms to store.RowsScanner
}

func (sq *sqlQuerier) QueryRow(ctx context.Context, query string, args ...any) store.RowScanner {
	return sq.q.QueryRowContext(ctx, query, args...)
}

// mapError translates SQL errors to our domain errors
func mapError(err error) error {
	if err == sql.ErrNoRows {
		return errors.ErrNotFound
	}
	// For other postgres specific errors like 23505 (unique_violation),
	// we would check the pq/pgconn error code here. To keep it generic:
	return fmt.Errorf("%w: %v", errors.ErrInternal, err)
}
