package repository

import (
	"context"
	"errors"
	"time"

	appquery "learning-video-recommendation-system/internal/recommendation/scheduler/application/query"
	apprepo "learning-video-recommendation-system/internal/recommendation/scheduler/application/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/mapper"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type userUnitStateRepository struct{}

func NewUserUnitStateRepository() apprepo.UserUnitStateRepository {
	return userUnitStateRepository{}
}

func (userUnitStateRepository) GetByUserAndUnit(ctx context.Context, q sqlcgen.Querier, userID uuid.UUID, coarseUnitID int64) (*model.UserUnitState, error) {
	row, err := q.GetUserUnitStateByUserAndUnit(ctx, sqlcgen.GetUserUnitStateByUserAndUnitParams{
		UserID:       mapper.UUIDToPG(userID),
		CoarseUnitID: coarseUnitID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	state, err := mapper.UserUnitStateFromRow(row)
	if err != nil {
		return nil, err
	}

	return &state, nil
}

func (userUnitStateRepository) Upsert(ctx context.Context, q sqlcgen.Querier, state *model.UserUnitState) error {
	params, err := mapper.UserUnitStateToUpsertParams(state)
	if err != nil {
		return err
	}

	return q.UpsertUserUnitState(ctx, params)
}

func (repo userUnitStateRepository) BatchUpsert(ctx context.Context, q sqlcgen.Querier, states []*model.UserUnitState) error {
	for _, state := range states {
		if err := repo.Upsert(ctx, q, state); err != nil {
			return err
		}
	}

	return nil
}

func (userUnitStateRepository) FindDueReviewCandidates(ctx context.Context, q sqlcgen.Querier, userID uuid.UUID, now time.Time) ([]appquery.ReviewCandidate, error) {
	rows, err := q.FindDueReviewCandidates(ctx, sqlcgen.FindDueReviewCandidatesParams{
		UserID: mapper.UUIDToPG(userID),
		Now:    mapper.TimeToPG(now),
	})
	if err != nil {
		return nil, err
	}

	return mapper.ReviewCandidatesFromRows(rows)
}

func (userUnitStateRepository) FindNewCandidates(ctx context.Context, q sqlcgen.Querier, userID uuid.UUID) ([]appquery.NewCandidate, error) {
	rows, err := q.FindNewCandidates(ctx, mapper.UUIDToPG(userID))
	if err != nil {
		return nil, err
	}

	return mapper.NewCandidatesFromRows(rows)
}
