package main

import (
	"log/slog"

	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/endquiz"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/feed"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/videodetail"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/videointeractions"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/videolibrary"
	catalogservice "learning-video-recommendation-system/internal/catalog/application/service"
	catalogrepo "learning-video-recommendation-system/internal/catalog/infrastructure/persistence/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

func buildVideoInteractionsHandler(pool *pgxpool.Pool) *videointeractions.Handler {
	writer := catalogrepo.NewVideoInteractionWriter(pool)
	setLike := catalogservice.NewSetVideoLikeUsecase(writer)
	setFavorite := catalogservice.NewSetVideoFavoriteUsecase(writer)
	return videointeractions.NewHandler(setLike, setFavorite)
}

func buildVideoDetailHandler(pool *pgxpool.Pool, config config) *videodetail.Handler {
	reader := catalogrepo.NewVideoPresentationReader(pool)
	lookup := catalogservice.NewGetVideoDetailUsecase(reader)
	service := apiservice.NewVideoDetailService(lookup, apiservice.NewPublicAssetURLBuilder(config.PublicAssetBaseURL))
	return videodetail.NewHandler(service)
}

func buildVideoLibraryHandler(pool *pgxpool.Pool, config config) *videolibrary.Handler {
	reader := catalogrepo.NewVideoLibraryReader(pool)
	listFavorites := catalogservice.NewListVideoFavoritesUsecase(reader)
	listHistory := catalogservice.NewListVideoHistoryUsecase(reader)
	service := apiservice.NewVideoLibraryService(listFavorites, listHistory, apiservice.NewPublicAssetURLBuilder(config.PublicAssetBaseURL))
	return videolibrary.NewHandler(service)
}

func buildEndQuizHandler(pool *pgxpool.Pool) *endquiz.Handler {
	reader := catalogrepo.NewEndQuizQuestionReader(pool)
	lookup := catalogservice.NewEndQuizQuestionLookupUsecase(reader)
	return endquiz.NewHandler(lookup)
}

func buildFeedHandler(pool *pgxpool.Pool, logger *slog.Logger, config config) (*feed.Handler, error) {
	recommendations, err := buildRecommendationUsecase(pool)
	if err != nil {
		return nil, err
	}

	videoReader := catalogrepo.NewVideoPresentationReader(pool)
	unitLabelReader := catalogrepo.NewUnitLabelReader(pool)
	feedVideos := catalogservice.NewFeedVideoLookupUsecase(videoReader)
	unitLabels := catalogservice.NewUnitLabelLookupUsecase(unitLabelReader)
	feedService := apiservice.NewFeedService(
		recommendations,
		feedVideos,
		unitLabels,
		apiservice.NewPublicAssetURLBuilder(config.PublicAssetBaseURL),
		logger,
	)
	return feed.NewHandler(feedService), nil
}
