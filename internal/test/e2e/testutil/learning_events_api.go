//go:build e2e

package testutil

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	analyticsservice "learning-video-recommendation-system/internal/analytics/application/service"
	analyticsrepo "learning-video-recommendation-system/internal/analytics/infrastructure/persistence/repository"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/endquiz"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/feed"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/feedback"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/learningevents"
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
	semanticservice "learning-video-recommendation-system/internal/semantic/application/service"
	semanticrepo "learning-video-recommendation-system/internal/semantic/infrastructure/persistence/repository"
	userservice "learning-video-recommendation-system/internal/user/application/service"
	userrepo "learning-video-recommendation-system/internal/user/infrastructure/persistence/repository"
)

// LearningEventsAPIServer returns a real HTTP server wired to real analytics,
// normalizer, and reducer usecases over the harness database.
func (h *Harness) LearningEventsAPIServer(t *testing.T, userID string) *httptest.Server {
	t.Helper()

	rawWriter := analyticsrepo.NewRawEventWriter(h.Pool)
	recordInteractions := analyticsservice.NewRecordLearningInteractionsBatchUsecase(rawWriter)
	recordQuiz := analyticsservice.NewRecordQuizAttemptUsecase(rawWriter)
	recordSelfMark := analyticsservice.NewRecordSelfMarkMasteredUsecase(rawWriter)

	learningRecorder := learningservice.NewRecordLearningEventsUsecase(learningtx.NewManager(h.Pool))
	interactionReader := normalizerrepo.NewRawLearningInteractionReader(h.Pool)
	quizReader := normalizerrepo.NewRawQuizEventReader(h.Pool)
	normalizeInteractions := normalizerservice.NewNormalizeLearningInteractionsByIDsUsecase(interactionReader, learningRecorder)
	normalizeQuiz := normalizerservice.NewNormalizeQuizAttemptByIDUsecase(quizReader, learningRecorder)
	normalizeSelfMark := normalizerservice.NewNormalizeSelfMarkMasteredByIDUsecase(interactionReader, learningRecorder)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	learningEvents := learningevents.NewHandler(
		apiservice.NewRecordLearningInteractionsBatchService(recordInteractions, normalizeInteractions, logger),
		apiservice.NewRecordQuizAttemptService(recordQuiz, normalizeQuiz, logger),
		apiservice.NewRecordSelfMarkMasteredService(recordSelfMark, normalizeSelfMark, logger),
	)

	handler := router.New(router.Options{LearningEvents: learningEvents})
	handler = middleware.BodyLimit(1 << 20)(handler)
	handler = middleware.Recover(handler)
	handler = middleware.Timeout(15 * time.Second)(handler)
	handler = auth.FakePrincipalMiddleware(auth.Principal{UserID: userID})(handler)
	handler = middleware.RequestID(handler)

	return httptest.NewServer(handler)
}

// WatchProgressAPIServer returns a real HTTP server wired to the real Catalog
// watch-progress usecase over the harness database.
func (h *Harness) WatchProgressAPIServer(t *testing.T, userID string) *httptest.Server {
	t.Helper()

	writer := catalogrepo.NewVideoWatchProgressWriter(h.Pool)
	watchProgress := watchprogress.NewHandler(catalogservice.NewRecordVideoWatchProgressUsecase(writer))

	handler := router.New(router.Options{WatchProgress: watchProgress})
	handler = middleware.BodyLimit(1 << 20)(handler)
	handler = middleware.Recover(handler)
	handler = middleware.Timeout(15 * time.Second)(handler)
	handler = auth.FakePrincipalMiddleware(auth.Principal{UserID: userID})(handler)
	handler = middleware.RequestID(handler)

	return httptest.NewServer(handler)
}

// APIServer returns a real HTTP server with all currently implemented API
// route groups wired over the harness database and a fixed test principal.
func (h *Harness) APIServer(t *testing.T, userID string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(h.apiHandler(t, auth.FakePrincipalMiddleware(auth.Principal{UserID: userID})))
}

// DevModeAPIServer returns a real HTTP server with production principal
// middleware configured to allow the DEV_MODE Authorization bearer fallback.
func (h *Harness) DevModeAPIServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(h.apiHandler(t, auth.PrincipalMiddleware(auth.Options{
		DevMode:               true,
		GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo",
	})))
}

func (h *Harness) apiHandler(t *testing.T, principalMiddleware func(http.Handler) http.Handler) http.Handler {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	learningEvents := h.learningEventsHandler(logger)
	watchProgress := watchprogress.NewHandler(catalogservice.NewRecordVideoWatchProgressUsecase(catalogrepo.NewVideoWatchProgressWriter(h.Pool)))
	videoInteractions := videointeractions.NewHandler(
		catalogservice.NewSetVideoLikeUsecase(catalogrepo.NewVideoInteractionWriter(h.Pool)),
		catalogservice.NewSetVideoFavoriteUsecase(catalogrepo.NewVideoInteractionWriter(h.Pool)),
	)
	endQuiz := endquiz.NewHandler(catalogservice.NewEndQuizQuestionLookupUsecase(catalogrepo.NewEndQuizQuestionReader(h.Pool)))
	unitProgress := unitprogress.NewHandler(learningservice.NewListUserUnitProgressUsecase(learningrepo.NewUserUnitProgressReader(h.Pool)))
	activeCollectionReader := learningrepo.NewActiveUnitCollectionReader(h.Pool)
	unitCollections := unitcollections.NewHandler(
		apiservice.NewUnitCollectionsService(
			semanticservice.NewListUnitCollectionsUsecase(semanticrepo.NewUnitCollectionReader(h.Pool)),
			learningservice.NewGetActiveUnitCollectionUsecase(activeCollectionReader),
		),
		apiservice.NewActivateLearningCollectionService(apitx.NewActivateCollectionManager(h.Pool)),
		learningservice.NewGetActiveLearningTargetCoarseUnitIDsUsecase(activeCollectionReader),
	)

	lookupReader := catalogrepo.NewFeedLookupReader(h.Pool)
	feedService := apiservice.NewFeedService(
		h.RecommendationUsecase(),
		catalogservice.NewFeedVideoLookupUsecase(lookupReader),
		catalogservice.NewUnitLabelLookupUsecase(lookupReader),
		apiservice.NewPublicAssetURLBuilder("https://cdn.example.com/assets"),
		logger,
	)
	feedHandler := feed.NewHandler(feedService)

	handler := router.New(router.Options{
		Feed:              feedHandler,
		EndQuiz:           endQuiz,
		UnitCollections:   unitCollections,
		VideoInteractions: videoInteractions,
		LearningEvents:    learningEvents,
		WatchProgress:     watchProgress,
		UnitProgress:      unitProgress,
		Feedback: feedback.NewHandler(
			userservice.NewSubmitFeedbackUsecase(userrepo.NewFeedbackWriter(h.Pool)),
		),
	})
	handler = middleware.BodyLimitByPath(1<<20, map[string]int64{"/api/feedback": feedback.MaxRequestBytes})(handler)
	handler = middleware.Recover(handler)
	handler = middleware.Timeout(15 * time.Second)(handler)
	handler = principalMiddleware(handler)
	handler = middleware.RequestID(handler)
	return handler
}

func (h *Harness) learningEventsHandler(logger *slog.Logger) *learningevents.Handler {
	rawWriter := analyticsrepo.NewRawEventWriter(h.Pool)
	recordInteractions := analyticsservice.NewRecordLearningInteractionsBatchUsecase(rawWriter)
	recordQuiz := analyticsservice.NewRecordQuizAttemptUsecase(rawWriter)
	recordSelfMark := analyticsservice.NewRecordSelfMarkMasteredUsecase(rawWriter)

	learningRecorder := learningservice.NewRecordLearningEventsUsecase(learningtx.NewManager(h.Pool))
	interactionReader := normalizerrepo.NewRawLearningInteractionReader(h.Pool)
	quizReader := normalizerrepo.NewRawQuizEventReader(h.Pool)
	normalizeInteractions := normalizerservice.NewNormalizeLearningInteractionsByIDsUsecase(interactionReader, learningRecorder)
	normalizeQuiz := normalizerservice.NewNormalizeQuizAttemptByIDUsecase(quizReader, learningRecorder)
	normalizeSelfMark := normalizerservice.NewNormalizeSelfMarkMasteredByIDUsecase(interactionReader, learningRecorder)

	return learningevents.NewHandler(
		apiservice.NewRecordLearningInteractionsBatchService(recordInteractions, normalizeInteractions, logger),
		apiservice.NewRecordQuizAttemptService(recordQuiz, normalizeQuiz, logger),
		apiservice.NewRecordSelfMarkMasteredService(recordSelfMark, normalizeSelfMark, logger),
	)
}

func (h *Harness) SeedVideoWatchSession(t *testing.T, userID string, videoID string, watchSessionID string, startedAt time.Time) {
	t.Helper()
	if _, err := h.Pool.Exec(
		context.Background(),
		`insert into analytics.video_watch_events (
			watch_session_id,
			user_id,
			video_id,
			started_at,
			last_seen_at,
			client_context,
			metadata
		) values (
			$1::uuid,
			$2::uuid,
			$3::uuid,
			$4,
			$4,
			'{}'::jsonb,
			'{}'::jsonb
		) on conflict (watch_session_id) do nothing`,
		watchSessionID,
		userID,
		videoID,
		startedAt,
	); err != nil {
		failNow(t, "seed analytics.video_watch_events: %v", err)
	}
}

func (h *Harness) SeedQuestion(t *testing.T, questionID string) {
	t.Helper()
	if _, err := h.Pool.Exec(context.Background(), `insert into catalog.questions (question_id) values ($1::uuid) on conflict (question_id) do nothing`, questionID); err != nil {
		failNow(t, "seed catalog.questions: %v", err)
	}
}

func (h *Harness) SeedEndQuizQuestion(t *testing.T, questionID string, coarseUnitID int64, videoID string) {
	t.Helper()
	if _, err := h.Pool.Exec(context.Background(), `
		insert into catalog.questions (
			question_id,
			scope_type,
			question_type,
			coarse_unit_id,
			target_text,
			video_id,
			context_sentence_index,
			context_span_index,
			context_start_ms,
			context_end_ms,
			content_payload,
			status,
			created_at
		) values (
			$1::uuid,
			'video_unit',
			'context_meaning_choice',
			$2,
			'test target',
			$3::uuid,
			0,
			0,
			1000,
			2000,
			jsonb_build_object(
				'question', 'What does the target mean?',
				'options', jsonb_build_array(
					jsonb_build_object('id', 'correct', 'text', 'Correct meaning'),
					jsonb_build_object('id', 'wrong', 'text', 'Wrong meaning')
				),
				'explanation', 'Correct explanation',
				'context_text', 'A sentence with the target.'
			),
			'active',
			now()
		)
		on conflict (question_id) do update
		set scope_type = excluded.scope_type,
		    question_type = excluded.question_type,
		    coarse_unit_id = excluded.coarse_unit_id,
		    target_text = excluded.target_text,
		    video_id = excluded.video_id,
		    context_sentence_index = excluded.context_sentence_index,
		    context_span_index = excluded.context_span_index,
		    context_start_ms = excluded.context_start_ms,
		    context_end_ms = excluded.context_end_ms,
		    content_payload = excluded.content_payload,
		    status = excluded.status,
		    created_at = excluded.created_at`,
		questionID,
		coarseUnitID,
		videoID,
	); err != nil {
		failNow(t, "seed end quiz question: %v", err)
	}
}
