package db

import (
	"context"

	"github.com/uptrace/bun"
)

type transactionKey struct{}

func Executor(ctx context.Context, fallback *bun.DB) bun.IDB {
	if tx, ok := ctx.Value(transactionKey{}).(bun.Tx); ok {
		return tx
	}
	return fallback
}

func WithinTransaction(ctx context.Context, database *bun.DB, fn func(context.Context) error) error {
	if _, ok := ctx.Value(transactionKey{}).(bun.Tx); ok {
		return fn(ctx)
	}
	return database.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return fn(context.WithValue(ctx, transactionKey{}, tx))
	})
}
