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

type SetTargetInactiveRequest struct {
	UserID       string
	CoarseUnitID int64
}

type SetTargetInactiveResponse struct{}

type SuspendTargetUnitRequest struct {
	UserID          string
	CoarseUnitID    int64
	SuspendedReason string
}

type SuspendTargetUnitResponse struct{}

type ResumeTargetUnitRequest struct {
	UserID       string
	CoarseUnitID int64
}

type ResumeTargetUnitResponse struct{}
