package policy

type Bucket string

const (
	BucketHardReview Bucket = "hard_review"
	BucketNewNow     Bucket = "new_now"
	BucketSoftReview Bucket = "soft_review"
	BucketNearFuture Bucket = "near_future"
)

type Lane string

const (
	LaneExactCore       Lane = "exact_core"
	LaneBundle          Lane = "bundle"
	LaneSoftFuture      Lane = "soft_future"
	LaneQualityFallback Lane = "quality_fallback"
)

type SelectorMode string

const (
	SelectorModeNormal        SelectorMode = "normal"
	SelectorModeLowSupply     SelectorMode = "low_supply"
	SelectorModeExtremeSparse SelectorMode = "extreme_sparse"
)

type SessionMode string

const (
	SessionModeReviewHeavy SessionMode = "review_heavy"
	SessionModeBalanced    SessionMode = "balanced"
	SessionModeExplore     SessionMode = "explore"
)

type ReasonCode string

const (
	ReasonCodeHardReviewCovered  ReasonCode = "hard_review_covered"
	ReasonCodeNewUnitIntroduced  ReasonCode = "new_unit_introduced"
	ReasonCodeSoftReviewSupport  ReasonCode = "soft_review_supported"
	ReasonCodeNearFutureWarmup   ReasonCode = "near_future_warmup"
	ReasonCodeBundleCoverageHigh ReasonCode = "bundle_coverage_high"
	ReasonCodeStrongEvidence     ReasonCode = "strong_evidence_window"
	ReasonCodeGoodLearningFit    ReasonCode = "good_learning_fit"
	ReasonCodeRecentlyNotServed  ReasonCode = "recently_not_served"
	ReasonCodeLowSupplyPreserve  ReasonCode = "low_supply_core_preserved"
	ReasonCodeFallbackQuality    ReasonCode = "fallback_quality_fill"
)
