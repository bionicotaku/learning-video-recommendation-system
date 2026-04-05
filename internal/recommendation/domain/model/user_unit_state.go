package model

import (
	"time"

	"learning-video-recommendation-system/internal/recommendation/domain/enum"

	"github.com/google/uuid"
)

// UserUnitState is the Learning engine snapshot consumed by Recommendation.
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

	FirstSeenAt    *time.Time
	LastSeenAt     *time.Time
	LastReviewedAt *time.Time

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
