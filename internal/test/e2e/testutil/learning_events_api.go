//go:build e2e

package testutil

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	analyticsservice "learning-video-recommendation-system/internal/analytics/application/service"
	analyticsrepo "learning-video-recommendation-system/internal/analytics/infrastructure/persistence/repository"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
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
