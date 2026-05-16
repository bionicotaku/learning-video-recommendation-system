package repository

import (
	"context"
	"errors"

	"learning-video-recommendation-system/internal/catalog/domain/model"
)

var (
	ErrVideoNotFound        = errors.New("video not found")
	ErrWatchSessionConflict = errors.New("watch session conflict")
)

type VideoWatchProgressWriter interface {
	RecordVideoWatchProgress(ctx context.Context, request model.VideoWatchProgress) (model.VideoWatchProgressResult, error)
}
