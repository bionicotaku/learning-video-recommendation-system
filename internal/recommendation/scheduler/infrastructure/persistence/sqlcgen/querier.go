package sqlcgen

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// Querier is the stable query surface used by repositories and transaction helpers.
type Querier interface {
	CountSchedulerRuns(ctx context.Context) (int64, error)
	CountUnitLearningEvents(ctx context.Context) (int64, error)
	CountUserUnitStates(ctx context.Context) (int64, error)
	DeleteUserUnitStatesForReplay(ctx context.Context, arg DeleteUserUnitStatesForReplayParams) error
	FindDueReviewCandidates(ctx context.Context, arg FindDueReviewCandidatesParams) ([]FindDueReviewCandidatesRow, error)
	FindNewCandidates(ctx context.Context, userID pgtype.UUID) ([]FindNewCandidatesRow, error)
	FindUnitLearningEventsForReplay(ctx context.Context, arg FindUnitLearningEventsForReplayParams) ([]LearningUnitLearningEvent, error)
	GetUserSchedulerSettings(ctx context.Context, userID pgtype.UUID) (LearningUserSchedulerSetting, error)
	GetUserUnitStateByUserAndUnit(ctx context.Context, arg GetUserUnitStateByUserAndUnitParams) (LearningUserUnitState, error)
	InsertSchedulerRun(ctx context.Context, arg InsertSchedulerRunParams) error
	InsertSchedulerRunItem(ctx context.Context, arg InsertSchedulerRunItemParams) error
	InsertUnitLearningEvent(ctx context.Context, arg InsertUnitLearningEventParams) error
	UpsertUserUnitState(ctx context.Context, arg UpsertUserUnitStateParams) error
}
