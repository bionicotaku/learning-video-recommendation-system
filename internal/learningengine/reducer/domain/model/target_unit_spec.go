package model

type TargetUnitSpec struct {
	CoarseUnitID      int64
	TargetSource      string
	TargetSourceRefID string
	TargetPriority    float64
}

type ActivatedUnitCollectionTarget struct {
	CollectionID   string
	CollectionSlug string
	TargetCount    int
}

type ActiveUnitCollection struct {
	CollectionID   string
	CollectionSlug string
}

type ActiveLearningTargetCoarseUnitIDs struct {
	ActiveCollection *string
	CoarseUnitIDs    []int64
}
