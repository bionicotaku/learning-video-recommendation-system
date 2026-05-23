package mapper

import (
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
	learningenginesqlc "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/sqlcgen"
)

func ToLearningEvent(row learningenginesqlc.LearningUnitLearningEvent) model.LearningEvent {
	return model.LearningEvent{
		EventID:                   UUIDToString(row.EventID),
		LedgerSeq:                 Int64FromPG(row.LedgerSeq),
		UserID:                    UUIDToString(row.UserID),
		CoarseUnitID:              row.CoarseUnitID,
		VideoID:                   UUIDToString(row.VideoID),
		EventType:                 row.EventType,
		ReducerEffect:             row.ReducerEffect,
		SourceType:                row.SourceType,
		SourceRefID:               row.SourceRefID,
		IsCorrect:                 BoolPointerFromPG(row.IsCorrect),
		ProgressQuality:           Int16PointerFromPG(row.ProgressQuality),
		CountsTowardSuccessStreak: row.CountsTowardSuccessStreak,
		ConsumedWatchSessionIDs:   UUIDsToStrings(row.ConsumedWatchSessionIds),
		Metadata:                  row.Metadata,
		OccurredAt:                TimeFromPG(row.OccurredAt),
		ResetBoundaryAt:           TimePointerFromPG(row.ResetBoundaryAt),
		CreatedAt:                 TimeFromPG(row.CreatedAt),
	}
}

func ToLearningEventFromAppendRow(row learningenginesqlc.AppendLearningEventsRow) model.LearningEvent {
	return model.LearningEvent{
		EventID:                   UUIDToString(row.EventID),
		LedgerSeq:                 Int64FromPG(row.LedgerSeq),
		UserID:                    UUIDToString(row.UserID),
		CoarseUnitID:              row.CoarseUnitID,
		VideoID:                   UUIDToString(row.VideoID),
		EventType:                 row.EventType,
		ReducerEffect:             row.ReducerEffect,
		SourceType:                row.SourceType,
		SourceRefID:               row.SourceRefID,
		IsCorrect:                 BoolPointerFromPG(row.IsCorrect),
		ProgressQuality:           Int16PointerFromPG(row.ProgressQuality),
		CountsTowardSuccessStreak: row.CountsTowardSuccessStreak,
		ConsumedWatchSessionIDs:   UUIDsToStrings(row.ConsumedWatchSessionIds),
		Metadata:                  row.Metadata,
		OccurredAt:                TimeFromPG(row.OccurredAt),
		ResetBoundaryAt:           TimePointerFromPG(row.ResetBoundaryAt),
		CreatedAt:                 TimeFromPG(row.CreatedAt),
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
		UserID:                        UUIDToString(row.UserID),
		CoarseUnitID:                  row.CoarseUnitID,
		IsTarget:                      row.IsTarget,
		TargetSource:                  TextToString(row.TargetSource),
		TargetSourceRefID:             TextToString(row.TargetSourceRefID),
		TargetPriority:                targetPriority,
		Status:                        row.Status,
		ProgressPercent:               progressPercent,
		MasteryScore:                  masteryScore,
		FirstObservedAt:               TimePointerFromPG(row.FirstObservedAt),
		LastObservedAt:                TimePointerFromPG(row.LastObservedAt),
		ObservationCount:              row.ObservationCount,
		ProgressEventCount:            row.ProgressEventCount,
		LastProgressAt:                TimePointerFromPG(row.LastProgressAt),
		LastProgressQuality:           Int16PointerFromPG(row.LastProgressQuality),
		RecentProgressQualities:       row.RecentProgressQualities,
		RecentProgressPasses:          row.RecentProgressPasses,
		ProgressSuccessCount:          row.ProgressSuccessCount,
		ProgressFailureCount:          row.ProgressFailureCount,
		ConsecutiveSuccessCount:       row.ConsecutiveSuccessCount,
		ConsecutiveFailureCount:       row.ConsecutiveFailureCount,
		ScheduleRepetition:            row.ScheduleRepetition,
		ScheduleIntervalDays:          scheduleIntervalDays,
		ScheduleEaseFactor:            scheduleEaseFactor,
		NextReviewAt:                  TimePointerFromPG(row.NextReviewAt),
		LatestLearningEventOccurredAt: TimePointerFromPG(row.LatestLearningEventOccurredAt),
		LatestResetBoundaryAt:         TimePointerFromPG(row.LatestResetBoundaryAt),
		LatestLearningEventLedgerSeq:  row.LatestLearningEventLedgerSeq,
		CreatedAt:                     TimeFromPG(row.CreatedAt),
		UpdatedAt:                     TimeFromPG(row.UpdatedAt),
	}, nil
}
