package repository

import (
	"context"
	"time"

	appquery "learning-video-recommendation-system/internal/recommendation/scheduler/application/query"
	apprepo "learning-video-recommendation-system/internal/recommendation/scheduler/application/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/mapper"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"

	"github.com/google/uuid"
)

type userUnitStateReadRepository struct {
	querier sqlcgen.Querier
}

func NewLearningStateSnapshotReadRepository(querier sqlcgen.Querier) apprepo.LearningStateSnapshotReadRepository {
	return userUnitStateReadRepository{querier: querier}
}

func (r userUnitStateReadRepository) FindDueReviewCandidates(ctx context.Context, userID uuid.UUID, now time.Time) ([]appquery.ReviewCandidate, error) {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return nil, err
	}

	rows, err := q.FindDueReviewCandidates(ctx, sqlcgen.FindDueReviewCandidatesParams{
		UserID: mapper.UUIDToPG(userID),
		Now:    mapper.TimeToPG(now),
	})
	if err != nil {
		return nil, err
	}

	return mapper.ReviewCandidatesFromRows(rows)
}

func (r userUnitStateReadRepository) FindNewCandidates(ctx context.Context, userID uuid.UUID) ([]appquery.NewCandidate, error) {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return nil, err
	}

	rows, err := q.FindNewCandidates(ctx, mapper.UUIDToPG(userID))
	if err != nil {
		return nil, err
	}

	return mapper.NewCandidatesFromRows(rows)
}
