package command

import (
	"time"

	"learning-video-recommendation-system/internal/learningengine/domain/enum"

	"github.com/google/uuid"
)

// LearningEventInput is the application-layer input used to record a learning event.
type LearningEventInput struct {
	CoarseUnitID   int64
	VideoID        *uuid.UUID
	EventType      enum.EventType
	SourceType     string
	SourceRefID    string
	IsCorrect      *bool
	Quality        *int
	ResponseTimeMs *int
	Metadata       map[string]any
	OccurredAt     time.Time
}

// RecordLearningEventsCommand records normalized learning events for one user.
type RecordLearningEventsCommand struct {
	UserID         uuid.UUID
	Events         []LearningEventInput
	IdempotencyKey string
}
