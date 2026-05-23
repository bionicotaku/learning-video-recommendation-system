package main

import (
	"log/slog"

	analyticsservice "learning-video-recommendation-system/internal/analytics/application/service"
	analyticsrepo "learning-video-recommendation-system/internal/analytics/infrastructure/persistence/repository"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/learningevents"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/learningtargets"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/unitcollections"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/unitprogress"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/watchprogress"
	apitx "learning-video-recommendation-system/internal/api/infrastructure/persistence/tx"
	catalogservice "learning-video-recommendation-system/internal/catalog/application/service"
	catalogrepo "learning-video-recommendation-system/internal/catalog/infrastructure/persistence/repository"
	normalizerservice "learning-video-recommendation-system/internal/learningengine/normalizer/application/service"
	normalizerrepo "learning-video-recommendation-system/internal/learningengine/normalizer/infrastructure/persistence/repository"
	learningservice "learning-video-recommendation-system/internal/learningengine/reducer/application/service"
	learningrepo "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/repository"
	learningtx "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/tx"
	semanticservice "learning-video-recommendation-system/internal/semantic/application/service"
	semanticrepo "learning-video-recommendation-system/internal/semantic/infrastructure/persistence/repository"
	userapprepo "learning-video-recommendation-system/internal/user/application/repository"
	userrepo "learning-video-recommendation-system/internal/user/infrastructure/persistence/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
	userUnitStateReader := learningservice.NewGetUserUnitStateUsecase(learningrepo.NewUserUnitStateRepository(pool))
	resetUserUnitProgress := learningservice.NewResetUserUnitProgressUsecase(learningtx.NewManager(pool))

	interactionService := apiservice.NewRecordLearningInteractionsBatchService(recordInteractions, normalizeInteractions, logger)
	quizService := apiservice.NewRecordQuizAttemptService(recordQuiz, normalizeQuiz, logger)
	selfMarkService := apiservice.NewRecordSelfMarkMasteredService(recordSelfMark, normalizeSelfMark, userUnitStateReader, logger)
	resetService := apiservice.NewResetUserUnitProgressService(resetUserUnitProgress)
	return learningevents.NewHandler(interactionService, quizService, selfMarkService, resetService), nil
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
	listCollectionsForUser := apiservice.NewUnitCollectionsService(listCollections, activeCollection)
	return unitcollections.NewHandler(listCollectionsForUser)
}

func buildLearningTargetsHandler(pool *pgxpool.Pool) *learningtargets.Handler {
	activeCollectionReader := learningrepo.NewActiveUnitCollectionReader(pool)
	activeTargetUnitIDs := learningservice.NewGetActiveLearningTargetCoarseUnitIDsUsecase(activeCollectionReader)
	activateTarget := apiservice.NewActivateLearningCollectionService(apitx.NewActivateCollectionManager(pool))
	return learningtargets.NewHandler(activateTarget, activeTargetUnitIDs)
}
