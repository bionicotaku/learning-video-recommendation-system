package model

import "time"

type RecallQueueState struct {
	UserID                     string
	SourceLearningMaxUpdatedAt *time.Time
	SourceProjectionUpdatedAt  time.Time
	ActiveTargetUnitCount      int32
	RebuiltAt                  time.Time
}

type RecallQueueCandidate struct {
	UserID              string
	CoarseUnitID        int64
	Status              string
	TargetPriority      float64
	MasteryScore        float64
	LastProgressQuality *int16
	NextReviewAt        *time.Time
	SupplyGrade         string
	StateUpdatedAt      time.Time
	LastServedAt        *time.Time
	ServedCount         int32
	Bucket              string
	DynamicPriority     float64
	BucketRank          int32
}

type RecallScopeSelection struct {
	PlannerScope     []RecallQueueCandidate
	RecallFetchScope []RecallQueueCandidate
	Summary          RecallScopeSummary
}

type RecallScopeSummary struct {
	ActiveTargetUnitCount         int
	QueueRebuilt                  bool
	QueueCandidateCount           int
	PlannerScopeUnitCount         int
	RecallFetchUnitCount          int
	PlannerScopeUnitCountByBucket map[string]int
	NoSupplyScopeUnitCount        int
	PerUnitRecallLimit            int32
	MaxPossibleRecallRows         int
	ActualRecallRowCount          int
	AggregatedVideoCandidateCount int
	VideoStateLookupCount         int
}
