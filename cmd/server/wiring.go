package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/feedback"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/router"

	"github.com/jackc/pgx/v5/pgxpool"
)

func buildHTTPHandler(pool *pgxpool.Pool, logger *slog.Logger, config config) (http.Handler, error) {
	if config.APIGatewayUserinfoHeader == "" {
		return nil, fmt.Errorf("api gateway userinfo header is required")
	}

	learningEvents, err := buildLearningEventsHandler(pool, logger)
	if err != nil {
		return nil, err
	}
	feedHandler, err := buildFeedHandler(pool, logger, config)
	if err != nil {
		return nil, err
	}
	endQuiz := buildEndQuizHandler(pool)
	unitCollections := buildUnitCollectionsHandler(pool)
	learningTargets := buildLearningTargetsHandler(pool)
	videoDetail := buildVideoDetailHandler(pool, config)
	videoLibrary := buildVideoLibraryHandler(pool, config)
	videoInteractions := buildVideoInteractionsHandler(pool)
	watchProgress := buildWatchProgressHandler(pool)
	unitProgress := buildUnitProgressHandler(pool)
	meHandler := buildMeHandler(pool)
	feedbackHandler := buildFeedbackHandler(pool)

	handler := router.New(router.Options{
		Feed:              feedHandler,
		VideoDetail:       videoDetail,
		VideoLibrary:      videoLibrary,
		EndQuiz:           endQuiz,
		UnitCollections:   unitCollections,
		LearningTargets:   learningTargets,
		VideoInteractions: videoInteractions,
		LearningEvents:    learningEvents,
		WatchProgress:     watchProgress,
		UnitProgress:      unitProgress,
		Me:                meHandler,
		Feedback:          feedbackHandler,
	})
	handler = middleware.BodyLimitByPath(1<<20, map[string]int64{"/api/feedback": feedback.MaxRequestBytes})(handler)
	handler = middleware.Recover(handler)
	handler = middleware.Timeout(15 * time.Second)(handler)
	handler = middleware.Logging(logger)(handler)
	handler = auth.PrincipalMiddleware(auth.Options{
		DevMode:               config.DevMode,
		GatewayUserinfoHeader: config.APIGatewayUserinfoHeader,
	})(handler)
	handler = middleware.RequestID(handler)
	return handler, nil
}
