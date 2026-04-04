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
	"github.com/jackc/pgx/v5/pgtype"
)

type userUnitStateRepository struct {
	querier sqlcgen.Querier
}

func NewUserUnitStateRepository(querier sqlcgen.Querier) apprepo.UserUnitStateRepository {
	return userUnitStateRepository{querier: querier}
}

func (r userUnitStateRepository) GetByUserAndUnit(ctx context.Context, userID uuid.UUID, coarseUnitID int64) (*model.UserUnitState, error) {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return nil, err
	}

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

func (r userUnitStateRepository) Upsert(ctx context.Context, state *model.UserUnitState) error {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return err
	}

	params, err := mapper.UserUnitStateToUpsertParams(state)
	if err != nil {
		return err
	}

	return q.UpsertUserUnitState(ctx, params)
}

func (repo userUnitStateRepository) BatchUpsert(ctx context.Context, states []*model.UserUnitState) error {
	for _, state := range states {
		if err := repo.Upsert(ctx, state); err != nil {
			return err
		}
	}

	return nil
}

func (r userUnitStateRepository) DeleteForReplay(ctx context.Context, userID uuid.UUID, coarseUnitID *int64) error {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return err
	}

	var coarseUnitParam pgtype.Int8
	if coarseUnitID != nil {
		coarseUnitParam = pgtype.Int8{Int64: *coarseUnitID, Valid: true}
	}

	return q.DeleteUserUnitStatesForReplay(ctx, sqlcgen.DeleteUserUnitStatesForReplayParams{
		UserID:       mapper.UUIDToPG(userID),
		CoarseUnitID: coarseUnitParam,
	})
}

func (r userUnitStateRepository) FindDueReviewCandidates(ctx context.Context, userID uuid.UUID, now time.Time) ([]appquery.ReviewCandidate, error) {
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

func (r userUnitStateRepository) FindNewCandidates(ctx context.Context, userID uuid.UUID) ([]appquery.NewCandidate, error) {
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
