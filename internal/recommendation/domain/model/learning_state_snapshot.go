package model

import "time"

type LearningStateSnapshot struct {
	UserID              string
	CoarseUnitID        int64
	IsTarget            bool
	TargetPriority      float64
	Status              string
	MasteryScore        float64
	LastProgressQuality *int16
	NextReviewAt        *time.Time
	UpdatedAt           time.Time
}
