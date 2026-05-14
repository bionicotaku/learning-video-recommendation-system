package mapper

import (
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	learningenginesqlc "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/sqlcgen"
)

func ToLearningEvent(row learningenginesqlc.LearningUnitLearningEvent) model.LearningEvent {
	return model.LearningEvent{
		EventID:         UUIDToString(row.EventID),
		UserID:          UUIDToString(row.UserID),
		CoarseUnitID:    row.CoarseUnitID,
		VideoID:         UUIDToString(row.VideoID),
		EventType:       row.EventType,
		ReducerEffect:   row.ReducerEffect,
		SourceType:      row.SourceType,
		SourceRefID:     row.SourceRefID,
		IsCorrect:       BoolPointerFromPG(row.IsCorrect),
		ProgressQuality: Int16PointerFromPG(row.ProgressQuality),
		Metadata:        row.Metadata,
		OccurredAt:      row.OccurredAt.Time,
		CreatedAt:       row.CreatedAt.Time,
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
	scheduleIntervalDays, err := NumericToFloat64(row.ScheduleIntervalDays)
	if err != nil {
		return model.UserUnitState{}, err
	}
	scheduleEaseFactor, err := NumericToFloat64(row.ScheduleEaseFactor)
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
		FirstObservedAt:         TimePointerFromPG(row.FirstObservedAt),
		LastObservedAt:          TimePointerFromPG(row.LastObservedAt),
		ObservationCount:        row.ObservationCount,
		ProgressEventCount:      row.ProgressEventCount,
		LastProgressAt:          TimePointerFromPG(row.LastProgressAt),
		LastProgressQuality:     Int16PointerFromPG(row.LastProgressQuality),
		RecentProgressQualities: row.RecentProgressQualities,
		RecentProgressPasses:    row.RecentProgressPasses,
		ProgressSuccessCount:    row.ProgressSuccessCount,
		ProgressFailureCount:    row.ProgressFailureCount,
		ConsecutiveSuccessCount: row.ConsecutiveSuccessCount,
		ConsecutiveFailureCount: row.ConsecutiveFailureCount,
		ScheduleRepetition:      row.ScheduleRepetition,
		ScheduleIntervalDays:    scheduleIntervalDays,
		ScheduleEaseFactor:      scheduleEaseFactor,
		NextReviewAt:            TimePointerFromPG(row.NextReviewAt),
		SuspendedReason:         TextToString(row.SuspendedReason),
		CreatedAt:               row.CreatedAt.Time,
		UpdatedAt:               row.UpdatedAt.Time,
	}, nil
}
