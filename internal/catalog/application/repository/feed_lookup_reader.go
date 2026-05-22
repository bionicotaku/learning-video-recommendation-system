package repository

import (
	"context"

	"learning-video-recommendation-system/internal/catalog/domain/model"
)

type FeedVideoReader interface {
	ListFeedVideosByIDs(ctx context.Context, userID string, videoIDs []string) ([]model.FeedVideoDisplay, error)
}

type VideoDetailReader interface {
	GetVideoDetailByID(ctx context.Context, userID string, videoID string) (model.VideoDetail, error)
}

type UnitLabelReader interface {
	ListUnitLabelsByIDs(ctx context.Context, coarseUnitIDs []int64) ([]model.UnitLabel, error)
}
