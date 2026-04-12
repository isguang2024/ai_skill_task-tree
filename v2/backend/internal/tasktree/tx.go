package tasktree

import (
	"context"
	"database/sql"
)

type txContextKey struct{}

func (a *App) withTx(ctx context.Context, fn func(context.Context) error) error {
	if existing := txFromContext(ctx); existing != nil {
		return fn(ctx)
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	txCtx := context.WithValue(ctx, txContextKey{}, tx)
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func txFromContext(ctx context.Context) *sql.Tx {
	tx, _ := ctx.Value(txContextKey{}).(*sql.Tx)
	return tx
}

func (a *App) execContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if tx := txFromContext(ctx); tx != nil {
		return tx.ExecContext(ctx, query, args...)
	}
	return a.db.ExecContext(ctx, query, args...)
}

func (a *App) queryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if tx := txFromContext(ctx); tx != nil {
		return tx.QueryContext(ctx, query, args...)
	}
	return a.db.QueryContext(ctx, query, args...)
}

func (a *App) queryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	if tx := txFromContext(ctx); tx != nil {
		return tx.QueryRowContext(ctx, query, args...)
	}
	return a.db.QueryRowContext(ctx, query, args...)
}
