// Package store reads and writes the metadata service's data in Postgres.
//
// All SQL lives in this package, and it knows nothing about gRPC: it takes
// and returns plain Go values. The server package turns those into API
// responses.
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store is the service's connection to Postgres.
type Store struct {
	// A pool keeps a few connections open and reuses them across requests,
	// instead of opening a new connection for every call.
	db *pgxpool.Pool
}

// Connect opens a connection pool to the database at url and checks that
// it works, so the service fails at startup instead of on the first request.
func Connect(ctx context.Context, url string) (*Store, error) {
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("connect to db: %w", err)
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &Store{db: db}, nil
}

// Close closes every connection in the pool.
func (s *Store) Close() {
	s.db.Close()
}

// querier is what both the pool and a transaction can do, so a helper can
// run inside or outside a transaction.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
