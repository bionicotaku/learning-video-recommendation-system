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
	FirstSeenAt             *time.Time
	LastSeenAt              *time.Time
	LastReviewedAt          *time.Time
	SeenCount               int32
	StrongEventCount        int32
	ReviewCount             int32
	CorrectCount            int32
	WrongCount              int32
	ConsecutiveCorrect      int32
	ConsecutiveWrong        int32
	LastQuality             *int16
	RecentQualityWindow     []int16
	RecentCorrectnessWindow []bool
	Repetition              int32
	IntervalDays            float64
	EaseFactor              float64
	NextReviewAt            *time.Time
	SuspendedReason         string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}
