package repository

import (
	"context"
	"errors"

	"learning-video-recommendation-system/internal/catalog/application/dto"
	"learning-video-recommendation-system/internal/catalog/domain/model"
)

var ErrWordFavoriteProjectionNotFound = errors.New("word favorite projection not found")

type WordFavoriteWriteOutcome string

const (
	WordFavoriteWriteApplied        WordFavoriteWriteOutcome = "applied"
	WordFavoriteWriteStale          WordFavoriteWriteOutcome = "stale"
	WordFavoriteWriteTargetNotFound WordFavoriteWriteOutcome = "target_not_found"
)

type WordFavoriteRepository interface {
	HasCoarseFavorite(ctx context.Context, key model.WordFavoriteCoarseKey) (bool, error)
	HasTokenFavorite(ctx context.Context, key model.WordFavoriteTokenKey) (bool, error)
	GetVideoContext(ctx context.Context, key model.WordFavoriteVideoContextKey) (model.WordFavoriteVideoContext, error)
	SetCoarseFavorite(ctx context.Context, command model.SetCoarseWordFavoriteCommand) (WordFavoriteWriteOutcome, error)
	SetTokenFavorite(ctx context.Context, command model.SetTokenWordFavoriteCommand) (WordFavoriteWriteOutcome, error)
	UnsetCoarseFavorite(ctx context.Context, command model.UnsetCoarseWordFavoriteCommand) error
	UnsetTokenFavorite(ctx context.Context, command model.UnsetTokenWordFavoriteCommand) error
	ListWordFavorites(ctx context.Context, query dto.ListWordFavoritesQuery) ([]model.WordFavoriteListItem, error)
}
