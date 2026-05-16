package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	apprepo "learning-video-recommendation-system/internal/catalog/application/repository"
	"learning-video-recommendation-system/internal/catalog/domain/model"
	"learning-video-recommendation-system/internal/catalog/infrastructure/persistence/mapper"
	catalogsqlc "learning-video-recommendation-system/internal/catalog/infrastructure/persistence/sqlcgen"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VideoWatchProgressWriter struct {
	pool *pgxpool.Pool
}

func NewVideoWatchProgressWriter(pool *pgxpool.Pool) *VideoWatchProgressWriter {
	return &VideoWatchProgressWriter{pool: pool}
}

func (w *VideoWatchProgressWriter) RecordVideoWatchProgress(ctx context.Context, request model.VideoWatchProgress) (model.VideoWatchProgressResult, error) {
	if w.pool == nil {
		return model.VideoWatchProgressResult{}, errors.New("pg pool is required")
	}

	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return model.VideoWatchProgressResult{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := catalogsqlc.New(tx)

	userID, err := mapper.StringToUUID(request.UserID)
	if err != nil {
		return model.VideoWatchProgressResult{}, fmt.Errorf("map user_id: %w", err)
	}
	videoID, err := mapper.StringToUUID(request.VideoID)
	if err != nil {
		return model.VideoWatchProgressResult{}, fmt.Errorf("map video_id: %w", err)
	}
	watchSessionID, err := mapper.StringToUUID(request.WatchSessionID)
	if err != nil {
		return model.VideoWatchProgressResult{}, fmt.Errorf("map watch_session_id: %w", err)
	}

	durationMS, err := queries.GetVideoDurationMS(ctx, videoID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.VideoWatchProgressResult{}, apprepo.ErrVideoNotFound
		}
		return model.VideoWatchProgressResult{}, err
	}

	metadata, err := metadataWithSourceSurface(request.Metadata, request.SourceSurface)
	if err != nil {
		return model.VideoWatchProgressResult{}, err
	}

	occurredAt := request.OccurredAt.UTC()
	sessionParams := catalogsqlc.UpsertVideoWatchSessionParams{
		WatchSessionID: watchSessionID,
		UserID:         userID,
		VideoID:        videoID,
		OccurredAt:     mapper.TimePointerToPG(&occurredAt),
		PositionMs:     request.PositionMS,
		ActiveWatchMs:  request.ActiveWatchMS,
		DurationMs:     durationMS,
		ClientContext:  request.ClientContext,
		Metadata:       metadata,
	}
	session, err := queries.UpsertVideoWatchSession(ctx, sessionParams)
	if errors.Is(err, pgx.ErrNoRows) {
		session, err = queries.UpsertVideoWatchSession(ctx, sessionParams)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.VideoWatchProgressResult{}, apprepo.ErrWatchSessionConflict
		}
		return model.VideoWatchProgressResult{}, err
	}

	if err := queries.UpsertVideoUserStateFromWatchProgress(ctx, catalogsqlc.UpsertVideoUserStateFromWatchProgressParams{
		UserID:             userID,
		VideoID:            videoID,
		StartedAt:          session.StartedAt,
		LastSeenAt:         session.LastSeenAt,
		CreatedSession:     session.CreatedSession,
		CompletedSession:   session.CompletedSession,
		LastPositionMs:     session.LastPositionMs,
		MaxPositionMs:      session.MaxPositionMs,
		DeltaActiveWatchMs: session.DeltaActiveWatchMs,
	}); err != nil {
		return model.VideoWatchProgressResult{}, err
	}

	if err := queries.UpsertVideoEngagementStatsFromWatchProgress(ctx, catalogsqlc.UpsertVideoEngagementStatsFromWatchProgressParams{
		VideoID:            videoID,
		CreatedSession:     session.CreatedSession,
		CompletedSession:   session.CompletedSession,
		DeltaActiveWatchMs: session.DeltaActiveWatchMs,
	}); err != nil {
		return model.VideoWatchProgressResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.VideoWatchProgressResult{}, err
	}

	return model.VideoWatchProgressResult{
		Accepted:           true,
		CreatedSession:     session.CreatedSession,
		CompletedSession:   session.CompletedSession,
		DeltaActiveWatchMS: session.DeltaActiveWatchMs,
	}, nil
}

func metadataWithSourceSurface(metadata []byte, sourceSurface string) ([]byte, error) {
	if len(metadata) == 0 {
		metadata = []byte(`{}`)
	}
	var object map[string]any
	if err := json.Unmarshal(metadata, &object); err != nil {
		return nil, err
	}
	if object == nil {
		object = map[string]any{}
	}
	if sourceSurface != "" {
		object["source_surface"] = sourceSurface
	}
	return json.Marshal(object)
}
