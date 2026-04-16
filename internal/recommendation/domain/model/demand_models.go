package model

type DemandUnit struct {
	UnitID       int64
	Bucket       string
	Weight       float64
	SupplyGrade  string
	TargetWeight float64
}

type LaneBudget struct {
	ExactCore       float64
	Bundle          float64
	SoftFuture      float64
	QualityFallback float64
}

type MixQuota struct {
	CoreDominantMin   int
	FutureDominantMax int
	FutureLikeMax     int
	FallbackMax       int
	SameUnitMax       int
}

type PlannerFlags struct {
	HardReviewLowSupply bool
	ExtremeSparse       bool
}

type DemandBundle struct {
	HardReview           []DemandUnit
	NewNow               []DemandUnit
	SoftReview           []DemandUnit
	NearFuture           []DemandUnit
	SessionMode          string
	TargetVideoCount     int
	PreferredDurationSec [2]int
	LaneBudget           LaneBudget
	MixQuota             MixQuota
	Flags                PlannerFlags
}
