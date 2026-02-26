package database

import (
	"context"
	"database/sql"
	"fmt"
)

// ctxKey is an unexported type used as the context key for transactions,
// preventing collisions with keys from other packages.
type ctxKey struct{}

// DBTX is the minimal interface satisfied by both *sql.DB and *sql.Tx,
// allowing repository methods to work transparently with either.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// WithTx stores a *sql.Tx in the returned context.
func WithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, ctxKey{}, tx)
}

// TxFromContext retrieves the *sql.Tx previously stored by WithTx.
// Returns nil if no transaction is present.
func TxFromContext(ctx context.Context) *sql.Tx {
	tx, _ := ctx.Value(ctxKey{}).(*sql.Tx)
	return tx
}

// Conn returns the DBTX to use: the context's transaction if present,
// otherwise the provided *sql.DB. Repositories should call this to
// obtain their executor.
func Conn(ctx context.Context, db *sql.DB) DBTX {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return db
}

// RunInTx executes fn inside a transaction. If fn returns an error the
// transaction is rolled back; otherwise it is committed.
func RunInTx(ctx context.Context, db *sql.DB, fn func(ctx context.Context) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	txCtx := WithTx(ctx, tx)

	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
