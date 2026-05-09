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
	RunID          string
	Rank           int32
	VideoID        string
	Score          float64
	PrimaryLane    string
	DominantRole   LearningRole
	DominantUnitID *int64
	ReasonCodes    []string
	LearningUnits  []ExpectedLearningUnit
}
