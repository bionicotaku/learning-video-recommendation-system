package model

import (
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"

	"github.com/google/uuid"
)

// UserUnitState is the current scheduling snapshot for a user-unit relation.
type UserUnitState struct {
	UserID       uuid.UUID
	CoarseUnitID int64

	IsTarget          bool
	TargetSource      string
	TargetSourceRefID string
	TargetPriority    float64

	Status enum.UnitStatus

	ProgressPercent float64
	MasteryScore    float64

	FirstSeenAt       *time.Time
	LastSeenAt        *time.Time
	LastReviewedAt    *time.Time
	LastRecommendedAt *time.Time

	SeenCount          int
	StrongEventCount   int
	ReviewCount        int
	CorrectCount       int
	WrongCount         int
	ConsecutiveCorrect int
	ConsecutiveWrong   int

	LastQuality *int

	RecentQualityWindow     []int
	RecentCorrectnessWindow []bool

	Repetition   int
	IntervalDays float64
	EaseFactor   float64
	NextReviewAt *time.Time

	SuspendedReason string

	CreatedAt time.Time
	UpdatedAt time.Time
}
