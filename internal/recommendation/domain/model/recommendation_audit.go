package model

type RecommendationRun struct {
	RunID              string
	UserID             string
	RequestContext     []byte
	SessionMode        string
	SelectorMode       string
	PlannerSnapshot    []byte
	LaneBudgetSnapshot []byte
	CandidateSummary   []byte
	Underfilled        bool
	ResultCount        int32
}

type RecommendationItem struct {
	RunID                     string
	Rank                      int32
	VideoID                   string
	Score                     float64
	PrimaryLane               string
	DominantBucket            string
	DominantUnitID            *int64
	ReasonCodes               []string
	CoveredHardReviewCount    int32
	CoveredNewNowCount        int32
	CoveredSoftReviewCount    int32
	CoveredNearFutureCount    int32
	BestEvidenceSentenceIndex *int32
	BestEvidenceSpanIndex     *int32
	BestEvidenceStartMs       *int32
	BestEvidenceEndMs         *int32
}
