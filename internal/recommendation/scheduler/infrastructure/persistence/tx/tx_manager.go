package tx

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"
)

// TxManager provides a single transaction boundary for scheduler use cases.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, q sqlcgen.Querier) error) error
}
