package mapper

import (
	"encoding/json"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"
)

// SchedulerRunParamsFromBatch maps a recommendation batch to scheduler_runs insert params.
func SchedulerRunParamsFromBatch(batch model.RecommendationBatch) (sqlcgen.InsertSchedulerRunParams, error) {
	selectedReviewCount := 0
	selectedNewCount := 0
	for _, item := range batch.Items {
		switch item.RecommendType {
		case enum.RecommendTypeReview:
			selectedReviewCount++
		case enum.RecommendTypeNew:
			selectedNewCount++
		}
	}

	contextPayload, err := json.Marshal(map[string]any{
		"backlog_protection": batch.BacklogProtection,
	})
	if err != nil {
		return sqlcgen.InsertSchedulerRunParams{}, err
	}

	return sqlcgen.InsertSchedulerRunParams{
		RunID:               UUIDToPG(batch.RunID),
		UserID:              UUIDToPG(batch.UserID),
		RequestedLimit:      int32(batch.SessionLimit),
		GeneratedAt:         TimeToPG(batch.GeneratedAt),
		DueReviewCount:      int32(selectedReviewCount),
		SelectedReviewCount: int32(selectedReviewCount),
		SelectedNewCount:    int32(selectedNewCount),
		Context:             contextPayload,
	}, nil
}

// SchedulerRunItemParamsFromBatch maps recommendation items to scheduler_run_items insert params.
func SchedulerRunItemParamsFromBatch(batch model.RecommendationBatch) ([]sqlcgen.InsertSchedulerRunItemParams, error) {
	params := make([]sqlcgen.InsertSchedulerRunItemParams, 0, len(batch.Items))
	for _, item := range batch.Items {
		score, err := floatToPG(item.Score)
		if err != nil {
			return nil, err
		}

		params = append(params, sqlcgen.InsertSchedulerRunItemParams{
			RunID:         UUIDToPG(batch.RunID),
			UserID:        UUIDToPG(batch.UserID),
			CoarseUnitID:  item.CoarseUnitID,
			RecommendType: string(item.RecommendType),
			Rank:          int32(item.Rank),
			Score:         score,
			ReasonCodes:   item.ReasonCodes,
		})
	}

	return params, nil
}
