package mapper

import (
	"learning-video-recommendation-system/internal/learningengine/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/sqlcgen"
)

func parseEventType(value string) enum.EventType {
	return enum.EventType(value)
}

// LearningEventFromRow maps a sqlc learning-event row to a domain learning event.
func LearningEventFromRow(row sqlcgen.LearningUnitLearningEvent) (model.LearningEvent, error) {
	userID, err := requiredUUID(row.UserID, "unit_learning_events.user_id")
	if err != nil {
		return model.LearningEvent{}, err
	}
	occurredAt, err := requiredTime(row.OccurredAt, "unit_learning_events.occurred_at")
	if err != nil {
		return model.LearningEvent{}, err
	}
	createdAt, err := requiredTime(row.CreatedAt, "unit_learning_events.created_at")
	if err != nil {
		return model.LearningEvent{}, err
	}
	metadata, err := metadataFromBytes(row.Metadata)
	if err != nil {
		return model.LearningEvent{}, err
	}

	responseTimeMs := (*int)(nil)
	if row.ResponseTimeMs.Valid {
		v := int(row.ResponseTimeMs.Int32)
		responseTimeMs = &v
	}

	return model.LearningEvent{
		EventID:        row.EventID,
		UserID:         userID,
		CoarseUnitID:   row.CoarseUnitID,
		VideoID:        optionalUUID(row.VideoID),
		EventType:      parseEventType(row.EventType),
		SourceType:     row.SourceType,
		SourceRefID:    textFromPG(row.SourceRefID),
		IsCorrect:      optionalBool(row.IsCorrect),
		Quality:        optionalInt(row.Quality),
		ResponseTimeMs: responseTimeMs,
		Metadata:       metadata,
		OccurredAt:     occurredAt,
		CreatedAt:      createdAt,
	}, nil
}

// LearningEventToInsertParams maps a domain learning event to sqlc insert params.
func LearningEventToInsertParams(event model.LearningEvent) (sqlcgen.InsertUnitLearningEventParams, error) {
	metadata, err := metadataToBytes(event.Metadata)
	if err != nil {
		return sqlcgen.InsertUnitLearningEventParams{}, err
	}

	return sqlcgen.InsertUnitLearningEventParams{
		UserID:         UUIDToPG(event.UserID),
		CoarseUnitID:   event.CoarseUnitID,
		VideoID:        OptionalUUIDToPG(event.VideoID),
		EventType:      string(event.EventType),
		SourceType:     event.SourceType,
		SourceRefID:    textToPG(event.SourceRefID),
		IsCorrect:      optionalBoolToPG(event.IsCorrect),
		Quality:        optionalIntToPG(event.Quality),
		ResponseTimeMs: optionalInt32ToPG(event.ResponseTimeMs),
		Metadata:       metadata,
		OccurredAt:     TimeToPG(event.OccurredAt),
		CreatedAt:      TimeToPG(event.CreatedAt),
	}, nil
}
