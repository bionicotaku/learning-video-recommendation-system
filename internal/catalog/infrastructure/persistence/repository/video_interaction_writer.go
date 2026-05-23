package repository

import (
	"context"
	"errors"
	"fmt"

	apprepo "learning-video-recommendation-system/internal/catalog/application/repository"
	"learning-video-recommendation-system/internal/catalog/domain/model"
	"learning-video-recommendation-system/internal/catalog/infrastructure/persistence/mapper"
	catalogsqlc "learning-video-recommendation-system/internal/catalog/infrastructure/persistence/sqlcgen"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VideoInteractionWriter struct {
	pool *pgxpool.Pool
}

func NewVideoInteractionWriter(pool *pgxpool.Pool) *VideoInteractionWriter {
	return &VideoInteractionWriter{pool: pool}
}

func (w *VideoInteractionWriter) SetVideoLike(ctx context.Context, command model.VideoLikeCommand) (model.VideoLikeResult, error) {
	if w.pool == nil {
		return model.VideoLikeResult{}, errors.New("pg pool is required")
	}

	userID, videoID, err := mapVideoInteractionIDs(command.UserID, command.VideoID)
	if err != nil {
		return model.VideoLikeResult{}, err
	}

	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return model.VideoLikeResult{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := catalogsqlc.New(tx)
	var row catalogsqlc.SetVideoLikedRow
	if command.Enabled {
		row, err = queries.SetVideoLiked(ctx, catalogsqlc.SetVideoLikedParams{
			UserID:     userID,
			VideoID:    videoID,
			OccurredAt: mapper.TimePointerToPG(&command.OccurredAt),
		})
	} else {
		unliked, queryErr := queries.SetVideoUnliked(ctx, catalogsqlc.SetVideoUnlikedParams{
			UserID:     userID,
			VideoID:    videoID,
			OccurredAt: mapper.TimePointerToPG(&command.OccurredAt),
		})
		err = queryErr
		row = catalogsqlc.SetVideoLikedRow{
			VideoID:   unliked.VideoID,
			HasLiked:  unliked.HasLiked,
			LikeCount: unliked.LikeCount,
		}
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.VideoLikeResult{}, apprepo.ErrVideoNotFound
		}
		return model.VideoLikeResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.VideoLikeResult{}, err
	}

	return model.VideoLikeResult{
		VideoID:   mapper.UUIDToString(row.VideoID),
		HasLiked:  row.HasLiked,
		LikeCount: row.LikeCount,
	}, nil
}

func (w *VideoInteractionWriter) SetVideoFavorite(ctx context.Context, command model.VideoFavoriteCommand) (model.VideoFavoriteResult, error) {
	if w.pool == nil {
		return model.VideoFavoriteResult{}, errors.New("pg pool is required")
	}

	userID, videoID, err := mapVideoInteractionIDs(command.UserID, command.VideoID)
	if err != nil {
		return model.VideoFavoriteResult{}, err
	}

	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return model.VideoFavoriteResult{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := catalogsqlc.New(tx)
	var row catalogsqlc.SetVideoFavoritedRow
	if command.Enabled {
		row, err = queries.SetVideoFavorited(ctx, catalogsqlc.SetVideoFavoritedParams{
			UserID:     userID,
			VideoID:    videoID,
			OccurredAt: mapper.TimePointerToPG(&command.OccurredAt),
		})
	} else {
		unfavorited, queryErr := queries.SetVideoUnfavorited(ctx, catalogsqlc.SetVideoUnfavoritedParams{
			UserID:     userID,
			VideoID:    videoID,
			OccurredAt: mapper.TimePointerToPG(&command.OccurredAt),
		})
		err = queryErr
		row = catalogsqlc.SetVideoFavoritedRow{
			VideoID:       unfavorited.VideoID,
			HasFavorited:  unfavorited.HasFavorited,
			FavoriteCount: unfavorited.FavoriteCount,
		}
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.VideoFavoriteResult{}, apprepo.ErrVideoNotFound
		}
		return model.VideoFavoriteResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.VideoFavoriteResult{}, err
	}

	return model.VideoFavoriteResult{
		VideoID:       mapper.UUIDToString(row.VideoID),
		HasFavorited:  row.HasFavorited,
		FavoriteCount: row.FavoriteCount,
	}, nil
}

func mapVideoInteractionIDs(userID string, videoID string) (userUUID pgtype.UUID, videoUUID pgtype.UUID, err error) {
	userUUID, err = mapper.StringToUUID(userID)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, fmt.Errorf("map user_id: %w", err)
	}
	videoUUID, err = mapper.StringToUUID(videoID)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, fmt.Errorf("map video_id: %w", err)
	}
	return userUUID, videoUUID, nil
}
