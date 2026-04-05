package model

import (
	"time"

	"github.com/google/uuid"
)

// UserUnitServingState is the Recommendation-owned serving snapshot for one user-unit pair.
type UserUnitServingState struct {
	UserID                  uuid.UUID
	CoarseUnitID            int64
	LastRecommendedAt       *time.Time
	LastRecommendationRunID *uuid.UUID
	CreatedAt               time.Time
	UpdatedAt               time.Time
}
