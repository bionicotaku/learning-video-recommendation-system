//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	catalogrepo "learning-video-recommendation-system/internal/catalog/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/catalog/test/fixture"
)

func TestEndQuizQuestionReaderSelectsVisibleVideoAndQuestionCandidates(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()

	videoID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	inactiveVideoID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	seedFeedVideo(t, db.Pool, videoID, "Visible", "", "hls/visible/master.m3u8", "", "active", "public", nil)
	seedFeedVideo(t, db.Pool, inactiveVideoID, "Inactive", "", "hls/inactive/master.m3u8", "", "inactive", "public", nil)
	seedQuizUnit(t, db, 101, "alpha")
	seedQuizUnit(t, db, 102, "beta")
	seedQuizQuestion(t, db, "11111111-1111-1111-1111-111111111111", "video_unit", "context_meaning_choice", 101, videoID, "older", "active", time.Now().UTC().Add(-time.Hour))
	seedQuizQuestion(t, db, "22222222-2222-2222-2222-222222222222", "video_unit", "context_meaning_choice", 101, videoID, "newer", "active", time.Now().UTC())
	seedQuizQuestion(t, db, "55555555-5555-5555-5555-555555555555", "video_unit", "reverse_identification_choice", 101, videoID, "wrong video type", "active", time.Now().UTC().Add(time.Hour))
	seedQuizQuestion(t, db, "33333333-3333-3333-3333-333333333333", "unit", "unit_meaning_choice", 102, "", "generic", "active", time.Now().UTC())
	seedQuizQuestion(t, db, "44444444-4444-4444-4444-444444444444", "unit", "unit_meaning_choice", 102, "", "retired", "retired", time.Now().UTC().Add(time.Hour))
	seedQuizQuestion(t, db, "66666666-6666-6666-6666-666666666666", "unit", "context_cloze_choice", 102, "", "wrong unit type", "active", time.Now().UTC().Add(time.Hour))

	reader := catalogrepo.NewEndQuizQuestionReader(db.Pool)
	visible, err := reader.HasVisibleVideoForEndQuiz(ctx, videoID)
	if err != nil {
		t.Fatalf("has visible video: %v", err)
	}
	if !visible {
		t.Fatal("expected visible video")
	}
	visible, err = reader.HasVisibleVideoForEndQuiz(ctx, inactiveVideoID)
	if err != nil {
		t.Fatalf("has inactive video: %v", err)
	}
	if visible {
		t.Fatal("inactive video should not be visible")
	}

	videoQuestions, err := reader.ListVideoUnitQuizQuestionCandidates(ctx, videoID, []int64{101, 102})
	if err != nil {
		t.Fatalf("list video questions: %v", err)
	}
	if len(videoQuestions) != 2 {
		t.Fatalf("expected 2 video questions, got %d: %+v", len(videoQuestions), videoQuestions)
	}
	if videoQuestions[0].QuestionID != "22222222-2222-2222-2222-222222222222" || videoQuestions[1].QuestionID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("video questions should be ordered newest first per unit: %+v", videoQuestions)
	}
	if videoQuestions[0].ContextSentenceIndex == nil || *videoQuestions[0].ContextSentenceIndex != 1 {
		t.Fatalf("context fields not mapped: %+v", videoQuestions[0])
	}

	unitQuestions, err := reader.ListUnitQuizQuestionCandidates(ctx, []int64{102})
	if err != nil {
		t.Fatalf("list unit questions: %v", err)
	}
	if len(unitQuestions) != 1 || unitQuestions[0].QuestionID != "33333333-3333-3333-3333-333333333333" {
		t.Fatalf("expected active generic question only, got %+v", unitQuestions)
	}
}

func seedQuizUnit(t *testing.T, db *fixture.TestDatabase, id int64, label string) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		insert into semantic.coarse_unit (id, label, status)
		values ($1, $2, 'active')
		on conflict (id) do update set label = excluded.label, status = 'active'`, id, label); err != nil {
		t.Fatalf("seed quiz unit: %v", err)
	}
}

func seedQuizQuestion(t *testing.T, db *fixture.TestDatabase, questionID string, scopeType string, questionType string, coarseUnitID int64, videoID string, question string, status string, createdAt time.Time) {
	t.Helper()
	var videoValue any
	if videoID != "" {
		videoValue = videoID
	}
	if _, err := db.Pool.Exec(context.Background(), `
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
			created_at,
			updated_at
		) values (
			$1::uuid,
			$2,
			$3,
			$4::bigint,
			'target-' || $4::text,
			$5::uuid,
			case when $2 = 'video_unit' then 1 else null end,
			case when $2 = 'video_unit' then 2 else null end,
			case when $2 = 'video_unit' then 1000 else null end,
			case when $2 = 'video_unit' then 1800 else null end,
			jsonb_build_object(
				'question', $6::text,
				'context_text', 'context',
				'options', jsonb_build_array(
					jsonb_build_object('id', 'correct', 'text', 'right'),
					jsonb_build_object('id', 'wrong_1', 'text', 'wrong')
				),
				'explanation', 'because'
			),
			$7,
			$8,
			$8
		)`, questionID, scopeType, questionType, coarseUnitID, videoValue, question, status, createdAt); err != nil {
		t.Fatalf("seed quiz question: %v", err)
	}
}
