package mapper

import (
	"learning-video-recommendation-system/internal/learningengine/normalizer/domain/model"
	normalizersqlc "learning-video-recommendation-system/internal/learningengine/normalizer/infrastructure/persistence/sqlcgen"
)

func ToRawQuizEvent(row normalizersqlc.ListPendingQuizEventsRow) model.RawQuizEvent {
	return model.RawQuizEvent{
		EventID:             UUIDToString(row.EventID),
		UserID:              UUIDToString(row.UserID),
		QuestionID:          UUIDToString(row.QuestionID),
		CoarseUnitID:        row.CoarseUnitID,
		VideoID:             UUIDToString(row.VideoID),
		RecommendationRunID: UUIDToString(row.RecommendationRunID),
		TriggerType:         row.TriggerType,
		SelectedOptionIDs:   row.SelectedOptionIds,
		SelectionIntervalMS: row.SelectionIntervalMs,
		IsFirstTryCorrect:   row.IsFirstTryCorrect,
		TotalElapsedMS:      row.TotalElapsedMs,
		ShownAt:             TimeFromPG(row.ShownAt),
		CompletedAt:         TimeFromPG(row.CompletedAt),
	}
}

func ToRawLearningInteraction(row normalizersqlc.ListPendingLearningInteractionsRow) model.RawLearningInteraction {
	return model.RawLearningInteraction{
		EventID:                        UUIDToString(row.EventID),
		UserID:                         UUIDToString(row.UserID),
		EventType:                      row.EventType,
		SourceSurface:                  row.SourceSurface,
		VideoID:                        UUIDToString(row.VideoID),
		WatchSessionID:                 UUIDToString(row.WatchSessionID),
		RecommendationRunID:            UUIDToString(row.RecommendationRunID),
		RelatedQuizEventID:             UUIDToString(row.RelatedQuizEventID),
		CoarseUnitID:                   Int64FromPG(row.CoarseUnitID),
		TokenText:                      TextToString(row.TokenText),
		SentenceIndex:                  Int32PointerFromPG(row.SentenceIndex),
		SpanIndex:                      Int32PointerFromPG(row.SpanIndex),
		OccurredAt:                     TimeFromPG(row.OccurredAt),
		ExposureStartMS:                Int32PointerFromPG(row.ExposureStartMs),
		ExposureEndMS:                  Int32PointerFromPG(row.ExposureEndMs),
		ExposureCount:                  Int32PointerFromPG(row.ExposureCount),
		LookupVisibleMS:                Int32PointerFromPG(row.LookupVisibleMs),
		LookupSentenceAudioReplayCount: row.LookupSentenceAudioReplayCount,
		LookupWordAudioPlayCount:       row.LookupWordAudioPlayCount,
		LookupPracticeNowClicked:       row.LookupPracticeNowClicked,
		EventPayload:                   row.EventPayload,
	}
}
