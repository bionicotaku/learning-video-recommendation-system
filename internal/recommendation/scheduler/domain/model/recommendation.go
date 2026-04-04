package model

import (
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"

	"github.com/google/uuid"
)

// RecommendationItem is a scheduler output item consumed by downstream stages.
type RecommendationItem struct {
	CoarseUnitID int64
	Kind         enum.UnitKind
	Label        string

	RecommendType enum.RecommendType
	Status        enum.UnitStatus
	Rank          int
	Score         float64
	ReasonCodes   []string

	TargetPriority  float64
	ProgressPercent float64
	MasteryScore    float64
	NextReviewAt    *time.Time
}

// RecommendationBatch is a full scheduler output batch.
type RecommendationBatch struct {
	RunID             uuid.UUID
	UserID            uuid.UUID
	GeneratedAt       time.Time
	SessionLimit      int
	ReviewQuota       int
	NewQuota          int
	BacklogProtection bool
	Items             []RecommendationItem
}
