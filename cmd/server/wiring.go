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
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/endquiz"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/feed"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/learningevents"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/watchprogress"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/router"
	catalogservice "learning-video-recommendation-system/internal/catalog/application/service"
	catalogrepo "learning-video-recommendation-system/internal/catalog/infrastructure/persistence/repository"
	normalizerservice "learning-video-recommendation-system/internal/learningengine/normalizer/application/service"
	normalizerrepo "learning-video-recommendation-system/internal/learningengine/normalizer/infrastructure/persistence/repository"
	learningservice "learning-video-recommendation-system/internal/learningengine/reducer/application/service"
	learningtx "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/tx"
	recommendationservice "learning-video-recommendation-system/internal/recommendation/application/service"
	recommendationusecase "learning-video-recommendation-system/internal/recommendation/application/usecase"
	recommendationaggregator "learning-video-recommendation-system/internal/recommendation/domain/aggregator"
	recommendationexplain "learning-video-recommendation-system/internal/recommendation/domain/explain"
	recommendationplanner "learning-video-recommendation-system/internal/recommendation/domain/planner"
	recommendationranking "learning-video-recommendation-system/internal/recommendation/domain/ranking"
	recommendationselector "learning-video-recommendation-system/internal/recommendation/domain/selector"
	recommendationrepo "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/repository"
	recommendationtx "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/tx"

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
	feedHandler, err := buildFeedHandler(pool, logger, config)
	if err != nil {
		return nil, err
	}
	endQuiz := buildEndQuizHandler(pool)
	watchProgress := buildWatchProgressHandler(pool)

	handler := router.New(router.Options{
		Feed:           feedHandler,
		EndQuiz:        endQuiz,
		LearningEvents: learningEvents,
		WatchProgress:  watchProgress,
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

func buildWatchProgressHandler(pool *pgxpool.Pool) *watchprogress.Handler {
	writer := catalogrepo.NewVideoWatchProgressWriter(pool)
	recordWatchProgress := catalogservice.NewRecordVideoWatchProgressUsecase(writer)
	return watchprogress.NewHandler(recordWatchProgress)
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

	lookupReader := catalogrepo.NewFeedLookupReader(pool)
	feedVideos := catalogservice.NewFeedVideoLookupUsecase(lookupReader)
	unitLabels := catalogservice.NewUnitLabelLookupUsecase(lookupReader)
	feedService := apiservice.NewFeedService(
		recommendations,
		feedVideos,
		unitLabels,
		apiservice.NewPublicAssetURLBuilder(config.PublicAssetBaseURL),
		logger,
	)
	return feed.NewHandler(feedService), nil
}

func buildRecommendationUsecase(pool *pgxpool.Pool) (*recommendationusecase.GenerateVideoRecommendationsService, error) {
	unitServing := recommendationrepo.NewUnitServingStateRepository(pool)
	videoServing := recommendationrepo.NewVideoServingStateRepository(pool)

	return recommendationusecase.NewGenerateVideoRecommendationsPipeline(
		recommendationservice.NewDefaultContextAssembler(
			recommendationrepo.NewLearningStateReader(pool),
			recommendationrepo.NewUnitInventoryReader(pool),
			unitServing,
		),
		recommendationplanner.NewDefaultDemandPlanner(),
		recommendationservice.NewDefaultCandidateGenerator(recommendationrepo.NewRecommendableVideoUnitReader(pool)),
		recommendationservice.NewDefaultEvidenceResolver(
			recommendationrepo.NewSemanticSpanReader(pool),
			recommendationrepo.NewTranscriptSentenceReader(pool),
		),
		recommendationaggregator.NewDefaultVideoEvidenceAggregator(),
		recommendationranking.NewDefaultVideoRanker(),
		recommendationselector.NewDefaultVideoSelector(),
		recommendationexplain.NewDefaultExplanationBuilder(),
		recommendationservice.NewDefaultVideoStateEnricher(
			videoServing,
			recommendationrepo.NewVideoUserStateReader(pool),
		),
		recommendationservice.NewDefaultRecommendationResultWriter(
			recommendationtx.NewManager(pool),
			recommendationservice.NewDefaultAuditWriter(recommendationrepo.NewRecommendationAuditRepository(pool)),
			recommendationservice.NewDefaultServingStateManager(unitServing, videoServing),
		),
	)
}
