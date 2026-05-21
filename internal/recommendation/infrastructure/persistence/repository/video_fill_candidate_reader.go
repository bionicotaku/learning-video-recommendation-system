package repository

import (
	"context"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/mapper"
	recommendationsqlc "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

type VideoFillCandidateReader struct {
	queries *recommendationsqlc.Queries
}

var _ apprepo.VideoFillCandidateReader = (*VideoFillCandidateReader)(nil)

func NewVideoFillCandidateReader(db recommendationsqlc.DBTX) *VideoFillCandidateReader {
	return &VideoFillCandidateReader{
		queries: recommendationsqlc.New(db),
	}
}

func (r *VideoFillCandidateReader) ListMasteredTargetFillCandidates(ctx context.Context, userID string, excludedVideoIDs []string, limit int32) ([]model.VideoFillCandidate, error) {
	userUUID, err := mapper.StringToUUID(userID)
	if err != nil {
		return nil, err
	}
	excluded, err := videoFillUUIDs(excludedVideoIDs)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.ListMasteredTargetFillVideoCandidates(ctx, recommendationsqlc.ListMasteredTargetFillVideoCandidatesParams{
		UserID:           userUUID,
		ExcludedVideoIds: excluded,
		FillLimit:        limit,
	})
	if err != nil {
		return nil, err
	}

	result := make([]model.VideoFillCandidate, 0, len(rows))
	for _, row := range rows {
		mapped, err := mapper.ToVideoFillCandidateFromMasteredTarget(row)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}

func (r *VideoFillCandidateReader) ListPopularFillCandidates(ctx context.Context, userID string, excludedVideoIDs []string, limit int32) ([]model.VideoFillCandidate, error) {
	userUUID, err := mapper.StringToUUID(userID)
	if err != nil {
		return nil, err
	}
	excluded, err := videoFillUUIDs(excludedVideoIDs)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.ListPopularFillVideoCandidates(ctx, recommendationsqlc.ListPopularFillVideoCandidatesParams{
		UserID:           userUUID,
		ExcludedVideoIds: excluded,
		FillLimit:        limit,
	})
	if err != nil {
		return nil, err
	}

	result := make([]model.VideoFillCandidate, 0, len(rows))
	for _, row := range rows {
		mapped, err := mapper.ToVideoFillCandidateFromPopular(row)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}

func videoFillUUIDs(values []string) ([]pgtype.UUID, error) {
	result := make([]pgtype.UUID, 0, len(values))
	for _, value := range values {
		parsed, err := mapper.StringToUUID(value)
		if err != nil {
			return nil, err
		}
		result = append(result, parsed)
	}
	return result, nil
}
