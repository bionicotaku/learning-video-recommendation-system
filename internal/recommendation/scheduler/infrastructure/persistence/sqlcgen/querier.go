package sqlcgen

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// Querier is the stable query surface used by repositories and transaction helpers.
type Querier interface {
	CountSchedulerRuns(ctx context.Context) (int64, error)
	FindDueReviewCandidates(ctx context.Context, arg FindDueReviewCandidatesParams) ([]FindDueReviewCandidatesRow, error)
	FindNewCandidates(ctx context.Context, userID pgtype.UUID) ([]FindNewCandidatesRow, error)
	UpsertSchedulerRun(ctx context.Context, arg UpsertSchedulerRunParams) error
	UpsertSchedulerRunItem(ctx context.Context, arg UpsertSchedulerRunItemParams) error
	UpsertUserUnitServingState(ctx context.Context, arg UpsertUserUnitServingStateParams) error
}
