package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	apprepo "learning-video-recommendation-system/internal/learningengine/reducer/application/repository"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
	"learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/mapper"
	learningenginesqlc "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/sqlcgen"
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

func (r *UserUnitStateRepository) GetByUserAndUnit(ctx context.Context, userID string, coarseUnitID int64) (*model.UserUnitState, error) {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return nil, err
	}

	row, err := r.queries.GetUserUnitState(ctx, learningenginesqlc.GetUserUnitStateParams{
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

func (r *UserUnitStateRepository) ListByUserAndUnitIDsForUpdate(ctx context.Context, userID string, coarseUnitIDs []int64) (map[int64]*model.UserUnitState, error) {
	result := make(map[int64]*model.UserUnitState, len(coarseUnitIDs))
	if len(coarseUnitIDs) == 0 {
		return result, nil
	}

	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.ListUserUnitStatesForUpdateByUnitIDs(ctx, learningenginesqlc.ListUserUnitStatesForUpdateByUnitIDsParams{
		UserID:        pgUserID,
		CoarseUnitIds: uniqueInt64s(coarseUnitIDs),
	})
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		mapped, err := mapper.ToUserUnitState(row)
		if err != nil {
			return nil, err
		}
		state := mapped
		result[state.CoarseUnitID] = &state
	}
	return result, nil
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
	if len(states) == 0 {
		return []*model.UserUnitState{}, nil
	}

	payload := make([]userUnitStatePayload, 0, len(states))
	for _, state := range states {
		payload = append(payload, toUserUnitStatePayload(state))
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.BatchUpsertUserUnitStates(ctx, encoded)
	if err != nil {
		return nil, err
	}

	result := make([]*model.UserUnitState, 0, len(rows))
	for _, row := range rows {
		mapped, err := mapper.ToUserUnitState(row)
		if err != nil {
			return nil, err
		}
		state := mapped
		result = append(result, &state)
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
		UserID:     pgUserID,
		OnlyTarget: filter.OnlyTarget,
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
	scheduleIntervalDays, err := mapper.Float64ToNumeric(state.ScheduleIntervalDays)
	if err != nil {
		return learningenginesqlc.UpsertUserUnitStateParams{}, err
	}
	scheduleEaseFactor, err := mapper.Float64ToNumeric(state.ScheduleEaseFactor)
	if err != nil {
		return learningenginesqlc.UpsertUserUnitStateParams{}, err
	}

	recentProgressQualities := state.RecentProgressQualities
	if recentProgressQualities == nil {
		recentProgressQualities = []int16{}
	}
	recentProgressPasses := state.RecentProgressPasses
	if recentProgressPasses == nil {
		recentProgressPasses = []bool{}
	}

	return learningenginesqlc.UpsertUserUnitStateParams{
		UserID:                        userID,
		CoarseUnitID:                  state.CoarseUnitID,
		IsTarget:                      state.IsTarget,
		TargetSource:                  mapper.StringToText(state.TargetSource),
		TargetSourceRefID:             mapper.StringToText(state.TargetSourceRefID),
		TargetPriority:                targetPriority,
		Status:                        state.Status,
		ProgressPercent:               progressPercent,
		MasteryScore:                  masteryScore,
		FirstObservedAt:               mapper.TimePointerToPG(state.FirstObservedAt),
		LastObservedAt:                mapper.TimePointerToPG(state.LastObservedAt),
		ObservationCount:              state.ObservationCount,
		ProgressEventCount:            state.ProgressEventCount,
		LastProgressAt:                mapper.TimePointerToPG(state.LastProgressAt),
		LastProgressQuality:           mapper.Int16PointerToPG(state.LastProgressQuality),
		RecentProgressQualities:       recentProgressQualities,
		RecentProgressPasses:          recentProgressPasses,
		ProgressSuccessCount:          state.ProgressSuccessCount,
		ProgressFailureCount:          state.ProgressFailureCount,
		ConsecutiveSuccessCount:       state.ConsecutiveSuccessCount,
		ConsecutiveFailureCount:       state.ConsecutiveFailureCount,
		ScheduleRepetition:            state.ScheduleRepetition,
		ScheduleIntervalDays:          scheduleIntervalDays,
		ScheduleEaseFactor:            scheduleEaseFactor,
		NextReviewAt:                  mapper.TimePointerToPG(state.NextReviewAt),
		LatestLearningEventOccurredAt: mapper.TimePointerToPG(state.LatestLearningEventOccurredAt),
		LatestResetBoundaryAt:         mapper.TimePointerToPG(state.LatestResetBoundaryAt),
		LatestLearningEventLedgerSeq:  state.LatestLearningEventLedgerSeq,
	}, nil
}

type userUnitStatePayload struct {
	UserID                        string     `json:"user_id"`
	CoarseUnitID                  int64      `json:"coarse_unit_id"`
	IsTarget                      bool       `json:"is_target"`
	TargetSource                  string     `json:"target_source"`
	TargetSourceRefID             string     `json:"target_source_ref_id"`
	TargetPriority                float64    `json:"target_priority"`
	Status                        string     `json:"status"`
	ProgressPercent               float64    `json:"progress_percent"`
	MasteryScore                  float64    `json:"mastery_score"`
	FirstObservedAt               *time.Time `json:"first_observed_at"`
	LastObservedAt                *time.Time `json:"last_observed_at"`
	ObservationCount              int32      `json:"observation_count"`
	ProgressEventCount            int32      `json:"progress_event_count"`
	LastProgressAt                *time.Time `json:"last_progress_at"`
	LastProgressQuality           *int16     `json:"last_progress_quality"`
	RecentProgressQualities       []int16    `json:"recent_progress_qualities"`
	RecentProgressPasses          []bool     `json:"recent_progress_passes"`
	ProgressSuccessCount          int32      `json:"progress_success_count"`
	ProgressFailureCount          int32      `json:"progress_failure_count"`
	ConsecutiveSuccessCount       int32      `json:"consecutive_success_count"`
	ConsecutiveFailureCount       int32      `json:"consecutive_failure_count"`
	ScheduleRepetition            int32      `json:"schedule_repetition"`
	ScheduleIntervalDays          float64    `json:"schedule_interval_days"`
	ScheduleEaseFactor            float64    `json:"schedule_ease_factor"`
	NextReviewAt                  *time.Time `json:"next_review_at"`
	LatestLearningEventOccurredAt *time.Time `json:"latest_learning_event_occurred_at"`
	LatestResetBoundaryAt         *time.Time `json:"latest_reset_boundary_at"`
	LatestLearningEventLedgerSeq  int64      `json:"latest_learning_event_ledger_seq"`
}

func toUserUnitStatePayload(state *model.UserUnitState) userUnitStatePayload {
	recentProgressQualities := state.RecentProgressQualities
	if recentProgressQualities == nil {
		recentProgressQualities = []int16{}
	}
	recentProgressPasses := state.RecentProgressPasses
	if recentProgressPasses == nil {
		recentProgressPasses = []bool{}
	}

	return userUnitStatePayload{
		UserID:                        state.UserID,
		CoarseUnitID:                  state.CoarseUnitID,
		IsTarget:                      state.IsTarget,
		TargetSource:                  state.TargetSource,
		TargetSourceRefID:             state.TargetSourceRefID,
		TargetPriority:                state.TargetPriority,
		Status:                        state.Status,
		ProgressPercent:               state.ProgressPercent,
		MasteryScore:                  state.MasteryScore,
		FirstObservedAt:               utcPointer(state.FirstObservedAt),
		LastObservedAt:                utcPointer(state.LastObservedAt),
		ObservationCount:              state.ObservationCount,
		ProgressEventCount:            state.ProgressEventCount,
		LastProgressAt:                utcPointer(state.LastProgressAt),
		LastProgressQuality:           state.LastProgressQuality,
		RecentProgressQualities:       recentProgressQualities,
		RecentProgressPasses:          recentProgressPasses,
		ProgressSuccessCount:          state.ProgressSuccessCount,
		ProgressFailureCount:          state.ProgressFailureCount,
		ConsecutiveSuccessCount:       state.ConsecutiveSuccessCount,
		ConsecutiveFailureCount:       state.ConsecutiveFailureCount,
		ScheduleRepetition:            state.ScheduleRepetition,
		ScheduleIntervalDays:          state.ScheduleIntervalDays,
		ScheduleEaseFactor:            state.ScheduleEaseFactor,
		NextReviewAt:                  utcPointer(state.NextReviewAt),
		LatestLearningEventOccurredAt: utcPointer(state.LatestLearningEventOccurredAt),
		LatestResetBoundaryAt:         utcPointer(state.LatestResetBoundaryAt),
		LatestLearningEventLedgerSeq:  state.LatestLearningEventLedgerSeq,
	}
}

func utcPointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func uniqueInt64s(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
