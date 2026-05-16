package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	analyticsservice "learning-video-recommendation-system/internal/analytics/application/service"
	analyticsrepo "learning-video-recommendation-system/internal/analytics/infrastructure/persistence/repository"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/learningevents"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/router"
	normalizerservice "learning-video-recommendation-system/internal/learningengine/normalizer/application/service"
	normalizerrepo "learning-video-recommendation-system/internal/learningengine/normalizer/infrastructure/persistence/repository"
	learningservice "learning-video-recommendation-system/internal/learningengine/reducer/application/service"
	learningtx "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/tx"

	"github.com/jackc/pgx/v5/pgxpool"
)

func buildHTTPHandler(pool *pgxpool.Pool, logger *slog.Logger, config config) (http.Handler, error) {
	if strings.TrimSpace(config.TrustedUserIDHeader) == "" {
		return nil, fmt.Errorf("trusted user id header is required")
	}

	learningEvents, err := buildLearningEventsHandler(pool, logger)
	if err != nil {
		return nil, err
	}

	handler := router.New(router.Options{
		LearningEvents: learningEvents,
	})
	handler = middleware.BodyLimit(1 << 20)(handler)
	handler = middleware.Recover(handler)
	handler = middleware.Timeout(15 * time.Second)(handler)
	handler = middleware.Logging(logger)(handler)
	handler = auth.TrustedHeaderPrincipalMiddleware(config.TrustedUserIDHeader)(handler)
	handler = middleware.RequestID(handler)
	return handler, nil
}

func buildLearningEventsHandler(pool *pgxpool.Pool, logger *slog.Logger) (*learningevents.Handler, error) {
	rawWriter := analyticsrepo.NewRawEventWriter(pool)
	recordInteractions := analyticsservice.NewRecordLearningInteractionsBatchUsecase(rawWriter)
	recordQuiz := analyticsservice.NewRecordQuizAttemptUsecase(rawWriter)
	recordSelfMark := analyticsservice.NewRecordSelfMarkMasteredUsecase(rawWriter)

	learningRecorder := learningservice.NewRecordLearningEventsUsecase(learningtx.NewManager(pool))
	interactionReader := normalizerrepo.NewRawLearningInteractionReader(pool)
	quizReader := normalizerrepo.NewRawQuizEventReader(pool)
	normalizeInteractions := normalizerservice.NewNormalizeLearningInteractionsByIDsUsecase(interactionReader, learningRecorder)
	normalizeQuiz := normalizerservice.NewNormalizeQuizAttemptByIDUsecase(quizReader, learningRecorder)
	normalizeSelfMark := normalizerservice.NewNormalizeSelfMarkMasteredByIDUsecase(interactionReader, learningRecorder)

	interactionService := apiservice.NewRecordLearningInteractionsBatchService(recordInteractions, normalizeInteractions, logger)
	quizService := apiservice.NewRecordQuizAttemptService(recordQuiz, normalizeQuiz, logger)
	selfMarkService := apiservice.NewRecordSelfMarkMasteredService(recordSelfMark, normalizeSelfMark, logger)
	return learningevents.NewHandler(interactionService, quizService, selfMarkService), nil
}
