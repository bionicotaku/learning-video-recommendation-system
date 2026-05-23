package dto

type TargetUnitSpec struct {
	CoarseUnitID      int64
	TargetSource      string
	TargetSourceRefID string
	TargetPriority    float64
}

type EnsureTargetUnitsRequest struct {
	UserID  string
	Targets []TargetUnitSpec
}

type EnsureTargetUnitsResponse struct {
	TargetCount int
}

type ActivateUnitCollectionTargetRequest struct {
	UserID         string
	CollectionSlug string
}

type ActivateUnitCollectionTargetResponse struct {
	CollectionID   string `json:"collection_id"`
	CollectionSlug string `json:"collection_slug"`
	TargetCount    int    `json:"target_count"`
}

type ActiveUnitCollection struct {
	CollectionID   string
	CollectionSlug string
}

type GetActiveUnitCollectionRequest struct {
	UserID string
}

type GetActiveUnitCollectionResponse struct {
	ActiveCollection *ActiveUnitCollection
}

type GetActiveLearningTargetCoarseUnitIDsRequest struct {
	UserID string
}

type GetActiveLearningTargetCoarseUnitIDsResponse struct {
	ActiveCollection *string `json:"active_collection"`
	TargetCount      int     `json:"target_count"`
	CoarseUnitIDs    []int64 `json:"coarse_unit_ids"`
}

type SetTargetInactiveRequest struct {
	UserID       string
	CoarseUnitID int64
}

type SetTargetInactiveResponse struct{}
