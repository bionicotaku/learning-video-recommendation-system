package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/mapper"
	recommendationsqlc "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"
)

type VideoUserStateReader struct {
	queries *recommendationsqlc.Queries
}

var _ apprepo.VideoUserStateReader = (*VideoUserStateReader)(nil)

func NewVideoUserStateReader(db recommendationsqlc.DBTX) *VideoUserStateReader {
	return &VideoUserStateReader{queries: recommendationsqlc.New(db)}
}

func (r *VideoUserStateReader) ListByUserAndVideoIDs(ctx context.Context, userID string, videoIDs []string) ([]model.VideoUserState, error) {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return nil, err
	}

	parsedIDs := make([]pgtype.UUID, 0, len(videoIDs))
	for _, videoID := range videoIDs {
		pgVideoID, err := mapper.StringToUUID(videoID)
		if err != nil {
			return nil, err
		}
		parsedIDs = append(parsedIDs, pgVideoID)
	}

	rows, err := r.queries.ListVideoUserStatesByUserAndVideoIDs(ctx, recommendationsqlc.ListVideoUserStatesByUserAndVideoIDsParams{
		UserID:   pgUserID,
		VideoIds: parsedIDs,
	})
	if err != nil {
		return nil, err
	}

	result := make([]model.VideoUserState, 0, len(rows))
	for _, row := range rows {
		mapped, err := mapper.ToVideoUserState(row)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}
