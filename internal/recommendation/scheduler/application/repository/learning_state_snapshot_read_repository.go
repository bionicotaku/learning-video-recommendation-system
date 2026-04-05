package repository

import (
	"context"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/application/query"

	"github.com/google/uuid"
)

// LearningStateSnapshotReadRepository only exposes candidate reads from Learning engine data.
type LearningStateSnapshotReadRepository interface {
	FindDueReviewCandidates(ctx context.Context, userID uuid.UUID, now time.Time) ([]query.ReviewCandidate, error)
	FindNewCandidates(ctx context.Context, userID uuid.UUID) ([]query.NewCandidate, error)
}
