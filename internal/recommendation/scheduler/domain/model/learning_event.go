package model

import (
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"

	"github.com/google/uuid"
)

// LearningEvent is a normalized learning activity record.
type LearningEvent struct {
	EventID       int64
	UserID        uuid.UUID
	CoarseUnitID  int64
	VideoID       *uuid.UUID
	EventType     enum.EventType
	SourceType    string
	SourceRefID   string
	IsCorrect     *bool
	Quality       *int
	ResponseTimeMs *int
	Metadata      map[string]any
	OccurredAt    time.Time
	CreatedAt     time.Time
}
