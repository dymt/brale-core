package position

import (
	"context"

	"brale-core/internal/store"
)

func withinStoreTx(ctx context.Context, st store.Store, fn func(context.Context) error) error {
	if txRunner, ok := st.(store.TxRunner); ok {
		return txRunner.WithinTx(ctx, fn)
	}
	return fn(ctx)
}
