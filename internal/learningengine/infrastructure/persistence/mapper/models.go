package mapper

import (
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	learningenginesqlc "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/sqlcgen"
)

func ToLearningEvent(row learningenginesqlc.LearningUnitLearningEvent) model.LearningEvent {
	return model.LearningEvent{
		EventID:        row.EventID,
		UserID:         UUIDToString(row.UserID),
		CoarseUnitID:   row.CoarseUnitID,
		VideoID:        UUIDToString(row.VideoID),
		EventType:      row.EventType,
		SourceType:     row.SourceType,
		SourceRefID:    TextToString(row.SourceRefID),
		IsCorrect:      BoolPointerFromPG(row.IsCorrect),
		Quality:        Int16PointerFromPG(row.Quality),
		ResponseTimeMs: Int32PointerFromPG(row.ResponseTimeMs),
		Metadata:       row.Metadata,
		OccurredAt:     row.OccurredAt.Time,
		CreatedAt:      row.CreatedAt.Time,
	}
}

func ToUserUnitState(row learningenginesqlc.LearningUserUnitState) (model.UserUnitState, error) {
	targetPriority, err := NumericToFloat64(row.TargetPriority)
	if err != nil {
		return model.UserUnitState{}, err
	}
	progressPercent, err := NumericToFloat64(row.ProgressPercent)
	if err != nil {
		return model.UserUnitState{}, err
	}
	masteryScore, err := NumericToFloat64(row.MasteryScore)
	if err != nil {
		return model.UserUnitState{}, err
	}
	intervalDays, err := NumericToFloat64(row.IntervalDays)
	if err != nil {
		return model.UserUnitState{}, err
	}
	easeFactor, err := NumericToFloat64(row.EaseFactor)
	if err != nil {
		return model.UserUnitState{}, err
	}

	return model.UserUnitState{
		UserID:                  UUIDToString(row.UserID),
		CoarseUnitID:            row.CoarseUnitID,
		IsTarget:                row.IsTarget,
		TargetSource:            TextToString(row.TargetSource),
		TargetSourceRefID:       TextToString(row.TargetSourceRefID),
		TargetPriority:          targetPriority,
		Status:                  row.Status,
		ProgressPercent:         progressPercent,
		MasteryScore:            masteryScore,
		FirstSeenAt:             TimePointerFromPG(row.FirstSeenAt),
		LastSeenAt:              TimePointerFromPG(row.LastSeenAt),
		LastReviewedAt:          TimePointerFromPG(row.LastReviewedAt),
		SeenCount:               row.SeenCount,
		StrongEventCount:        row.StrongEventCount,
		ReviewCount:             row.ReviewCount,
		CorrectCount:            row.CorrectCount,
		WrongCount:              row.WrongCount,
		ConsecutiveCorrect:      row.ConsecutiveCorrect,
		ConsecutiveWrong:        row.ConsecutiveWrong,
		LastQuality:             Int16PointerFromPG(row.LastQuality),
		RecentQualityWindow:     row.RecentQualityWindow,
		RecentCorrectnessWindow: row.RecentCorrectnessWindow,
		Repetition:              row.Repetition,
		IntervalDays:            intervalDays,
		EaseFactor:              easeFactor,
		NextReviewAt:            TimePointerFromPG(row.NextReviewAt),
		SuspendedReason:         TextToString(row.SuspendedReason),
		CreatedAt:               row.CreatedAt.Time,
		UpdatedAt:               row.UpdatedAt.Time,
	}, nil
}
