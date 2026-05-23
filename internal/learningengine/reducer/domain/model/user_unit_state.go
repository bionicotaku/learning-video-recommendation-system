package model

import "time"

type UserUnitState struct {
	UserID                  string
	CoarseUnitID            int64
	IsTarget                bool
	TargetSource            string
	TargetSourceRefID       string
	TargetPriority          float64
	Status                  string
	ProgressPercent         float64
	MasteryScore            float64
	FirstObservedAt         *time.Time
	LastObservedAt          *time.Time
	ObservationCount        int32
	ProgressEventCount      int32
	LastProgressAt          *time.Time
	LastProgressQuality     *int16
	RecentProgressQualities []int16
	RecentProgressPasses    []bool
	ProgressSuccessCount    int32
	ProgressFailureCount    int32
	ConsecutiveSuccessCount int32
	ConsecutiveFailureCount int32
	ScheduleRepetition      int32
	ScheduleIntervalDays    float64
	ScheduleEaseFactor      float64
	NextReviewAt            *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}
