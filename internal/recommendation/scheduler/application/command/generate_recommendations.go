package command

import (
	"time"

	"github.com/google/uuid"
)

// GenerateRecommendationsCommand requests one scheduler recommendation batch.
type GenerateRecommendationsCommand struct {
	UserID         uuid.UUID
	RequestedLimit int
	Now            time.Time
	RequestContext map[string]any
}
