package model

type EvidenceRef struct {
	SentenceIndex int32 `json:"sentence_index"`
	SpanIndex     int32 `json:"span_index"`
}

type VideoUnitCandidate struct {
	VideoID            string
	CoarseUnitID       int64
	Lane               string
	Bucket             string
	UnitWeight         float64
	MentionCount       int32
	SentenceCount      int32
	CoverageMs         int32
	CoverageRatio      float64
	SentenceIndexes    []int32
	EvidenceSpanRefs   []byte
	SampleSurfaceForms []string
	DurationMs         int32
	MappedSpanRatio    float64
	CandidateScore     float64
}

type ResolvedEvidenceWindow struct {
	Candidate             VideoUnitCandidate
	BestEvidenceRef       *EvidenceRef
	BestEvidenceStartMs   *int32
	BestEvidenceEndMs     *int32
	WindowSentenceIndexes []int32
	WindowStartMs         *int32
	WindowEndMs           *int32
	ResolvedSpans         []SemanticSpan
	ResolvedSentences     []TranscriptSentence
}

type VideoCandidate struct {
	VideoID                   string
	LaneSources               []string
	DominantBucket            string
	DominantUnitID            *int64
	CoveredHardReviewUnits    []int64
	CoveredNewNowUnits        []int64
	CoveredSoftReviewUnits    []int64
	CoveredNearFutureUnits    []int64
	HardReviewCover           float64
	NewNowCover               float64
	SoftReviewCover           float64
	NearFutureCover           float64
	CoverageStrengthScore     float64
	BundleValueScore          float64
	EducationalFitScore       float64
	FutureValueScore          float64
	FreshnessScore            float64
	RecentServedPenalty       float64
	RecentWatchedPenalty      float64
	OverloadPenalty           float64
	BaseScore                 float64
	BestEvidenceSentenceIndex *int32
	BestEvidenceSpanIndex     *int32
	BestEvidenceStartMs       *int32
	BestEvidenceEndMs         *int32
}
