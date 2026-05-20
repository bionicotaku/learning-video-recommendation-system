package model

type RecommendationRequest struct {
	UserID           string
	TargetVideoCount int
	RequestContext   []byte
}
