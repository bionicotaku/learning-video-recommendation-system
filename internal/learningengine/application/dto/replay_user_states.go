package dto

type ReplayUserStatesRequest struct {
	UserID string
}

type ReplayUserStatesResponse struct {
	RebuiltUnitCount    int
	ProcessedEventCount int
}
