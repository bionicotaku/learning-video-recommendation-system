package service

import (
	"context"
	"errors"
	"time"

	"learning-video-recommendation-system/internal/catalog/application/dto"
	"learning-video-recommendation-system/internal/catalog/application/repository"
	"learning-video-recommendation-system/internal/catalog/domain/model"
)

type Option func(*RecordVideoWatchProgressUsecase)

func WithNow(now func() time.Time) Option {
	return func(u *RecordVideoWatchProgressUsecase) {
		if now != nil {
			u.now = now
		}
	}
}

type RecordVideoWatchProgressUsecase struct {
	writer repository.VideoWatchProgressWriter
	now    func() time.Time
}

func NewRecordVideoWatchProgressUsecase(writer repository.VideoWatchProgressWriter, options ...Option) *RecordVideoWatchProgressUsecase {
	usecase := &RecordVideoWatchProgressUsecase{
		writer: writer,
		now:    func() time.Time { return time.Now().UTC() },
	}
	for _, option := range options {
		option(usecase)
	}
	return usecase
}

func (u *RecordVideoWatchProgressUsecase) Execute(ctx context.Context, request dto.RecordVideoWatchProgressRequest) (dto.RecordVideoWatchProgressResponse, error) {
	if u.writer == nil {
		return dto.RecordVideoWatchProgressResponse{}, errors.New("video watch progress writer is required")
	}
	if request.UserID == "" {
		return dto.RecordVideoWatchProgressResponse{}, validationError("user_id is required")
	}
	if request.VideoID == "" {
		return dto.RecordVideoWatchProgressResponse{}, validationError("video_id is required")
	}
	if request.WatchSessionID == "" {
		return dto.RecordVideoWatchProgressResponse{}, validationError("watch_session_id is required")
	}
	if request.PositionMS < 0 {
		return dto.RecordVideoWatchProgressResponse{}, validationError("position_ms must be non-negative")
	}
	if request.ActiveWatchMS < 0 {
		return dto.RecordVideoWatchProgressResponse{}, validationError("active_watch_ms must be non-negative")
	}

	occurredAt := request.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = u.now()
	}
	occurredAt = occurredAt.UTC()
	now := u.now().UTC()
	if occurredAt.After(now.Add(5 * time.Minute)) {
		return dto.RecordVideoWatchProgressResponse{}, UnprocessableError("occurred_at is too far in the future")
	}
	if occurredAt.Before(now.AddDate(-1, 0, 0)) {
		return dto.RecordVideoWatchProgressResponse{}, UnprocessableError("occurred_at is too far in the past")
	}

	clientContext, err := normalizeJSONObject(request.ClientContext)
	if err != nil {
		if IsValidationError(err) {
			return dto.RecordVideoWatchProgressResponse{}, err
		}
		return dto.RecordVideoWatchProgressResponse{}, validationError("client_context must be a JSON object")
	}
	metadata, err := normalizeJSONObject(request.Metadata)
	if err != nil {
		if IsValidationError(err) {
			return dto.RecordVideoWatchProgressResponse{}, err
		}
		return dto.RecordVideoWatchProgressResponse{}, validationError("metadata must be a JSON object")
	}

	result, err := u.writer.RecordVideoWatchProgress(ctx, model.VideoWatchProgress{
		UserID:         request.UserID,
		VideoID:        request.VideoID,
		WatchSessionID: request.WatchSessionID,
		PositionMS:     request.PositionMS,
		ActiveWatchMS:  request.ActiveWatchMS,
		OccurredAt:     occurredAt,
		SourceSurface:  request.SourceSurface,
		ClientContext:  clientContext,
		Metadata:       metadata,
	})
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrVideoNotFound):
			return dto.RecordVideoWatchProgressResponse{}, NotFoundError("video not found")
		case errors.Is(err, repository.ErrWatchSessionConflict):
			return dto.RecordVideoWatchProgressResponse{}, ConflictError("watch_session_id is already bound to another user or video")
		default:
			return dto.RecordVideoWatchProgressResponse{}, err
		}
	}
	return dto.RecordVideoWatchProgressResponse{Accepted: result.Accepted}, nil
}
