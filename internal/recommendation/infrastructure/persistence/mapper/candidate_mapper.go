package mapper

import (
	"fmt"
	"time"

	"learning-video-recommendation-system/internal/recommendation/application/query"
	"learning-video-recommendation-system/internal/recommendation/domain/enum"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

func parseUnitKind(value string) (enum.UnitKind, error) {
	kind := enum.UnitKind(value)
	switch kind {
	case enum.UnitKindWord, enum.UnitKindPhrase, enum.UnitKindGrammar:
		return kind, nil
	default:
		return "", fmt.Errorf("unknown unit kind: %s", value)
	}
}

func parseUnitStatus(value string) (enum.UnitStatus, error) {
	status := enum.UnitStatus(value)
	switch status {
	case enum.UnitStatusNew, enum.UnitStatusLearning, enum.UnitStatusReviewing, enum.UnitStatusMastered, enum.UnitStatusSuspended:
		return status, nil
	default:
		return "", fmt.Errorf("unknown unit status: %s", value)
	}
}

func reviewCandidateFromRow(row sqlcgen.FindDueReviewCandidatesRow) (query.ReviewCandidate, error) {
	status, err := parseUnitStatus(row.Status)
	if err != nil {
		return query.ReviewCandidate{}, err
	}
	kind, err := parseUnitKind(row.UnitKind)
	if err != nil {
		return query.ReviewCandidate{}, err
	}
	targetPriority, err := requiredFloat(row.TargetPriority, "user_unit_states.target_priority")
	if err != nil {
		return query.ReviewCandidate{}, err
	}
	progressPercent, err := requiredFloat(row.ProgressPercent, "user_unit_states.progress_percent")
	if err != nil {
		return query.ReviewCandidate{}, err
	}
	masteryScore, err := requiredFloat(row.MasteryScore, "user_unit_states.mastery_score")
	if err != nil {
		return query.ReviewCandidate{}, err
	}
	intervalDays, err := requiredFloat(row.IntervalDays, "user_unit_states.interval_days")
	if err != nil {
		return query.ReviewCandidate{}, err
	}
	easeFactor, err := requiredFloat(row.EaseFactor, "user_unit_states.ease_factor")
	if err != nil {
		return query.ReviewCandidate{}, err
	}
	userID, err := requiredUUID(row.UserID, "user_unit_states.user_id")
	if err != nil {
		return query.ReviewCandidate{}, err
	}
	stateCreatedAt, err := requiredTime(row.StateCreatedAt, "user_unit_states.created_at")
	if err != nil {
		return query.ReviewCandidate{}, err
	}
	stateUpdatedAt, err := requiredTime(row.StateUpdatedAt, "user_unit_states.updated_at")
	if err != nil {
		return query.ReviewCandidate{}, err
	}

	return query.ReviewCandidate{
		State: model.UserUnitState{
			UserID:                  userID,
			CoarseUnitID:            row.CoarseUnitID,
			IsTarget:                row.IsTarget,
			TargetSource:            textFromPG(row.TargetSource),
			TargetSourceRefID:       textFromPG(row.TargetSourceRefID),
			TargetPriority:          targetPriority,
			Status:                  status,
			ProgressPercent:         progressPercent,
			MasteryScore:            masteryScore,
			FirstSeenAt:             optionalTime(row.FirstSeenAt),
			LastSeenAt:              optionalTime(row.LastSeenAt),
			LastReviewedAt:          optionalTime(row.LastReviewedAt),
			SeenCount:               int(row.SeenCount),
			StrongEventCount:        int(row.StrongEventCount),
			ReviewCount:             int(row.ReviewCount),
			CorrectCount:            int(row.CorrectCount),
			WrongCount:              int(row.WrongCount),
			ConsecutiveCorrect:      int(row.ConsecutiveCorrect),
			ConsecutiveWrong:        int(row.ConsecutiveWrong),
			LastQuality:             optionalInt(row.LastQuality),
			RecentQualityWindow:     intsFromPG(row.RecentQualityWindow),
			RecentCorrectnessWindow: boolsFromPG(row.RecentCorrectnessWindow),
			Repetition:              int(row.Repetition),
			IntervalDays:            intervalDays,
			EaseFactor:              easeFactor,
			NextReviewAt:            optionalTime(row.NextReviewAt),
			SuspendedReason:         textFromPG(row.SuspendedReason),
			CreatedAt:               stateCreatedAt,
			UpdatedAt:               stateUpdatedAt,
		},
		Serving: model.UserUnitServingState{
			UserID:                  userID,
			CoarseUnitID:            row.CoarseUnitID,
			LastRecommendedAt:       optionalTime(row.LastRecommendedAt),
			LastRecommendationRunID: optionalUUID(row.LastRecommendationRunID),
			CreatedAt:               zeroTimeIfInvalid(row.ServingCreatedAt),
			UpdatedAt:               zeroTimeIfInvalid(row.ServingUpdatedAt),
		},
		Unit: model.LearningUnitRef{
			CoarseUnitID: row.CoarseUnitID,
			Kind:         kind,
			Label:        row.UnitLabel,
			Pos:          textFromPG(row.UnitPos),
			EnglishDef:   textFromPG(row.UnitEnglishDef),
			ChineseDef:   textFromPG(row.UnitChineseDef),
		},
	}, nil
}

func newCandidateFromRow(row sqlcgen.FindNewCandidatesRow) (query.NewCandidate, error) {
	status, err := parseUnitStatus(row.Status)
	if err != nil {
		return query.NewCandidate{}, err
	}
	kind, err := parseUnitKind(row.UnitKind)
	if err != nil {
		return query.NewCandidate{}, err
	}
	targetPriority, err := requiredFloat(row.TargetPriority, "user_unit_states.target_priority")
	if err != nil {
		return query.NewCandidate{}, err
	}
	progressPercent, err := requiredFloat(row.ProgressPercent, "user_unit_states.progress_percent")
	if err != nil {
		return query.NewCandidate{}, err
	}
	masteryScore, err := requiredFloat(row.MasteryScore, "user_unit_states.mastery_score")
	if err != nil {
		return query.NewCandidate{}, err
	}
	intervalDays, err := requiredFloat(row.IntervalDays, "user_unit_states.interval_days")
	if err != nil {
		return query.NewCandidate{}, err
	}
	easeFactor, err := requiredFloat(row.EaseFactor, "user_unit_states.ease_factor")
	if err != nil {
		return query.NewCandidate{}, err
	}
	userID, err := requiredUUID(row.UserID, "user_unit_states.user_id")
	if err != nil {
		return query.NewCandidate{}, err
	}
	stateCreatedAt, err := requiredTime(row.StateCreatedAt, "user_unit_states.created_at")
	if err != nil {
		return query.NewCandidate{}, err
	}
	stateUpdatedAt, err := requiredTime(row.StateUpdatedAt, "user_unit_states.updated_at")
	if err != nil {
		return query.NewCandidate{}, err
	}

	return query.NewCandidate{
		State: model.UserUnitState{
			UserID:                  userID,
			CoarseUnitID:            row.CoarseUnitID,
			IsTarget:                row.IsTarget,
			TargetSource:            textFromPG(row.TargetSource),
			TargetSourceRefID:       textFromPG(row.TargetSourceRefID),
			TargetPriority:          targetPriority,
			Status:                  status,
			ProgressPercent:         progressPercent,
			MasteryScore:            masteryScore,
			FirstSeenAt:             optionalTime(row.FirstSeenAt),
			LastSeenAt:              optionalTime(row.LastSeenAt),
			LastReviewedAt:          optionalTime(row.LastReviewedAt),
			SeenCount:               int(row.SeenCount),
			StrongEventCount:        int(row.StrongEventCount),
			ReviewCount:             int(row.ReviewCount),
			CorrectCount:            int(row.CorrectCount),
			WrongCount:              int(row.WrongCount),
			ConsecutiveCorrect:      int(row.ConsecutiveCorrect),
			ConsecutiveWrong:        int(row.ConsecutiveWrong),
			LastQuality:             optionalInt(row.LastQuality),
			RecentQualityWindow:     intsFromPG(row.RecentQualityWindow),
			RecentCorrectnessWindow: boolsFromPG(row.RecentCorrectnessWindow),
			Repetition:              int(row.Repetition),
			IntervalDays:            intervalDays,
			EaseFactor:              easeFactor,
			NextReviewAt:            optionalTime(row.NextReviewAt),
			SuspendedReason:         textFromPG(row.SuspendedReason),
			CreatedAt:               stateCreatedAt,
			UpdatedAt:               stateUpdatedAt,
		},
		Serving: model.UserUnitServingState{
			UserID:                  userID,
			CoarseUnitID:            row.CoarseUnitID,
			LastRecommendedAt:       optionalTime(row.LastRecommendedAt),
			LastRecommendationRunID: optionalUUID(row.LastRecommendationRunID),
			CreatedAt:               zeroTimeIfInvalid(row.ServingCreatedAt),
			UpdatedAt:               zeroTimeIfInvalid(row.ServingUpdatedAt),
		},
		Unit: model.LearningUnitRef{
			CoarseUnitID: row.CoarseUnitID,
			Kind:         kind,
			Label:        row.UnitLabel,
			Pos:          textFromPG(row.UnitPos),
			EnglishDef:   textFromPG(row.UnitEnglishDef),
			ChineseDef:   textFromPG(row.UnitChineseDef),
		},
	}, nil
}

func ReviewCandidatesFromRows(rows []sqlcgen.FindDueReviewCandidatesRow) ([]query.ReviewCandidate, error) {
	items := make([]query.ReviewCandidate, 0, len(rows))
	for _, row := range rows {
		item, err := reviewCandidateFromRow(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func NewCandidatesFromRows(rows []sqlcgen.FindNewCandidatesRow) ([]query.NewCandidate, error) {
	items := make([]query.NewCandidate, 0, len(rows))
	for _, row := range rows {
		item, err := newCandidateFromRow(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func zeroTimeIfInvalid(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}

	return value.Time
}
