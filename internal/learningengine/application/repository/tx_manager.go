package repository

import "context"

// TxManager defines the application-facing transaction boundary.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
