package learningevents_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/learningevents"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/router"
)

func TestLearningInteractionsBatchPassesPrincipalUserIDAndReturnsAcceptedRawFacts(t *testing.T) {
	recorder := &fakeLearningInteractionRecorder{
		response: apvdto.RecordLearningInteractionsBatchResponse{
			AcceptedCount:  1,
			InsertedCount:  1,
			DuplicateCount: 0,
			Events: []apvdto.AcceptedLearningInteractionEvent{{
				ClientEventID:              "evt-1",
				LearningInteractionEventID: "11111111-1111-1111-1111-111111111111",
				Inserted:                   true,
			}},
		},
	}
	server := newTestServer(recorder, &fakeQuizAttemptRecorder{}, &fakeSelfMarkMasteredRecorder{})
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/learning-interactions:batch", `{
		"client_context": {"platform": "ios"},
		"video_id": "33333333-3333-3333-3333-333333333333",
		"watch_session_id": "44444444-4444-4444-4444-444444444444",
		"recommendation_run_id": "55555555-5555-5555-5555-555555555555",
		"events": [{
			"client_event_id": "evt-1",
			"event_type": "lookup",
			"source_surface": "video_subtitle",
			"coarse_unit_id": 101,
			"token_text": "constrain",
			"sentence_index": 12,
			"span_index": 4,
			"occurred_at": "2026-05-15T17:00:01Z",
			"event_payload": {"displayed_base_form": "constrain"}
		}]
	}`)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if recorder.userID != "user-1" {
		t.Fatalf("expected principal user id to be passed, got %q", recorder.userID)
	}
	if recorder.request.VideoID != "33333333-3333-3333-3333-333333333333" ||
		recorder.request.WatchSessionID != "44444444-4444-4444-4444-444444444444" ||
		recorder.request.RecommendationRunID != "55555555-5555-5555-5555-555555555555" {
		t.Fatalf("unexpected video context: %+v", recorder.request)
	}
	if len(recorder.request.Events) != 1 ||
		recorder.request.Events[0].SentenceIndex == nil || *recorder.request.Events[0].SentenceIndex != 12 ||
		recorder.request.Events[0].SpanIndex == nil || *recorder.request.Events[0].SpanIndex != 4 {
		t.Fatalf("unexpected event indexes: %+v", recorder.request.Events)
	}

	var body struct {
		AcceptedCount int `json:"accepted_count"`
		Events        []struct {
			ClientEventID string `json:"client_event_id"`
			Inserted      bool   `json:"inserted"`
		} `json:"events"`
	}
	decodeJSON(t, response, &body)
	if body.AcceptedCount != 1 || len(body.Events) != 1 || body.Events[0].ClientEventID != "evt-1" || !body.Events[0].Inserted {
		t.Fatalf("unexpected response body: %#v", body)
	}
}

func TestQuizAttemptsMapsApplicationValidationErrorToInvalidRequest(t *testing.T) {
	server := newTestServer(&fakeLearningInteractionRecorder{}, &fakeQuizAttemptRecorder{
		err: apiservice.InvalidRequestError("selected_option_ids is required"),
	}, &fakeSelfMarkMasteredRecorder{})
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/quiz-attempts", `{
		"client_event_id": "quiz-1",
		"question_id": "44444444-4444-4444-4444-444444444444",
		"coarse_unit_id": 101,
		"trigger_type": "lookup_practice",
		"selected_option_ids": ["correct"],
		"selection_interval_ms": [1200],
		"is_first_try_correct": true,
		"total_elapsed_ms": 1200,
		"shown_at": "2026-05-15T17:01:00Z",
		"completed_at": "2026-05-15T17:01:04Z"
	}`)

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.StatusCode)
	}
	var body struct {
		Error struct {
			Code      string `json:"code"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	decodeJSON(t, response, &body)
	if body.Error.Code != "invalid_request" {
		t.Fatalf("unexpected error code: %s", body.Error.Code)
	}
	if body.Error.RequestID == "" {
		t.Fatalf("expected request id in error response")
	}
}

func TestQuizAttemptsMapsInternalErrorToInternalError(t *testing.T) {
	server := newTestServer(&fakeLearningInteractionRecorder{}, &fakeQuizAttemptRecorder{
		err: errors.New("database unavailable"),
	}, &fakeSelfMarkMasteredRecorder{})
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/quiz-attempts", `{
		"client_event_id": "quiz-1",
		"question_id": "44444444-4444-4444-4444-444444444444",
		"coarse_unit_id": 101,
		"trigger_type": "lookup_practice",
		"selected_option_ids": ["correct"],
		"selection_interval_ms": [1200],
		"is_first_try_correct": true,
		"total_elapsed_ms": 1200,
		"shown_at": "2026-05-15T17:01:00Z",
		"completed_at": "2026-05-15T17:01:04Z"
	}`)

	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", response.StatusCode)
	}
}

func TestLearningInteractionsBatchRejectsMissingPrincipal(t *testing.T) {
	group := learningevents.NewHandler(&fakeLearningInteractionRecorder{}, &fakeQuizAttemptRecorder{}, &fakeSelfMarkMasteredRecorder{}, &fakeResetUserUnitProgressRecorder{})
	handler := router.New(router.Options{LearningEvents: group})
	server := httptest.NewServer(middleware.RequestID(handler))
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/learning-interactions:batch", `{"events":[]}`)

	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.StatusCode)
	}
	var body struct {
		Error struct {
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	decodeJSON(t, response, &body)
	if body.Error.RequestID == "" {
		t.Fatalf("expected request id in error response")
	}
}

func TestLearningInteractionsBatchRejectsSelfMarkMastered(t *testing.T) {
	recorder := &fakeLearningInteractionRecorder{}
	server := newTestServer(recorder, &fakeQuizAttemptRecorder{}, &fakeSelfMarkMasteredRecorder{})
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/learning-interactions:batch", `{
		"video_id": "33333333-3333-3333-3333-333333333333",
		"watch_session_id": "44444444-4444-4444-4444-444444444444",
		"events": [{
			"client_event_id": "evt-1",
			"event_type": "self_mark_mastered",
			"source_surface": "word_detail",
			"coarse_unit_id": 101,
			"occurred_at": "2026-05-15T17:00:01Z"
		}]
	}`)

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if recorder.called {
		t.Fatalf("expected batch usecase not to be called")
	}
}

func TestLearningInteractionsBatchRequiresSentenceAndSpanIndexesForCurrentEventTypes(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			name: "lookup sentence_index",
			body: `{
				"video_id": "33333333-3333-3333-3333-333333333333",
				"watch_session_id": "44444444-4444-4444-4444-444444444444",
				"events": [{
					"client_event_id": "lookup-1",
					"event_type": "lookup",
					"source_surface": "video_subtitle",
					"token_text": "constrain",
					"span_index": 4,
					"occurred_at": "2026-05-15T17:00:01Z"
				}]
			}`,
		},
		{
			name: "lookup span_index",
			body: `{
				"video_id": "33333333-3333-3333-3333-333333333333",
				"watch_session_id": "44444444-4444-4444-4444-444444444444",
				"events": [{
					"client_event_id": "lookup-1",
					"event_type": "lookup",
					"source_surface": "video_subtitle",
					"token_text": "constrain",
					"sentence_index": 12,
					"occurred_at": "2026-05-15T17:00:01Z"
				}]
			}`,
		},
		{
			name: "exposure sentence_index",
			body: `{
				"video_id": "33333333-3333-3333-3333-333333333333",
				"watch_session_id": "44444444-4444-4444-4444-444444444444",
				"events": [{
					"client_event_id": "exposure-1",
					"event_type": "exposure",
					"source_surface": "video_subtitle",
					"coarse_unit_id": 101,
					"span_index": 4,
					"occurred_at": "2026-05-15T17:00:01Z"
				}]
			}`,
		},
		{
			name: "exposure span_index",
			body: `{
				"video_id": "33333333-3333-3333-3333-333333333333",
				"watch_session_id": "44444444-4444-4444-4444-444444444444",
				"events": [{
					"client_event_id": "exposure-1",
					"event_type": "exposure",
					"source_surface": "video_subtitle",
					"coarse_unit_id": 101,
					"sentence_index": 12,
					"occurred_at": "2026-05-15T17:00:01Z"
				}]
			}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &fakeLearningInteractionRecorder{}
			server := newTestServer(recorder, &fakeQuizAttemptRecorder{}, &fakeSelfMarkMasteredRecorder{})
			t.Cleanup(server.Close)

			response := postJSON(t, server, "/api/learning-interactions:batch", tc.body)
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
			}
			if recorder.called {
				t.Fatalf("expected batch usecase not to be called")
			}
		})
	}
}

func TestLearningInteractionsBatchRejectsMissingVideoContext(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			name: "video_id",
			body: `{
				"watch_session_id": "44444444-4444-4444-4444-444444444444",
				"events": [{"client_event_id": "evt-1", "event_type": "lookup", "source_surface": "video_subtitle", "token_text": "test", "occurred_at": "2026-05-15T17:00:01Z"}]
			}`,
		},
		{
			name: "watch_session_id",
			body: `{
				"video_id": "33333333-3333-3333-3333-333333333333",
				"events": [{"client_event_id": "evt-1", "event_type": "lookup", "source_surface": "video_subtitle", "token_text": "test", "occurred_at": "2026-05-15T17:00:01Z"}]
			}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &fakeLearningInteractionRecorder{}
			server := newTestServer(recorder, &fakeQuizAttemptRecorder{}, &fakeSelfMarkMasteredRecorder{})
			t.Cleanup(server.Close)

			response := postJSON(t, server, "/api/learning-interactions:batch", tc.body)
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
			}
			if recorder.called {
				t.Fatalf("expected batch usecase not to be called")
			}
		})
	}
}

func TestLearningInteractionsBatchRejectsEventLevelVideoContext(t *testing.T) {
	recorder := &fakeLearningInteractionRecorder{}
	server := newTestServer(recorder, &fakeQuizAttemptRecorder{}, &fakeSelfMarkMasteredRecorder{})
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/learning-interactions:batch", `{
		"video_id": "33333333-3333-3333-3333-333333333333",
		"watch_session_id": "44444444-4444-4444-4444-444444444444",
		"events": [{
			"client_event_id": "evt-1",
			"event_type": "lookup",
			"source_surface": "video_subtitle",
			"video_id": "33333333-3333-3333-3333-333333333333",
			"token_text": "constrain",
			"occurred_at": "2026-05-15T17:00:01Z"
		}]
	}`)

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if recorder.called {
		t.Fatalf("expected batch usecase not to be called")
	}
}

func TestLearningInteractionsBatchRequiresJSONContentType(t *testing.T) {
	body := `{
		"video_id": "33333333-3333-3333-3333-333333333333",
		"watch_session_id": "44444444-4444-4444-4444-444444444444",
		"events": [{
			"client_event_id": "evt-1",
			"event_type": "lookup",
			"source_surface": "video_subtitle",
			"token_text": "constrain",
			"sentence_index": 12,
			"span_index": 4,
			"occurred_at": "2026-05-15T17:00:01Z"
		}]
	}`
	cases := []struct {
		name        string
		contentType string
	}{
		{name: "missing"},
		{name: "wrong", contentType: "text/plain"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &fakeLearningInteractionRecorder{}
			server := newTestServer(recorder, &fakeQuizAttemptRecorder{}, &fakeSelfMarkMasteredRecorder{})
			t.Cleanup(server.Close)

			response := postRaw(t, server, "/api/learning-interactions:batch", body, tc.contentType)
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
			}
			if recorder.called {
				t.Fatalf("expected batch usecase not to be called")
			}
		})
	}
}

func TestLearningInteractionsBatchValidatesPositiveCoarseUnitID(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			name: "exposure zero",
			body: `{
				"video_id": "33333333-3333-3333-3333-333333333333",
				"watch_session_id": "44444444-4444-4444-4444-444444444444",
				"events": [{
					"client_event_id": "exposure-1",
					"event_type": "exposure",
					"source_surface": "video_subtitle",
					"coarse_unit_id": 0,
					"sentence_index": 12,
					"span_index": 4,
					"occurred_at": "2026-05-15T17:00:01Z"
				}]
			}`,
		},
		{
			name: "lookup negative",
			body: `{
				"video_id": "33333333-3333-3333-3333-333333333333",
				"watch_session_id": "44444444-4444-4444-4444-444444444444",
				"events": [{
					"client_event_id": "lookup-1",
					"event_type": "lookup",
					"source_surface": "video_subtitle",
					"coarse_unit_id": -1,
					"token_text": "constrain",
					"sentence_index": 12,
					"span_index": 4,
					"occurred_at": "2026-05-15T17:00:01Z"
				}]
			}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &fakeLearningInteractionRecorder{}
			server := newTestServer(recorder, &fakeQuizAttemptRecorder{}, &fakeSelfMarkMasteredRecorder{})
			t.Cleanup(server.Close)

			response := postJSON(t, server, "/api/learning-interactions:batch", tc.body)
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
			}
			if recorder.called {
				t.Fatalf("expected batch usecase not to be called")
			}
		})
	}
}

func TestLearningInteractionsBatchAllowsUnmappedLookupWithoutCoarseUnitID(t *testing.T) {
	recorder := &fakeLearningInteractionRecorder{
		response: apvdto.RecordLearningInteractionsBatchResponse{
			AcceptedCount: 1,
			InsertedCount: 1,
			Events: []apvdto.AcceptedLearningInteractionEvent{{
				ClientEventID:              "lookup-1",
				LearningInteractionEventID: "11111111-1111-1111-1111-111111111111",
				Inserted:                   true,
			}},
		},
	}
	server := newTestServer(recorder, &fakeQuizAttemptRecorder{}, &fakeSelfMarkMasteredRecorder{})
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/learning-interactions:batch", `{
		"video_id": "33333333-3333-3333-3333-333333333333",
		"watch_session_id": "44444444-4444-4444-4444-444444444444",
		"events": [{
			"client_event_id": "lookup-1",
			"event_type": "lookup",
			"source_surface": "video_subtitle",
			"token_text": "unknown",
			"sentence_index": 12,
			"span_index": 4,
			"occurred_at": "2026-05-15T17:00:01Z"
		}]
	}`)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if len(recorder.request.Events) != 1 || recorder.request.Events[0].CoarseUnitID != nil {
		t.Fatalf("expected lookup without coarse_unit_id to pass, got %+v", recorder.request.Events)
	}
}

func TestQuizAttemptsRejectsUnsupportedTriggerType(t *testing.T) {
	for _, triggerType := range []string{"practice_now", "scheduled_review"} {
		t.Run(triggerType, func(t *testing.T) {
			recorder := &fakeQuizAttemptRecorder{}
			server := newTestServer(&fakeLearningInteractionRecorder{}, recorder, &fakeSelfMarkMasteredRecorder{})
			t.Cleanup(server.Close)

			response := postJSON(t, server, "/api/quiz-attempts", `{
				"client_event_id": "quiz-1",
				"question_id": "44444444-4444-4444-4444-444444444444",
				"coarse_unit_id": 101,
				"trigger_type": "`+triggerType+`",
				"selected_option_ids": ["correct"],
				"selection_interval_ms": [1200],
				"is_first_try_correct": true,
				"total_elapsed_ms": 1200,
				"shown_at": "2026-05-15T17:01:00Z",
				"completed_at": "2026-05-15T17:01:04Z"
			}`)
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
			}
			if recorder.userID != "" {
				t.Fatalf("expected quiz usecase not to be called")
			}
		})
	}
}

func TestQuizAttemptsRequiresTotalElapsedMS(t *testing.T) {
	recorder := &fakeQuizAttemptRecorder{}
	server := newTestServer(&fakeLearningInteractionRecorder{}, recorder, &fakeSelfMarkMasteredRecorder{})
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/quiz-attempts", `{
		"client_event_id": "quiz-1",
		"question_id": "44444444-4444-4444-4444-444444444444",
		"coarse_unit_id": 101,
		"trigger_type": "lookup_practice",
		"selected_option_ids": ["correct"],
		"selection_interval_ms": [1200],
		"is_first_try_correct": true,
		"shown_at": "2026-05-15T17:01:00Z",
		"completed_at": "2026-05-15T17:01:04Z"
	}`)

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if recorder.userID != "" {
		t.Fatalf("expected quiz usecase not to be called")
	}
}

func TestQuizAndSelfMarkRequirePositiveCoarseUnitID(t *testing.T) {
	t.Run("quiz zero", func(t *testing.T) {
		recorder := &fakeQuizAttemptRecorder{}
		server := newTestServer(&fakeLearningInteractionRecorder{}, recorder, &fakeSelfMarkMasteredRecorder{})
		t.Cleanup(server.Close)

		response := postJSON(t, server, "/api/quiz-attempts", `{
			"client_event_id": "quiz-1",
			"question_id": "44444444-4444-4444-4444-444444444444",
			"coarse_unit_id": 0,
			"trigger_type": "lookup_practice",
			"selected_option_ids": ["correct"],
			"selection_interval_ms": [1200],
			"is_first_try_correct": true,
			"total_elapsed_ms": 1200,
			"shown_at": "2026-05-15T17:01:00Z",
			"completed_at": "2026-05-15T17:01:04Z"
		}`)
		if response.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
		}
		if recorder.userID != "" {
			t.Fatalf("expected quiz usecase not to be called")
		}
	})

	t.Run("self mark negative", func(t *testing.T) {
		recorder := &fakeSelfMarkMasteredRecorder{}
		server := newTestServer(&fakeLearningInteractionRecorder{}, &fakeQuizAttemptRecorder{}, recorder)
		t.Cleanup(server.Close)

		response := postJSON(t, server, "/api/learning-units:mark-mastered", `{
			"client_event_id": "self-mark-1",
			"coarse_unit_id": -1,
			"source_surface": "word_detail",
			"occurred_at": "2026-05-15T17:02:00Z"
		}`)
		if response.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
		}
		if recorder.userID != "" {
			t.Fatalf("expected self mark usecase not to be called")
		}
	})
}

func TestSelfMarkMasteredPassesPrincipalAndBodyUnitAndReturnsAcceptedRawFact(t *testing.T) {
	recorder := &fakeSelfMarkMasteredRecorder{
		response: apvdto.RecordSelfMarkMasteredResponse{
			Accepted:                   true,
			LearningInteractionEventID: "22222222-2222-2222-2222-222222222222",
			Inserted:                   true,
		},
	}
	server := newTestServer(&fakeLearningInteractionRecorder{}, &fakeQuizAttemptRecorder{}, recorder)
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/learning-units:mark-mastered", `{
		"client_context": {"platform": "ios"},
		"client_event_id": "self-mark-1",
		"coarse_unit_id": 101,
		"source_surface": "word_detail",
		"video_id": "33333333-3333-3333-3333-333333333333",
		"occurred_at": "2026-05-15T17:02:00Z",
		"event_payload": {"origin": "manual"}
	}`)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if recorder.userID != "user-1" {
		t.Fatalf("expected principal user id to be passed, got %q", recorder.userID)
	}
	if recorder.coarseUnitID != 101 {
		t.Fatalf("expected body coarse unit id 101, got %d", recorder.coarseUnitID)
	}

	var body struct {
		Accepted                   bool   `json:"accepted"`
		LearningInteractionEventID string `json:"learning_interaction_event_id"`
		Inserted                   bool   `json:"inserted"`
	}
	decodeJSON(t, response, &body)
	if !body.Accepted || body.LearningInteractionEventID != "22222222-2222-2222-2222-222222222222" || !body.Inserted {
		t.Fatalf("unexpected response body: %#v", body)
	}
}

func TestSelfMarkMasteredOldPathDoesNotMatch(t *testing.T) {
	recorder := &fakeSelfMarkMasteredRecorder{}
	server := newTestServer(&fakeLearningInteractionRecorder{}, &fakeQuizAttemptRecorder{}, recorder)
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/learning-units/101:mark-mastered", `{
		"client_event_id": "self-mark-1",
		"coarse_unit_id": 101,
		"source_surface": "word_detail",
		"occurred_at": "2026-05-15T17:02:00Z"
	}`)
	if response.StatusCode == http.StatusOK {
		t.Fatalf("expected old path not to succeed")
	}
	if recorder.userID != "" {
		t.Fatalf("expected self mark usecase not to be called")
	}
}

func TestSelfMarkMasteredRejectsMissingCoarseUnitID(t *testing.T) {
	recorder := &fakeSelfMarkMasteredRecorder{}
	server := newTestServer(&fakeLearningInteractionRecorder{}, &fakeQuizAttemptRecorder{}, recorder)
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/learning-units:mark-mastered", `{
		"client_event_id": "self-mark-1",
		"source_surface": "word_detail",
		"occurred_at": "2026-05-15T17:02:00Z"
	}`)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if recorder.userID != "" {
		t.Fatalf("expected self mark usecase not to be called")
	}
}

func TestSelfMarkMasteredRejectsNonObjectEventPayload(t *testing.T) {
	server := newTestServer(&fakeLearningInteractionRecorder{}, &fakeQuizAttemptRecorder{}, &fakeSelfMarkMasteredRecorder{})
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/learning-units:mark-mastered", `{
		"client_event_id": "self-mark-1",
		"coarse_unit_id": 101,
		"source_surface": "word_detail",
		"occurred_at": "2026-05-15T17:02:00Z",
		"event_payload": []
	}`)

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.StatusCode)
	}
}

func TestSelfMarkMasteredMapsContextDeadlineToServiceUnavailable(t *testing.T) {
	server := newTestServer(&fakeLearningInteractionRecorder{}, &fakeQuizAttemptRecorder{}, &fakeSelfMarkMasteredRecorder{
		err: context.DeadlineExceeded,
	})
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/learning-units:mark-mastered", `{
		"client_event_id": "self-mark-1",
		"coarse_unit_id": 101,
		"source_surface": "word_detail",
		"occurred_at": "2026-05-15T17:02:00Z"
	}`)

	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", response.StatusCode)
	}
	var body struct {
		Error struct {
			Code      string `json:"code"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	decodeJSON(t, response, &body)
	if body.Error.Code != "service_unavailable" {
		t.Fatalf("unexpected error code: %s", body.Error.Code)
	}
	if body.Error.RequestID == "" {
		t.Fatalf("expected request id in error response")
	}
}

func TestResetUnlearnedPassesPrincipalAndBodyUnitAndReturnsAcceptedLearningEvent(t *testing.T) {
	recorder := &fakeResetUserUnitProgressRecorder{
		response: apvdto.ResetUserUnitProgressResponse{
			Accepted:            true,
			UnitLearningEventID: "22222222-2222-2222-2222-222222222222",
			Inserted:            true,
		},
	}
	server := newTestServer(&fakeLearningInteractionRecorder{}, &fakeQuizAttemptRecorder{}, &fakeSelfMarkMasteredRecorder{}, recorder)
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/learning-units:reset-unlearned", `{
		"client_context": {"platform": "ios"},
		"client_event_id": "reset-1",
		"coarse_unit_id": 101,
		"source_surface": "word_detail",
		"video_id": "33333333-3333-3333-3333-333333333333",
		"occurred_at": "2026-05-15T17:02:00Z",
		"event_payload": {"origin": "manual"}
	}`)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if recorder.userID != "user-1" {
		t.Fatalf("expected principal user id to be passed, got %q", recorder.userID)
	}
	if recorder.coarseUnitID != 101 {
		t.Fatalf("expected body coarse unit id 101, got %d", recorder.coarseUnitID)
	}

	var body struct {
		Accepted            bool   `json:"accepted"`
		UnitLearningEventID string `json:"unit_learning_event_id"`
		Inserted            bool   `json:"inserted"`
	}
	decodeJSON(t, response, &body)
	if !body.Accepted || body.UnitLearningEventID != "22222222-2222-2222-2222-222222222222" || !body.Inserted {
		t.Fatalf("unexpected response body: %#v", body)
	}
}

func TestResetUnlearnedRejectsMissingCoarseUnitID(t *testing.T) {
	recorder := &fakeResetUserUnitProgressRecorder{}
	server := newTestServer(&fakeLearningInteractionRecorder{}, &fakeQuizAttemptRecorder{}, &fakeSelfMarkMasteredRecorder{}, recorder)
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/learning-units:reset-unlearned", `{
		"client_event_id": "reset-1",
		"source_surface": "word_detail",
		"occurred_at": "2026-05-15T17:02:00Z"
	}`)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if recorder.userID != "" {
		t.Fatalf("expected reset usecase not to be called")
	}
}

func TestResetUnlearnedRejectsNonObjectEventPayload(t *testing.T) {
	server := newTestServer(&fakeLearningInteractionRecorder{}, &fakeQuizAttemptRecorder{}, &fakeSelfMarkMasteredRecorder{}, &fakeResetUserUnitProgressRecorder{})
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/learning-units:reset-unlearned", `{
		"client_event_id": "reset-1",
		"coarse_unit_id": 101,
		"source_surface": "word_detail",
		"occurred_at": "2026-05-15T17:02:00Z",
		"event_payload": []
	}`)

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.StatusCode)
	}
}

func TestResetUnlearnedMapsContextDeadlineToServiceUnavailable(t *testing.T) {
	server := newTestServer(&fakeLearningInteractionRecorder{}, &fakeQuizAttemptRecorder{}, &fakeSelfMarkMasteredRecorder{}, &fakeResetUserUnitProgressRecorder{
		err: context.DeadlineExceeded,
	})
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/learning-units:reset-unlearned", `{
		"client_event_id": "reset-1",
		"coarse_unit_id": 101,
		"source_surface": "word_detail",
		"occurred_at": "2026-05-15T17:02:00Z"
	}`)

	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", response.StatusCode)
	}
	var body struct {
		Error struct {
			Code      string `json:"code"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	decodeJSON(t, response, &body)
	if body.Error.Code != "service_unavailable" {
		t.Fatalf("unexpected error code: %s", body.Error.Code)
	}
	if body.Error.RequestID == "" {
		t.Fatalf("expected request id in error response")
	}
}

func TestGatewayUserinfoPrincipalMiddlewareInjectsPrincipal(t *testing.T) {
	recorder := &fakeSelfMarkMasteredRecorder{
		response: apvdto.RecordSelfMarkMasteredResponse{Accepted: true, LearningInteractionEventID: "22222222-2222-2222-2222-222222222222", Inserted: false},
	}
	group := learningevents.NewHandler(&fakeLearningInteractionRecorder{}, &fakeQuizAttemptRecorder{}, recorder, &fakeResetUserUnitProgressRecorder{})
	handler := router.New(router.Options{LearningEvents: group})
	handler = middleware.RequestID(auth.PrincipalMiddleware(auth.Options{GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo"})(handler))
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/learning-units:mark-mastered", bytes.NewBufferString(`{
		"client_event_id": "self-mark-1",
		"coarse_unit_id": 101,
		"source_surface": "word_detail",
		"occurred_at": "2026-05-15T17:02:00Z"
	}`))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Apigateway-Api-Userinfo", "eyJzdWIiOiJ0cnVzdGVkLXVzZXIifQ")

	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("post json: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if recorder.userID != "trusted-user" {
		t.Fatalf("expected gateway userinfo user id, got %q", recorder.userID)
	}
}

func TestDevModeAuthorizationFallbackInjectsPrincipal(t *testing.T) {
	recorder := &fakeSelfMarkMasteredRecorder{
		response: apvdto.RecordSelfMarkMasteredResponse{Accepted: true, LearningInteractionEventID: "22222222-2222-2222-2222-222222222222", Inserted: false},
	}
	group := learningevents.NewHandler(&fakeLearningInteractionRecorder{}, &fakeQuizAttemptRecorder{}, recorder, &fakeResetUserUnitProgressRecorder{})
	handler := router.New(router.Options{LearningEvents: group})
	handler = middleware.RequestID(auth.PrincipalMiddleware(auth.Options{
		DevMode:               true,
		GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo",
	})(handler))
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/learning-units:mark-mastered", bytes.NewBufferString(`{
		"client_event_id": "self-mark-1",
		"coarse_unit_id": 101,
		"source_surface": "word_detail",
		"occurred_at": "2026-05-15T17:02:00Z"
	}`))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer e30.eyJzdWIiOiJ1c2VyLWRldiJ9.sig")

	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("post json: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if recorder.userID != "user-dev" {
		t.Fatalf("expected dev fallback user id, got %q", recorder.userID)
	}
}

func newTestServer(
	interactions learningevents.RecordLearningInteractionsBatchService,
	quiz learningevents.RecordQuizAttemptService,
	selfMark learningevents.RecordSelfMarkMasteredService,
	reset ...learningevents.ResetUserUnitProgressService,
) *httptest.Server {
	resetService := learningevents.ResetUserUnitProgressService(&fakeResetUserUnitProgressRecorder{})
	if len(reset) > 0 {
		resetService = reset[0]
	}
	group := learningevents.NewHandler(interactions, quiz, selfMark, resetService)
	handler := router.New(router.Options{LearningEvents: group})
	handler = middleware.RequestID(auth.FakePrincipalMiddleware(auth.Principal{UserID: "user-1"})(handler))
	return httptest.NewServer(handler)
}

func postJSON(t *testing.T, server *httptest.Server, path string, body string) *http.Response {
	t.Helper()
	return postRaw(t, server, path, body, "application/json")
}

func postRaw(t *testing.T, server *httptest.Server, path string, body string, contentType string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, server.URL+path, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("post json: %v", err)
	}
	return response
}

func decodeJSON(t *testing.T, response *http.Response, out any) {
	t.Helper()
	defer response.Body.Close()
	if err := json.NewDecoder(response.Body).Decode(out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func readBody(t *testing.T, response *http.Response) string {
	t.Helper()
	defer response.Body.Close()
	var buffer bytes.Buffer
	if _, err := buffer.ReadFrom(response.Body); err != nil {
		t.Fatalf("read response: %v", err)
	}
	return buffer.String()
}

type fakeLearningInteractionRecorder struct {
	userID   string
	called   bool
	request  apvdto.RecordLearningInteractionsBatchRequest
	response apvdto.RecordLearningInteractionsBatchResponse
	err      error
}

func (f *fakeLearningInteractionRecorder) Execute(ctx context.Context, request apvdto.RecordLearningInteractionsBatchRequest) (apvdto.RecordLearningInteractionsBatchResponse, error) {
	f.called = true
	f.userID = request.UserID
	f.request = request
	return f.response, f.err
}

type fakeQuizAttemptRecorder struct {
	userID   string
	response apvdto.RecordQuizAttemptResponse
	err      error
}

func (f *fakeQuizAttemptRecorder) Execute(ctx context.Context, request apvdto.RecordQuizAttemptRequest) (apvdto.RecordQuizAttemptResponse, error) {
	f.userID = request.UserID
	return f.response, f.err
}

type fakeSelfMarkMasteredRecorder struct {
	userID       string
	coarseUnitID int64
	response     apvdto.RecordSelfMarkMasteredResponse
	err          error
}

func (f *fakeSelfMarkMasteredRecorder) Execute(ctx context.Context, request apvdto.RecordSelfMarkMasteredRequest) (apvdto.RecordSelfMarkMasteredResponse, error) {
	f.userID = request.UserID
	f.coarseUnitID = request.CoarseUnitID
	return f.response, f.err
}

type fakeResetUserUnitProgressRecorder struct {
	userID       string
	coarseUnitID int64
	response     apvdto.ResetUserUnitProgressResponse
	err          error
}

func (f *fakeResetUserUnitProgressRecorder) Execute(ctx context.Context, request apvdto.ResetUserUnitProgressRequest) (apvdto.ResetUserUnitProgressResponse, error) {
	f.userID = request.UserID
	f.coarseUnitID = request.CoarseUnitID
	return f.response, f.err
}
