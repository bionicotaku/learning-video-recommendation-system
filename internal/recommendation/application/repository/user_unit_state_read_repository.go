package repository

import (
	"context"
	"time"

	"learning-video-recommendation-system/internal/recommendation/application/query"

	"github.com/google/uuid"
)

// UserUnitStateReadRepository only exposes candidate reads from Learning engine data.
type UserUnitStateReadRepository interface {
	FindDueReviewCandidates(ctx context.Context, userID uuid.UUID, now time.Time) ([]query.ReviewCandidate, error)
	FindNewCandidates(ctx context.Context, userID uuid.UUID) ([]query.NewCandidate, error)
}
