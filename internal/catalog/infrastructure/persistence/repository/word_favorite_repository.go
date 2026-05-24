package repository

import (
	"context"
	"errors"
	"fmt"

	"learning-video-recommendation-system/internal/catalog/application/dto"
	apprepo "learning-video-recommendation-system/internal/catalog/application/repository"
	"learning-video-recommendation-system/internal/catalog/domain/model"
	"learning-video-recommendation-system/internal/catalog/infrastructure/persistence/mapper"
	catalogsqlc "learning-video-recommendation-system/internal/catalog/infrastructure/persistence/sqlcgen"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type WordFavoriteRepository struct {
	queries *catalogsqlc.Queries
}

var _ apprepo.WordFavoriteRepository = (*WordFavoriteRepository)(nil)

func NewWordFavoriteRepository(db catalogsqlc.DBTX) *WordFavoriteRepository {
	return &WordFavoriteRepository{queries: catalogsqlc.New(db)}
}

func (r *WordFavoriteRepository) HasCoarseFavorite(ctx context.Context, key model.WordFavoriteCoarseKey) (bool, error) {
	userID, err := mapper.StringToUUID(key.UserID)
	if err != nil {
		return false, fmt.Errorf("map user_id: %w", err)
	}
	return r.queries.HasWordFavoriteByCoarse(ctx, catalogsqlc.HasWordFavoriteByCoarseParams{
		UserID:       userID,
		CoarseUnitID: key.CoarseUnitID,
	})
}

func (r *WordFavoriteRepository) HasTokenFavorite(ctx context.Context, key model.WordFavoriteTokenKey) (bool, error) {
	params, err := tokenKeyParams(key)
	if err != nil {
		return false, err
	}
	return r.queries.HasWordFavoriteByToken(ctx, catalogsqlc.HasWordFavoriteByTokenParams(params))
}

func (r *WordFavoriteRepository) GetVideoContext(ctx context.Context, key model.WordFavoriteVideoContextKey) (model.WordFavoriteVideoContext, error) {
	videoID, err := mapper.StringToUUID(key.VideoID)
	if err != nil {
		return model.WordFavoriteVideoContext{}, fmt.Errorf("map video_id: %w", err)
	}
	row, err := r.queries.GetWordFavoriteVideoContext(ctx, catalogsqlc.GetWordFavoriteVideoContextParams{
		VideoID:       videoID,
		SentenceIndex: key.SentenceIndex,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.WordFavoriteVideoContext{}, apprepo.ErrWordFavoriteProjectionNotFound
		}
		return model.WordFavoriteVideoContext{}, err
	}
	return model.WordFavoriteVideoContext{
		VideoTitle:          row.Title,
		VideoDurationMS:     row.DurationMs,
		SentenceText:        row.SentenceText,
		SentenceTranslation: textPointer(row.SentenceTranslation),
		SentenceStartMS:     row.SentenceStartMs,
		SentenceEndMS:       row.SentenceEndMs,
	}, nil
}

func (r *WordFavoriteRepository) SetCoarseFavorite(ctx context.Context, command model.SetCoarseWordFavoriteCommand) (apprepo.WordFavoriteWriteOutcome, error) {
	userID, err := mapper.StringToUUID(command.UserID)
	if err != nil {
		return "", fmt.Errorf("map user_id: %w", err)
	}
	videoID, err := optionalUUID(command.VideoID)
	if err != nil {
		return "", err
	}
	outcome, err := r.queries.SetCoarseWordFavorite(ctx, catalogsqlc.SetCoarseWordFavoriteParams{
		UserID:        userID,
		CoarseUnitID:  command.CoarseUnitID,
		Source:        command.Source,
		VideoID:       videoID,
		SentenceIndex: optionalInt4(command.SentenceIndex),
		TokenIndex:    optionalInt4(command.TokenIndex),
		OccurredAt:    mapper.TimePointerToPG(&command.OccurredAt),
	})
	return mapWordFavoriteWriteOutcome(outcome, err)
}

func (r *WordFavoriteRepository) SetTokenFavorite(ctx context.Context, command model.SetTokenWordFavoriteCommand) (apprepo.WordFavoriteWriteOutcome, error) {
	key := model.WordFavoriteTokenKey{
		UserID:        command.UserID,
		VideoID:       command.VideoID,
		SentenceIndex: command.SentenceIndex,
		TokenIndex:    command.TokenIndex,
	}
	params, err := tokenKeyParams(key)
	if err != nil {
		return "", err
	}
	outcome, err := r.queries.SetTokenWordFavorite(ctx, catalogsqlc.SetTokenWordFavoriteParams{
		UserID:        params.UserID,
		VideoID:       params.VideoID,
		SentenceIndex: params.SentenceIndex,
		TokenIndex:    params.TokenIndex,
		OccurredAt:    mapper.TimePointerToPG(&command.OccurredAt),
	})
	return mapWordFavoriteWriteOutcome(outcome, err)
}

func (r *WordFavoriteRepository) UnsetCoarseFavorite(ctx context.Context, command model.UnsetCoarseWordFavoriteCommand) error {
	userID, err := mapper.StringToUUID(command.UserID)
	if err != nil {
		return fmt.Errorf("map user_id: %w", err)
	}
	videoID, err := optionalUUID(command.VideoID)
	if err != nil {
		return err
	}
	return r.queries.UnsetWordFavoriteByCoarse(ctx, catalogsqlc.UnsetWordFavoriteByCoarseParams{
		UserID:        userID,
		CoarseUnitID:  command.CoarseUnitID,
		Source:        command.Source,
		VideoID:       videoID,
		SentenceIndex: optionalInt4(command.SentenceIndex),
		TokenIndex:    optionalInt4(command.TokenIndex),
		OccurredAt:    mapper.TimePointerToPG(&command.OccurredAt),
	})
}

func (r *WordFavoriteRepository) UnsetTokenFavorite(ctx context.Context, command model.UnsetTokenWordFavoriteCommand) error {
	params, err := tokenKeyParams(model.WordFavoriteTokenKey{
		UserID:        command.UserID,
		VideoID:       command.VideoID,
		SentenceIndex: command.SentenceIndex,
		TokenIndex:    command.TokenIndex,
	})
	if err != nil {
		return err
	}
	return r.queries.UnsetWordFavoriteByToken(ctx, catalogsqlc.UnsetWordFavoriteByTokenParams{
		UserID:        params.UserID,
		VideoID:       params.VideoID,
		SentenceIndex: params.SentenceIndex,
		TokenIndex:    params.TokenIndex,
		OccurredAt:    mapper.TimePointerToPG(&command.OccurredAt),
	})
}

func (r *WordFavoriteRepository) ListWordFavorites(ctx context.Context, query dto.ListWordFavoritesQuery) ([]model.WordFavoriteListItem, error) {
	userID, err := mapper.StringToUUID(query.UserID)
	if err != nil {
		return nil, fmt.Errorf("map user_id: %w", err)
	}
	cursorFavoriteID, err := cursorWordFavoriteID(query.Cursor)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.ListWordFavorites(ctx, catalogsqlc.ListWordFavoritesParams{
		UserID:            userID,
		HasCursor:         query.Cursor != nil,
		CursorFavoritedAt: cursorWordFavoriteFavoritedAt(query.Cursor),
		CursorFavoriteID:  cursorFavoriteID,
		LimitPlusOne:      int32(query.LimitPlusOne),
	})
	if err != nil {
		return nil, err
	}
	result := make([]model.WordFavoriteListItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, model.WordFavoriteListItem{
			FavoriteID:        mapper.UUIDToString(row.FavoriteID),
			FavoritedAt:       mapper.TimeFromPG(row.FavoritedAt),
			CoarseUnitID:      int8Pointer(row.CoarseUnitID),
			Label:             textPointer(row.Label),
			Pos:               textPointer(row.Pos),
			ChineseLabel:      textPointer(row.ChineseLabel),
			ChineseDef:        textPointer(row.ChineseDef),
			Source:            row.Source,
			VideoID:           uuidPointer(row.VideoID),
			SentenceIndex:     int4Pointer(row.SentenceIndex),
			TokenIndex:        int4Pointer(row.TokenIndex),
			SourceText:        textPointer(row.SourceText),
			SourceTranslation: textPointer(row.SourceTranslation),
			SourceDictionary:  textPointer(row.SourceDictionary),
			SourceExplanation: textPointer(row.SourceExplanation),
		})
	}
	return result, nil
}

type tokenParams struct {
	UserID        pgtype.UUID `json:"user_id"`
	VideoID       pgtype.UUID `json:"video_id"`
	SentenceIndex int32       `json:"sentence_index"`
	TokenIndex    int32       `json:"token_index"`
}

func tokenKeyParams(key model.WordFavoriteTokenKey) (tokenParams, error) {
	userID, err := mapper.StringToUUID(key.UserID)
	if err != nil {
		return tokenParams{}, fmt.Errorf("map user_id: %w", err)
	}
	videoID, err := mapper.StringToUUID(key.VideoID)
	if err != nil {
		return tokenParams{}, fmt.Errorf("map video_id: %w", err)
	}
	return tokenParams{
		UserID:        userID,
		VideoID:       videoID,
		SentenceIndex: key.SentenceIndex,
		TokenIndex:    key.TokenIndex,
	}, nil
}

func optionalUUID(value *string) (pgtype.UUID, error) {
	if value == nil {
		return pgtype.UUID{}, nil
	}
	uuid, err := mapper.StringToUUID(*value)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("map video_id: %w", err)
	}
	return uuid, nil
}

func mapWordFavoriteWriteOutcome(outcome string, err error) (apprepo.WordFavoriteWriteOutcome, error) {
	if err != nil {
		return "", err
	}
	switch apprepo.WordFavoriteWriteOutcome(outcome) {
	case apprepo.WordFavoriteWriteApplied:
		return apprepo.WordFavoriteWriteApplied, nil
	case apprepo.WordFavoriteWriteStale:
		return apprepo.WordFavoriteWriteStale, nil
	case apprepo.WordFavoriteWriteTargetNotFound:
		return apprepo.WordFavoriteWriteTargetNotFound, nil
	default:
		return "", fmt.Errorf("unknown word favorite write outcome: %s", outcome)
	}
}

func optionalInt4(value *int32) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *value, Valid: true}
}

func int4Pointer(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	result := value.Int32
	return &result
}

func int8Pointer(value pgtype.Int8) *int64 {
	if !value.Valid {
		return nil
	}
	result := value.Int64
	return &result
}

func uuidPointer(value pgtype.UUID) *string {
	if !value.Valid {
		return nil
	}
	result := mapper.UUIDToString(value)
	return &result
}

func cursorWordFavoriteFavoritedAt(cursor *dto.WordFavoritesCursor) pgtype.Timestamptz {
	if cursor == nil {
		return pgtype.Timestamptz{}
	}
	return mapper.TimePointerToPG(&cursor.FavoritedAt)
}

func cursorWordFavoriteID(cursor *dto.WordFavoritesCursor) (pgtype.UUID, error) {
	if cursor == nil {
		return pgtype.UUID{}, nil
	}
	value, err := mapper.StringToUUID(cursor.FavoriteID)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("map cursor favorite_id: %w", err)
	}
	return value, nil
}
