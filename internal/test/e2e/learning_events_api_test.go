//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestE2E_LearningInteractionsBatchHTTPWritesRawAndObservationOnlyState(t *testing.T) {
	h := harness(t)

	userID := h.NewUserID()
	unitLookup := h.NewUnitID()
	unitExposure := h.NewUnitID()
	videoID := h.NewVideoID()
	watchSessionID := h.NewVideoID()
	recommendationRunID := h.NewVideoID()
	startedAt := time.Date(2026, 5, 15, 17, 0, 0, 0, time.UTC)

	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, unitLookup, unitExposure)
	h.SeedCatalogVideo(t, strongSupplyVideo(videoID, unitLookup, 1_000, 2_000, 0, "learning-events-api", 90_000))
	h.SeedVideoWatchSession(t, userID, videoID, watchSessionID, startedAt)

	server := h.LearningEventsAPIServer(t, userID)
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/learning-interactions:batch", `{
		"client_context": {"platform": "ios", "app_version": "1.3.0"},
		"video_id": "`+videoID+`",
		"watch_session_id": "`+watchSessionID+`",
		"recommendation_run_id": "`+recommendationRunID+`",
		"events": [
			{
				"client_event_id": "e2e-lookup-1",
				"event_type": "lookup",
				"source_surface": "video_subtitle",
				"coarse_unit_id": `+itoa64(unitLookup)+`,
				"token_text": "constrain",
				"sentence_index": 0,
				"span_index": 0,
				"occurred_at": "2026-05-15T10:00:01-07:00",
				"lookup_visible_ms": 7200,
				"lookup_sentence_audio_replay_count": 1,
				"lookup_word_audio_play_count": 2,
				"event_payload": {"displayed_base_form": "constrain"}
			},
			{
				"client_event_id": "e2e-exposure-1",
				"event_type": "exposure",
				"source_surface": "video_subtitle",
				"coarse_unit_id": `+itoa64(unitExposure)+`,
				"sentence_index": 1,
				"span_index": 0,
				"occurred_at": "2026-05-15T10:00:05-07:00",
				"exposure_start_ms": 142000,
				"exposure_end_ms": 146300,
				"exposure_count": 1
			}
		]
	}`)
	requireStatus(t, response, http.StatusOK)

	var body struct {
		AcceptedCount  int `json:"accepted_count"`
		InsertedCount  int `json:"inserted_count"`
		DuplicateCount int `json:"duplicate_count"`
	}
	decodeResponse(t, response, &body)
	if body.AcceptedCount != 2 || body.InsertedCount != 2 || body.DuplicateCount != 0 {
		t.Fatalf("response = %+v, want accepted=2 inserted=2 duplicate=0", body)
	}

	assertLearningInteractionContext(t, h.Pool, "e2e-lookup-1", userID, videoID, watchSessionID, recommendationRunID)
	assertLearningInteractionContext(t, h.Pool, "e2e-exposure-1", userID, videoID, watchSessionID, recommendationRunID)

	lookupState := loadLearningState(t, h.Pool, userID, unitLookup)
	if lookupState.Status != "new" || lookupState.ObservationCount != 1 || lookupState.ProgressEventCount != 0 || lookupState.LastProgressQuality != nil {
		t.Fatalf("lookup state = %+v, want observe-only new state", lookupState)
	}
	exposureState := loadLearningState(t, h.Pool, userID, unitExposure)
	if exposureState.Status != "new" || exposureState.ObservationCount != 1 || exposureState.ProgressEventCount != 0 || exposureState.LastProgressQuality != nil {
		t.Fatalf("exposure state = %+v, want observe-only new state", exposureState)
	}
	assertLearningEvent(t, h.Pool, userID, unitLookup, "learning_interaction_event", "lookup", "observe_only", nil)
	assertLearningEvent(t, h.Pool, userID, unitExposure, "learning_interaction_event", "exposure", "observe_only", nil)
}

func TestE2E_QuizAttemptHTTPWritesRawAndProgressesLearningState(t *testing.T) {
	h := harness(t)

	userID := h.NewUserID()
	unitID := h.NewUnitID()
	questionID := h.NewVideoID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, unitID)
	h.SeedQuestion(t, questionID)

	server := h.LearningEventsAPIServer(t, userID)
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/quiz-attempts", `{
		"client_context": {"platform": "ios"},
		"client_event_id": "e2e-quiz-1",
		"question_id": "`+questionID+`",
		"coarse_unit_id": `+itoa64(unitID)+`,
		"trigger_type": "lookup_practice",
		"selected_option_ids": ["correct"],
		"selection_interval_ms": [5000],
		"is_first_try_correct": true,
		"total_elapsed_ms": 5000,
		"shown_at": "2026-05-15T10:00:00-07:00",
		"completed_at": "2026-05-15T10:00:05-07:00"
	}`)
	requireStatus(t, response, http.StatusOK)

	var body struct {
		Accepted    bool   `json:"accepted"`
		QuizEventID string `json:"quiz_event_id"`
		Inserted    bool   `json:"inserted"`
	}
	decodeResponse(t, response, &body)
	if !body.Accepted || body.QuizEventID == "" || !body.Inserted {
		t.Fatalf("response = %+v, want inserted quiz raw fact", body)
	}

	if got := countRows(t, h.Pool, `select count(*) from analytics.quiz_events where user_id = $1 and client_event_id = 'e2e-quiz-1'`, userID); got != 1 {
		t.Fatalf("quiz raw rows = %d, want 1", got)
	}
	q5 := int16(5)
	assertLearningEvent(t, h.Pool, userID, unitID, "quiz_event", "quiz", "affects_progress", &q5)

	state := loadLearningState(t, h.Pool, userID, unitID)
	if state.Status != "learning" || state.ObservationCount != 1 || state.ProgressEventCount != 1 || state.LastProgressQuality == nil || *state.LastProgressQuality != 5 {
		t.Fatalf("state = %+v, want one quality=5 progress event", state)
	}
}

func TestE2E_SelfMarkMasteredHTTPWritesRawAndTerminalMasteredState(t *testing.T) {
	h := harness(t)

	userID := h.NewUserID()
	unitID := h.NewUnitID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, unitID)

	server := h.LearningEventsAPIServer(t, userID)
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/learning-units:mark-mastered", `{
		"client_context": {"platform": "ios"},
		"client_event_id": "e2e-self-mark-1",
		"coarse_unit_id": `+itoa64(unitID)+`,
		"source_surface": "word_detail",
		"token_text": "constrain",
		"occurred_at": "2026-05-15T10:02:00-07:00",
		"event_payload": {"origin": "manual"}
	}`)
	requireStatus(t, response, http.StatusOK)

	var body struct {
		Accepted                   bool   `json:"accepted"`
		LearningInteractionEventID string `json:"learning_interaction_event_id"`
		Inserted                   bool   `json:"inserted"`
	}
	decodeResponse(t, response, &body)
	if !body.Accepted || body.LearningInteractionEventID == "" || !body.Inserted {
		t.Fatalf("response = %+v, want inserted self mark raw fact", body)
	}

	if got := countRows(t, h.Pool, `select count(*) from analytics.learning_interaction_events where user_id = $1 and client_event_id = 'e2e-self-mark-1' and event_type = 'self_mark_mastered'`, userID); got != 1 {
		t.Fatalf("self mark raw rows = %d, want 1", got)
	}
	assertLearningEvent(t, h.Pool, userID, unitID, "learning_interaction_event", "self_mark_mastered", "set_mastered", nil)
	assertTerminalMastered(t, loadLearningState(t, h.Pool, userID, unitID))
}

func TestE2E_LearningEventsHTTPIdempotentRetryDoesNotDoubleReduce(t *testing.T) {
	h := harness(t)

	userID := h.NewUserID()
	unitID := h.NewUnitID()
	questionID := h.NewVideoID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, unitID)
	h.SeedQuestion(t, questionID)

	server := h.LearningEventsAPIServer(t, userID)
	t.Cleanup(server.Close)

	body := `{
		"client_context": {"platform": "ios"},
		"client_event_id": "e2e-quiz-idempotent",
		"question_id": "` + questionID + `",
		"coarse_unit_id": ` + itoa64(unitID) + `,
		"trigger_type": "lookup_practice",
		"selected_option_ids": ["correct"],
		"selection_interval_ms": [5000],
		"is_first_try_correct": true,
		"total_elapsed_ms": 5000,
		"shown_at": "2026-05-15T10:00:00-07:00",
		"completed_at": "2026-05-15T10:00:05-07:00"
	}`
	first := postJSON(t, server, "/api/quiz-attempts", body)
	requireStatus(t, first, http.StatusOK)
	var firstBody struct {
		QuizEventID string `json:"quiz_event_id"`
		Inserted    bool   `json:"inserted"`
	}
	decodeResponse(t, first, &firstBody)
	if !firstBody.Inserted {
		t.Fatalf("first response inserted = false, want true")
	}

	second := postJSON(t, server, "/api/quiz-attempts", body)
	requireStatus(t, second, http.StatusOK)
	var secondBody struct {
		QuizEventID string `json:"quiz_event_id"`
		Inserted    bool   `json:"inserted"`
	}
	decodeResponse(t, second, &secondBody)
	if secondBody.Inserted || secondBody.QuizEventID != firstBody.QuizEventID {
		t.Fatalf("second response = %+v, want duplicate existing id %s", secondBody, firstBody.QuizEventID)
	}

	if got := countRows(t, h.Pool, `select count(*) from analytics.quiz_events where user_id = $1 and client_event_id = 'e2e-quiz-idempotent'`, userID); got != 1 {
		t.Fatalf("quiz raw rows = %d, want 1", got)
	}
	if got := countRows(t, h.Pool, `select count(*) from learning.unit_learning_events where user_id = $1 and coarse_unit_id = $2`, userID, unitID); got != 1 {
		t.Fatalf("learning event rows = %d, want 1", got)
	}
	state := loadLearningState(t, h.Pool, userID, unitID)
	if state.ProgressEventCount != 1 || state.ObservationCount != 1 {
		t.Fatalf("state counters = observation:%d progress:%d, want 1/1", state.ObservationCount, state.ProgressEventCount)
	}
}

func TestE2E_TerminalSelfMarkIgnoresLaterQuizProgress(t *testing.T) {
	h := harness(t)

	userID := h.NewUserID()
	unitID := h.NewUnitID()
	questionID := h.NewVideoID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, unitID)
	h.SeedQuestion(t, questionID)

	server := h.LearningEventsAPIServer(t, userID)
	t.Cleanup(server.Close)

	selfMark := postJSON(t, server, "/api/learning-units:mark-mastered", `{
		"client_context": {"platform": "ios"},
		"client_event_id": "e2e-terminal-self-mark",
		"coarse_unit_id": `+itoa64(unitID)+`,
		"source_surface": "word_detail",
		"occurred_at": "2026-05-15T10:02:00-07:00"
	}`)
	requireStatus(t, selfMark, http.StatusOK)

	quiz := postJSON(t, server, "/api/quiz-attempts", `{
		"client_context": {"platform": "ios"},
		"client_event_id": "e2e-terminal-later-quiz",
		"question_id": "`+questionID+`",
		"coarse_unit_id": `+itoa64(unitID)+`,
		"trigger_type": "lookup_practice",
		"selected_option_ids": ["wrong", "correct"],
		"selection_interval_ms": [1000, 5000],
		"is_first_try_correct": false,
		"total_elapsed_ms": 6000,
		"shown_at": "2026-05-15T10:03:00-07:00",
		"completed_at": "2026-05-15T10:03:06-07:00"
	}`)
	requireStatus(t, quiz, http.StatusOK)

	if got := countRows(t, h.Pool, `select count(*) from learning.unit_learning_events where user_id = $1 and coarse_unit_id = $2`, userID, unitID); got != 2 {
		t.Fatalf("learning event rows = %d, want 2 ledger rows", got)
	}
	assertTerminalMastered(t, loadLearningState(t, h.Pool, userID, unitID))
}

func postJSON(t *testing.T, server *httptest.Server, path string, body string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, server.URL+path, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post json: %v", err)
	}
	return response
}

func requireStatus(t *testing.T, response *http.Response, want int) {
	t.Helper()
	if response.StatusCode != want {
		t.Fatalf("status = %d, want %d: %s", response.StatusCode, want, readResponseBody(t, response))
	}
}

func decodeResponse(t *testing.T, response *http.Response, target any) {
	t.Helper()
	defer response.Body.Close()
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func readResponseBody(t *testing.T, response *http.Response) string {
	t.Helper()
	defer response.Body.Close()
	content, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return string(content)
}

func assertLearningInteractionContext(t *testing.T, db queryer, clientEventID, userID, videoID, watchSessionID, recommendationRunID string) {
	t.Helper()
	var gotVideoID string
	var gotWatchSessionID string
	var gotRecommendationRunID string
	if err := db.QueryRow(context.Background(), `
		select video_id::text, watch_session_id::text, recommendation_run_id::text
		from analytics.learning_interaction_events
		where user_id = $1 and client_event_id = $2
	`, userID, clientEventID).Scan(&gotVideoID, &gotWatchSessionID, &gotRecommendationRunID); err != nil {
		t.Fatalf("load learning interaction context: %v", err)
	}
	if gotVideoID != videoID || gotWatchSessionID != watchSessionID || gotRecommendationRunID != recommendationRunID {
		t.Fatalf("context for %s = %s/%s/%s, want %s/%s/%s", clientEventID, gotVideoID, gotWatchSessionID, gotRecommendationRunID, videoID, watchSessionID, recommendationRunID)
	}
}

func assertLearningEvent(t *testing.T, db queryer, userID string, unitID int64, sourceType string, eventType string, reducerEffect string, progressQuality *int16) {
	t.Helper()
	var gotEventType string
	var gotReducerEffect string
	var gotProgressQuality *int16
	if err := db.QueryRow(context.Background(), `
		select event_type, reducer_effect, progress_quality
		from learning.unit_learning_events
		where user_id = $1
		  and coarse_unit_id = $2
		  and source_type = $3
		  and event_type = $4
	`, userID, unitID, sourceType, eventType).Scan(&gotEventType, &gotReducerEffect, &gotProgressQuality); err != nil {
		t.Fatalf("load learning event: %v", err)
	}
	if gotEventType != eventType || gotReducerEffect != reducerEffect {
		t.Fatalf("learning event = %s/%s, want %s/%s", gotEventType, gotReducerEffect, eventType, reducerEffect)
	}
	if progressQuality == nil {
		if gotProgressQuality != nil {
			t.Fatalf("progress_quality = %v, want nil", *gotProgressQuality)
		}
		return
	}
	if gotProgressQuality == nil || *gotProgressQuality != *progressQuality {
		t.Fatalf("progress_quality = %v, want %d", gotProgressQuality, *progressQuality)
	}
}

type learningStateRow struct {
	Status              string
	IsTarget            bool
	ProgressPercent     float64
	MasteryScore        float64
	ObservationCount    int32
	ProgressEventCount  int32
	LastProgressQuality *int16
	NextReviewIsNull    bool
	SuspendedReason     string
}

func loadLearningState(t *testing.T, db queryer, userID string, unitID int64) learningStateRow {
	t.Helper()
	var row learningStateRow
	if err := db.QueryRow(context.Background(), `
		select
			status,
			is_target,
			progress_percent::float8,
			mastery_score::float8,
			observation_count,
			progress_event_count,
			last_progress_quality,
			next_review_at is null,
			coalesce(suspended_reason, '')
		from learning.user_unit_states
		where user_id = $1 and coarse_unit_id = $2
	`, userID, unitID).Scan(
		&row.Status,
		&row.IsTarget,
		&row.ProgressPercent,
		&row.MasteryScore,
		&row.ObservationCount,
		&row.ProgressEventCount,
		&row.LastProgressQuality,
		&row.NextReviewIsNull,
		&row.SuspendedReason,
	); err != nil {
		t.Fatalf("load learning state: %v", err)
	}
	return row
}

func assertTerminalMastered(t *testing.T, state learningStateRow) {
	t.Helper()
	if state.Status != "mastered" ||
		state.IsTarget ||
		state.ProgressPercent != 100 ||
		state.MasteryScore != 1 ||
		!state.NextReviewIsNull ||
		state.SuspendedReason != "" {
		t.Fatalf("state = %+v, want terminal mastered", state)
	}
}

func countRows(t *testing.T, db queryer, sql string, args ...any) int {
	t.Helper()
	var count int
	if err := db.QueryRow(context.Background(), sql, args...).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return count
}

type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func itoa64(value int64) string {
	return strconv.FormatInt(value, 10)
}
