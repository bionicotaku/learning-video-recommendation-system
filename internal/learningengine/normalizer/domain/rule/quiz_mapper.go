package rule

import (
	"fmt"

	"learning-video-recommendation-system/internal/learningengine/normalizer/domain/model"
	"learning-video-recommendation-system/internal/learningengine/normalizer/domain/policy"
	learningenum "learning-video-recommendation-system/internal/learningengine/reducer/domain/enum"
)

const SourceTypeQuizEvent = "quiz_event"

func MapQuizEvent(raw model.RawQuizEvent) (model.NormalizationResult, error) {
	if raw.EventID == "" {
		return model.Skipped("event_id is required"), nil
	}
	if raw.UserID == "" {
		return model.Skipped("user_id is required"), nil
	}
	if raw.CoarseUnitID == 0 {
		return model.Skipped("coarse_unit_id is required"), nil
	}
	if raw.CompletedAt.IsZero() {
		return model.Skipped("completed_at is required"), nil
	}
	if raw.TotalElapsedMS < 0 {
		return model.Skipped("total_elapsed_ms must be non-negative"), nil
	}

	quality := policy.QuizProgressQuality(raw.IsFirstTryCorrect, raw.TotalElapsedMS)
	isCorrect := raw.IsFirstTryCorrect
	metadata, err := marshalMetadata(map[string]any{
		"question_id":           raw.QuestionID,
		"trigger_type":          raw.TriggerType,
		"recommendation_run_id": raw.RecommendationRunID,
		"selected_option_ids":   raw.SelectedOptionIDs,
		"selection_interval_ms": raw.SelectionIntervalMS,
		"wrong_selection_count": wrongSelectionCount(raw.SelectedOptionIDs),
		"total_elapsed_ms":      raw.TotalElapsedMS,
		"shown_at":              raw.ShownAt,
		"completed_at":          raw.CompletedAt,
		"quality_policy": map[string]any{
			"name":                    "quiz_first_try_speed_v1",
			"quiz_speed_threshold_ms": policy.QuizSpeedThresholdMS,
		},
	})
	if err != nil {
		return model.NormalizationResult{}, fmt.Errorf("build quiz metadata: %w", err)
	}

	return model.Normalized(model.NormalizedLearningEvent{
		UserID:                    raw.UserID,
		CoarseUnitID:              raw.CoarseUnitID,
		VideoID:                   raw.VideoID,
		EventType:                 learningenum.EventQuiz,
		ReducerEffect:             learningenum.ReducerEffectAffectsProgress,
		SourceType:                SourceTypeQuizEvent,
		SourceRefID:               raw.EventID,
		IsCorrect:                 &isCorrect,
		ProgressQuality:           &quality,
		CountsTowardSuccessStreak: true,
		Metadata:                  metadata,
		OccurredAt:                raw.CompletedAt,
	}), nil
}

func wrongSelectionCount(selectedOptionIDs []string) int {
	count := 0
	for _, optionID := range selectedOptionIDs {
		if optionID != "correct" {
			count++
		}
	}
	return count
}
