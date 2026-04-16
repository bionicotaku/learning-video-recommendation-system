package model

import "time"

type LearningStateSnapshot struct {
	UserID                  string
	CoarseUnitID            int64
	IsTarget                bool
	TargetPriority          float64
	Status                  string
	ProgressPercent         float64
	MasteryScore            float64
	LastQuality             *int16
	NextReviewAt            *time.Time
	RecentQualityWindow     []int16
	RecentCorrectnessWindow []bool
	StrongEventCount        int32
	ReviewCount             int32
	UpdatedAt               time.Time
}
