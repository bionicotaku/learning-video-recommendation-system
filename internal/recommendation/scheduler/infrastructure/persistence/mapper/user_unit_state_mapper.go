package mapper

import (
	"fmt"

	"learning-video-recommendation-system/internal/recommendation/scheduler/application/query"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"
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

// UserUnitStateFromRow maps a sqlc row to a domain user-unit state.
func UserUnitStateFromRow(row sqlcgen.LearningUserUnitState) (model.UserUnitState, error) {
	userID, err := requiredUUID(row.UserID, "user_unit_states.user_id")
	if err != nil {
		return model.UserUnitState{}, err
	}

	status, err := parseUnitStatus(row.Status)
	if err != nil {
		return model.UserUnitState{}, err
	}

	targetPriority, err := requiredFloat(row.TargetPriority, "user_unit_states.target_priority")
	if err != nil {
		return model.UserUnitState{}, err
	}
	progressPercent, err := requiredFloat(row.ProgressPercent, "user_unit_states.progress_percent")
	if err != nil {
		return model.UserUnitState{}, err
	}
	masteryScore, err := requiredFloat(row.MasteryScore, "user_unit_states.mastery_score")
	if err != nil {
		return model.UserUnitState{}, err
	}
	intervalDays, err := requiredFloat(row.IntervalDays, "user_unit_states.interval_days")
	if err != nil {
		return model.UserUnitState{}, err
	}
	easeFactor, err := requiredFloat(row.EaseFactor, "user_unit_states.ease_factor")
	if err != nil {
		return model.UserUnitState{}, err
	}
	createdAt, err := requiredTime(row.CreatedAt, "user_unit_states.created_at")
	if err != nil {
		return model.UserUnitState{}, err
	}
	updatedAt, err := requiredTime(row.UpdatedAt, "user_unit_states.updated_at")
	if err != nil {
		return model.UserUnitState{}, err
	}

	return model.UserUnitState{
		UserID:             userID,
		CoarseUnitID:       row.CoarseUnitID,
		IsTarget:           row.IsTarget,
		TargetSource:       textFromPG(row.TargetSource),
		TargetSourceRefID:  textFromPG(row.TargetSourceRefID),
		TargetPriority:     targetPriority,
		Status:             status,
		ProgressPercent:    progressPercent,
		MasteryScore:       masteryScore,
		FirstSeenAt:        optionalTime(row.FirstSeenAt),
		LastSeenAt:         optionalTime(row.LastSeenAt),
		LastReviewedAt:     optionalTime(row.LastReviewedAt),
		LastRecommendedAt:  optionalTime(row.LastRecommendedAt),
		SeenCount:          int(row.SeenCount),
		StrongEventCount:   int(row.StrongEventCount),
		ReviewCount:        int(row.ReviewCount),
		CorrectCount:       int(row.CorrectCount),
		WrongCount:         int(row.WrongCount),
		ConsecutiveCorrect: int(row.ConsecutiveCorrect),
		ConsecutiveWrong:   int(row.ConsecutiveWrong),
		LastQuality:        optionalInt(row.LastQuality),
		Repetition:         int(row.Repetition),
		IntervalDays:       intervalDays,
		EaseFactor:         easeFactor,
		NextReviewAt:       optionalTime(row.NextReviewAt),
		SuspendedReason:    textFromPG(row.SuspendedReason),
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
	}, nil
}

// ReviewCandidateFromRow maps a sqlc review-candidate row to a domain query object.
func ReviewCandidateFromRow(row sqlcgen.FindDueReviewCandidatesRow) (query.ReviewCandidate, error) {
	state, err := UserUnitStateFromRow(sqlcgen.LearningUserUnitState{
		UserID:             row.UserID,
		CoarseUnitID:       row.CoarseUnitID,
		IsTarget:           row.IsTarget,
		TargetSource:       row.TargetSource,
		TargetSourceRefID:  row.TargetSourceRefID,
		TargetPriority:     row.TargetPriority,
		Status:             row.Status,
		ProgressPercent:    row.ProgressPercent,
		MasteryScore:       row.MasteryScore,
		FirstSeenAt:        row.FirstSeenAt,
		LastSeenAt:         row.LastSeenAt,
		LastReviewedAt:     row.LastReviewedAt,
		LastRecommendedAt:  row.LastRecommendedAt,
		SeenCount:          row.SeenCount,
		StrongEventCount:   row.StrongEventCount,
		ReviewCount:        row.ReviewCount,
		CorrectCount:       row.CorrectCount,
		WrongCount:         row.WrongCount,
		ConsecutiveCorrect: row.ConsecutiveCorrect,
		ConsecutiveWrong:   row.ConsecutiveWrong,
		LastQuality:        row.LastQuality,
		Repetition:         row.Repetition,
		IntervalDays:       row.IntervalDays,
		EaseFactor:         row.EaseFactor,
		NextReviewAt:       row.NextReviewAt,
		SuspendedReason:    row.SuspendedReason,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	})
	if err != nil {
		return query.ReviewCandidate{}, err
	}

	unit, err := parseUnitKind(row.UnitKind)
	if err != nil {
		return query.ReviewCandidate{}, err
	}

	return query.ReviewCandidate{
		State: state,
		Unit: model.LearningUnitRef{
			CoarseUnitID: row.CoarseUnitID,
			Kind:         unit,
			Label:        row.UnitLabel,
			Pos:          textFromPG(row.UnitPos),
			EnglishDef:   textFromPG(row.UnitEnglishDef),
			ChineseDef:   textFromPG(row.UnitChineseDef),
		},
	}, nil
}

// NewCandidateFromRow maps a sqlc new-candidate row to a domain query object.
func NewCandidateFromRow(row sqlcgen.FindNewCandidatesRow) (query.NewCandidate, error) {
	state, err := UserUnitStateFromRow(sqlcgen.LearningUserUnitState{
		UserID:             row.UserID,
		CoarseUnitID:       row.CoarseUnitID,
		IsTarget:           row.IsTarget,
		TargetSource:       row.TargetSource,
		TargetSourceRefID:  row.TargetSourceRefID,
		TargetPriority:     row.TargetPriority,
		Status:             row.Status,
		ProgressPercent:    row.ProgressPercent,
		MasteryScore:       row.MasteryScore,
		FirstSeenAt:        row.FirstSeenAt,
		LastSeenAt:         row.LastSeenAt,
		LastReviewedAt:     row.LastReviewedAt,
		LastRecommendedAt:  row.LastRecommendedAt,
		SeenCount:          row.SeenCount,
		StrongEventCount:   row.StrongEventCount,
		ReviewCount:        row.ReviewCount,
		CorrectCount:       row.CorrectCount,
		WrongCount:         row.WrongCount,
		ConsecutiveCorrect: row.ConsecutiveCorrect,
		ConsecutiveWrong:   row.ConsecutiveWrong,
		LastQuality:        row.LastQuality,
		Repetition:         row.Repetition,
		IntervalDays:       row.IntervalDays,
		EaseFactor:         row.EaseFactor,
		NextReviewAt:       row.NextReviewAt,
		SuspendedReason:    row.SuspendedReason,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	})
	if err != nil {
		return query.NewCandidate{}, err
	}

	unit, err := parseUnitKind(row.UnitKind)
	if err != nil {
		return query.NewCandidate{}, err
	}

	return query.NewCandidate{
		State: state,
		Unit: model.LearningUnitRef{
			CoarseUnitID: row.CoarseUnitID,
			Kind:         unit,
			Label:        row.UnitLabel,
			Pos:          textFromPG(row.UnitPos),
			EnglishDef:   textFromPG(row.UnitEnglishDef),
			ChineseDef:   textFromPG(row.UnitChineseDef),
		},
	}, nil
}

// UserUnitStateToUpsertParams maps a domain state to sqlc upsert params.
func UserUnitStateToUpsertParams(state *model.UserUnitState) (sqlcgen.UpsertUserUnitStateParams, error) {
	targetPriority, err := floatToPG(state.TargetPriority)
	if err != nil {
		return sqlcgen.UpsertUserUnitStateParams{}, err
	}
	progressPercent, err := floatToPG(state.ProgressPercent)
	if err != nil {
		return sqlcgen.UpsertUserUnitStateParams{}, err
	}
	masteryScore, err := floatToPG(state.MasteryScore)
	if err != nil {
		return sqlcgen.UpsertUserUnitStateParams{}, err
	}
	intervalDays, err := floatToPG(state.IntervalDays)
	if err != nil {
		return sqlcgen.UpsertUserUnitStateParams{}, err
	}
	easeFactor, err := floatToPG(state.EaseFactor)
	if err != nil {
		return sqlcgen.UpsertUserUnitStateParams{}, err
	}

	return sqlcgen.UpsertUserUnitStateParams{
		UserID:             UUIDToPG(state.UserID),
		CoarseUnitID:       state.CoarseUnitID,
		IsTarget:           state.IsTarget,
		TargetSource:       textToPG(state.TargetSource),
		TargetSourceRefID:  textToPG(state.TargetSourceRefID),
		TargetPriority:     targetPriority,
		Status:             string(state.Status),
		ProgressPercent:    progressPercent,
		MasteryScore:       masteryScore,
		FirstSeenAt:        OptionalTimeToPG(state.FirstSeenAt),
		LastSeenAt:         OptionalTimeToPG(state.LastSeenAt),
		LastReviewedAt:     OptionalTimeToPG(state.LastReviewedAt),
		LastRecommendedAt:  OptionalTimeToPG(state.LastRecommendedAt),
		SeenCount:          int32(state.SeenCount),
		StrongEventCount:   int32(state.StrongEventCount),
		ReviewCount:        int32(state.ReviewCount),
		CorrectCount:       int32(state.CorrectCount),
		WrongCount:         int32(state.WrongCount),
		ConsecutiveCorrect: int32(state.ConsecutiveCorrect),
		ConsecutiveWrong:   int32(state.ConsecutiveWrong),
		LastQuality:        optionalIntToPG(state.LastQuality),
		Repetition:         int32(state.Repetition),
		IntervalDays:       intervalDays,
		EaseFactor:         easeFactor,
		NextReviewAt:       OptionalTimeToPG(state.NextReviewAt),
		SuspendedReason:    textToPG(state.SuspendedReason),
		CreatedAt:          TimeToPG(state.CreatedAt),
		UpdatedAt:          TimeToPG(state.UpdatedAt),
	}, nil
}
