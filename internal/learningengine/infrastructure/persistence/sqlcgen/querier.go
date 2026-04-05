package sqlcgen

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// Querier is the stable query surface used by repositories and transaction helpers.
type Querier interface {
	CountUnitLearningEvents(ctx context.Context) (int64, error)
	CountUserUnitStates(ctx context.Context) (int64, error)
	DeleteUserUnitStatesByUser(ctx context.Context, userID pgtype.UUID) error
	GetUserUnitStateByUserAndUnit(ctx context.Context, arg GetUserUnitStateByUserAndUnitParams) (LearningUserUnitState, error)
	InsertUnitLearningEvent(ctx context.Context, arg InsertUnitLearningEventParams) error
	ListUnitLearningEventsByUserOrdered(ctx context.Context, userID pgtype.UUID) ([]LearningUnitLearningEvent, error)
	UpsertUserUnitState(ctx context.Context, arg UpsertUserUnitStateParams) error
}
