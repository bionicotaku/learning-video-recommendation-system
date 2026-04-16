package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	apprepo "learning-video-recommendation-system/internal/learningengine/application/repository"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/mapper"
	learningenginesqlc "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/sqlcgen"
)

type UserUnitStateRepository struct {
	queries *learningenginesqlc.Queries
}

var _ apprepo.UserUnitStateRepository = (*UserUnitStateRepository)(nil)

func NewUserUnitStateRepository(db learningenginesqlc.DBTX) *UserUnitStateRepository {
	return &UserUnitStateRepository{
		queries: learningenginesqlc.New(db),
	}
}

func (r *UserUnitStateRepository) GetByUserAndUnitForUpdate(ctx context.Context, userID string, coarseUnitID int64) (*model.UserUnitState, error) {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return nil, err
	}

	row, err := r.queries.GetUserUnitStateForUpdate(ctx, learningenginesqlc.GetUserUnitStateForUpdateParams{
		UserID:       pgUserID,
		CoarseUnitID: coarseUnitID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	state, err := mapper.ToUserUnitState(row)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *UserUnitStateRepository) Upsert(ctx context.Context, state *model.UserUnitState) (*model.UserUnitState, error) {
	params, err := toUpsertUserUnitStateParams(state)
	if err != nil {
		return nil, err
	}

	row, err := r.queries.UpsertUserUnitState(ctx, params)
	if err != nil {
		return nil, err
	}

	mapped, err := mapper.ToUserUnitState(row)
	if err != nil {
		return nil, err
	}
	return &mapped, nil
}

func (r *UserUnitStateRepository) BatchUpsert(ctx context.Context, states []*model.UserUnitState) ([]*model.UserUnitState, error) {
	result := make([]*model.UserUnitState, 0, len(states))
	for _, state := range states {
		mapped, err := r.Upsert(ctx, state)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}

func (r *UserUnitStateRepository) DeleteByUser(ctx context.Context, userID string) error {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return err
	}

	return r.queries.DeleteUserUnitStatesByUser(ctx, pgUserID)
}

func (r *UserUnitStateRepository) ListByUser(ctx context.Context, userID string, filter model.UserUnitStateFilter) ([]model.UserUnitState, error) {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.ListUserUnitStates(ctx, learningenginesqlc.ListUserUnitStatesParams{
		UserID:           pgUserID,
		OnlyTarget:       filter.OnlyTarget,
		ExcludeSuspended: filter.ExcludeSuspended,
	})
	if err != nil {
		return nil, err
	}

	result := make([]model.UserUnitState, 0, len(rows))
	for _, row := range rows {
		mapped, err := mapper.ToUserUnitState(row)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}

func toUpsertUserUnitStateParams(state *model.UserUnitState) (learningenginesqlc.UpsertUserUnitStateParams, error) {
	userID, err := mapper.StringToUUID(state.UserID)
	if err != nil {
		return learningenginesqlc.UpsertUserUnitStateParams{}, err
	}
	targetPriority, err := mapper.Float64ToNumeric(state.TargetPriority)
	if err != nil {
		return learningenginesqlc.UpsertUserUnitStateParams{}, err
	}
	progressPercent, err := mapper.Float64ToNumeric(state.ProgressPercent)
	if err != nil {
		return learningenginesqlc.UpsertUserUnitStateParams{}, err
	}
	masteryScore, err := mapper.Float64ToNumeric(state.MasteryScore)
	if err != nil {
		return learningenginesqlc.UpsertUserUnitStateParams{}, err
	}
	intervalDays, err := mapper.Float64ToNumeric(state.IntervalDays)
	if err != nil {
		return learningenginesqlc.UpsertUserUnitStateParams{}, err
	}
	easeFactor, err := mapper.Float64ToNumeric(state.EaseFactor)
	if err != nil {
		return learningenginesqlc.UpsertUserUnitStateParams{}, err
	}

	recentQualityWindow := state.RecentQualityWindow
	if recentQualityWindow == nil {
		recentQualityWindow = []int16{}
	}
	recentCorrectnessWindow := state.RecentCorrectnessWindow
	if recentCorrectnessWindow == nil {
		recentCorrectnessWindow = []bool{}
	}

	return learningenginesqlc.UpsertUserUnitStateParams{
		UserID:                  userID,
		CoarseUnitID:            state.CoarseUnitID,
		IsTarget:                state.IsTarget,
		TargetSource:            mapper.StringToText(state.TargetSource),
		TargetSourceRefID:       mapper.StringToText(state.TargetSourceRefID),
		TargetPriority:          targetPriority,
		Status:                  state.Status,
		ProgressPercent:         progressPercent,
		MasteryScore:            masteryScore,
		FirstSeenAt:             mapper.TimePointerToPG(state.FirstSeenAt),
		LastSeenAt:              mapper.TimePointerToPG(state.LastSeenAt),
		LastReviewedAt:          mapper.TimePointerToPG(state.LastReviewedAt),
		SeenCount:               state.SeenCount,
		StrongEventCount:        state.StrongEventCount,
		ReviewCount:             state.ReviewCount,
		CorrectCount:            state.CorrectCount,
		WrongCount:              state.WrongCount,
		ConsecutiveCorrect:      state.ConsecutiveCorrect,
		ConsecutiveWrong:        state.ConsecutiveWrong,
		LastQuality:             mapper.Int16PointerToPG(state.LastQuality),
		RecentQualityWindow:     recentQualityWindow,
		RecentCorrectnessWindow: recentCorrectnessWindow,
		Repetition:              state.Repetition,
		IntervalDays:            intervalDays,
		EaseFactor:              easeFactor,
		NextReviewAt:            mapper.TimePointerToPG(state.NextReviewAt),
		SuspendedReason:         mapper.StringToText(state.SuspendedReason),
	}, nil
}
