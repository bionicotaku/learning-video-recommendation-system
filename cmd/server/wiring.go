package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	analyticsservice "learning-video-recommendation-system/internal/analytics/application/service"
	analyticsrepo "learning-video-recommendation-system/internal/analytics/infrastructure/persistence/repository"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/endquiz"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/feed"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/feedback"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/learningevents"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/me"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/unitcollections"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/unitprogress"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/videointeractions"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/watchprogress"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/router"
	apitx "learning-video-recommendation-system/internal/api/infrastructure/persistence/tx"
	catalogservice "learning-video-recommendation-system/internal/catalog/application/service"
	catalogrepo "learning-video-recommendation-system/internal/catalog/infrastructure/persistence/repository"
	normalizerservice "learning-video-recommendation-system/internal/learningengine/normalizer/application/service"
	normalizerrepo "learning-video-recommendation-system/internal/learningengine/normalizer/infrastructure/persistence/repository"
	learningservice "learning-video-recommendation-system/internal/learningengine/reducer/application/service"
	learningrepo "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/repository"
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
	semanticservice "learning-video-recommendation-system/internal/semantic/application/service"
	semanticrepo "learning-video-recommendation-system/internal/semantic/infrastructure/persistence/repository"
	userapprepo "learning-video-recommendation-system/internal/user/application/repository"
	userservice "learning-video-recommendation-system/internal/user/application/service"
	userrepo "learning-video-recommendation-system/internal/user/infrastructure/persistence/repository"

	"github.com/jackc/pgx/v5"
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
	videoInteractions := buildVideoInteractionsHandler(pool)
	watchProgress := buildWatchProgressHandler(pool)
	unitProgress := buildUnitProgressHandler(pool)
	meHandler := buildMeHandler(pool)
	feedbackHandler := buildFeedbackHandler(pool)

	handler := router.New(router.Options{
		Feed:              feedHandler,
		EndQuiz:           endQuiz,
		UnitCollections:   unitCollections,
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

func buildLearningEventsHandler(pool *pgxpool.Pool, logger *slog.Logger) (*learningevents.Handler, error) {
	rawWriter := analyticsrepo.NewRawEventWriterWithActivityStats(pool)
	recordInteractions := analyticsservice.NewRecordLearningInteractionsBatchUsecase(rawWriter)
	recordQuiz := analyticsservice.NewRecordQuizAttemptUsecase(rawWriter)
	recordSelfMark := analyticsservice.NewRecordSelfMarkMasteredUsecase(analyticsrepo.NewRawEventWriter(pool))

	learningRecorder := learningservice.NewRecordLearningEventsUsecase(learningtx.NewManagerWithActivityStats(pool))
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
	writer := catalogrepo.NewVideoWatchProgressWriter(pool, catalogrepo.WithWatchProgressActivityStats(func(tx pgx.Tx) userapprepo.ActivityStatsRecorder {
		return userrepo.NewRepository(tx)
	}))
	recordWatchProgress := catalogservice.NewRecordVideoWatchProgressUsecase(writer)
	return watchprogress.NewHandler(recordWatchProgress)
}

func buildUnitProgressHandler(pool *pgxpool.Pool) *unitprogress.Handler {
	reader := learningrepo.NewUserUnitProgressReader(pool)
	listProgress := learningservice.NewListUserUnitProgressUsecase(reader)
	return unitprogress.NewHandler(listProgress)
}

func buildUnitCollectionsHandler(pool *pgxpool.Pool) *unitcollections.Handler {
	reader := semanticrepo.NewUnitCollectionReader(pool)
	listCollections := semanticservice.NewListUnitCollectionsUsecase(reader)
	activeCollectionReader := learningrepo.NewActiveUnitCollectionReader(pool)
	activeCollection := learningservice.NewGetActiveUnitCollectionUsecase(activeCollectionReader)
	activeTargetUnitIDs := learningservice.NewGetActiveLearningTargetCoarseUnitIDsUsecase(activeCollectionReader)
	listCollectionsForUser := apiservice.NewUnitCollectionsService(listCollections, activeCollection)
	activateTarget := apiservice.NewActivateLearningCollectionService(apitx.NewActivateCollectionManager(pool))
	return unitcollections.NewHandler(listCollectionsForUser, activateTarget, activeTargetUnitIDs)
}

func buildMeHandler(pool *pgxpool.Pool) *me.Handler {
	repository := userrepo.NewRepository(pool)
	getMe := userservice.NewGetMeUsecase(repository, repository)
	return me.NewHandler(getMe)
}

func buildFeedbackHandler(pool *pgxpool.Pool) *feedback.Handler {
	writer := userrepo.NewFeedbackWriter(pool)
	submitFeedback := userservice.NewSubmitFeedbackUsecase(writer)
	return feedback.NewHandler(submitFeedback)
}

func buildVideoInteractionsHandler(pool *pgxpool.Pool) *videointeractions.Handler {
	writer := catalogrepo.NewVideoInteractionWriter(pool)
	setLike := catalogservice.NewSetVideoLikeUsecase(writer)
	setFavorite := catalogservice.NewSetVideoFavoriteUsecase(writer)
	return videointeractions.NewHandler(setLike, setFavorite)
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
	recommendable := recommendationrepo.NewRecommendableVideoUnitReader(pool)

	return recommendationusecase.NewGenerateVideoRecommendationsPipeline(
		recommendationservice.NewDefaultContextAssembler(
			recommendationrepo.NewLearningStateReader(pool),
			recommendationrepo.NewUnitInventoryReader(pool),
			unitServing,
			recommendationservice.NewRecallQueueService(recommendationrepo.NewRecallQueueRepository(pool)),
			recommendable,
		),
		recommendationplanner.NewDefaultDemandPlanner(),
		recommendationservice.NewDefaultCandidateGenerator(recommendable),
		recommendationservice.NewDefaultEvidenceResolver(),
		recommendationaggregator.NewDefaultVideoEvidenceAggregator(),
		recommendationranking.NewDefaultVideoRanker(),
		recommendationselector.NewDefaultVideoSelector(),
		recommendationservice.NewDefaultVideoFillService(recommendationrepo.NewVideoFillCandidateReader(pool)),
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
