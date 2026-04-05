package mapper

import (
	"time"

	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"

	"github.com/google/uuid"
)

func UserUnitServingStateToUpsertParams(userID uuid.UUID, coarseUnitID int64, runID uuid.UUID, recommendedAt time.Time) sqlcgen.UpsertUserUnitServingStateParams {
	return sqlcgen.UpsertUserUnitServingStateParams{
		UserID:                  UUIDToPG(userID),
		CoarseUnitID:            coarseUnitID,
		LastRecommendedAt:       TimeToPG(recommendedAt),
		LastRecommendationRunID: UUIDToPG(runID),
		CreatedAt:               TimeToPG(recommendedAt),
		UpdatedAt:               TimeToPG(recommendedAt),
	}
}
