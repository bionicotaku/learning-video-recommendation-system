package model

type RecommendationRequest struct {
	UserID               string
	TargetVideoCount     int
	PreferredDurationSec [2]int
	SessionHint          string
	RequestContext       []byte
}
