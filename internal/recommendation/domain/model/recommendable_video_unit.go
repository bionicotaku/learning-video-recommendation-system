package model

import "time"

type RecommendableVideoUnit struct {
	VideoID            string
	CoarseUnitID       int64
	MentionCount       int32
	SentenceCount      int32
	FirstStartMs       int32
	LastEndMs          int32
	CoverageMs         int32
	CoverageRatio      float64
	SentenceIndexes    []int32
	EvidenceSpanRefs   []byte
	SampleSurfaceForms []string
	DurationMs         int32
	MappedSpanRatio    float64
	Status             string
	VisibilityStatus   string
	PublishAt          *time.Time
}
