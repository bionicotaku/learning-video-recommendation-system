package model

import "time"

type UnitVideoInventory struct {
	CoarseUnitID       int64
	DistinctVideoCount int32
	AvgMentionCount    float64
	AvgSentenceCount   float64
	AvgCoverageMs      float64
	AvgCoverageRatio   float64
	StrongVideoCount   int32
	SupplyGrade        string
	UpdatedAt          time.Time
}
