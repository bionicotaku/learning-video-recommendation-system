package repository

import (
	"context"
	"time"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type LearningStateVersion struct {
	ActiveTargetUnitCount      int32
	SourceLearningMaxUpdatedAt *time.Time
}

type RecallQueueRepository interface {
	GetLearningStateVersion(ctx context.Context, userID string) (LearningStateVersion, error)
	GetProjectionUpdatedAt(ctx context.Context) (time.Time, error)
	GetQueueState(ctx context.Context, userID string) (model.RecallQueueState, bool, error)
	RebuildUserQueue(ctx context.Context, userID string, projectionUpdatedAt time.Time) (model.RecallQueueState, error)
	ListCandidates(ctx context.Context, userID string, now time.Time, suppliedPerBucketLimit int32, noSupplyPerBucketLimit int32) ([]model.RecallQueueCandidate, error)
}
