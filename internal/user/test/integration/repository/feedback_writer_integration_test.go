//go:build integration

package repository_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"learning-video-recommendation-system/internal/user/domain/model"
	userrepo "learning-video-recommendation-system/internal/user/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/user/test/fixture"
)

var suite *fixture.Suite

func TestMain(m *testing.M) {
	var err error
	suite, err = fixture.OpenSuite()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "open user integration suite: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	if err := suite.Close(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "close user integration suite: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

func TestFeedbackWriterStoresSubmissionImagesAndDeduplicatesClientFeedbackID(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	userID := "11111111-1111-4111-8111-111111111111"
	db.SeedAuthUser(t, userID, "alice@example.com")
	clientID := "22222222-2222-4222-8222-222222222222"
	writer := userrepo.NewFeedbackWriter(db.Pool)

	first, err := writer.SubmitFeedback(context.Background(), model.FeedbackSubmission{
		UserID:           userID,
		ClientFeedbackID: &clientID,
		Payload:          json.RawMessage(`{"message":"bug"}`),
		Images: []model.FeedbackImage{{
			SortOrder:   1,
			ContentType: "image/jpeg",
			SizeBytes:   3,
			SHA256:      "sha",
			Width:       1,
			Height:      1,
			Data:        []byte{0xff, 0xd8, 0xff},
		}},
	})
	if err != nil {
		t.Fatalf("SubmitFeedback first: %v", err)
	}
	second, err := writer.SubmitFeedback(context.Background(), model.FeedbackSubmission{
		UserID:           userID,
		ClientFeedbackID: &clientID,
		Payload:          json.RawMessage(`{"message":"ignored"}`),
		Images: []model.FeedbackImage{{
			SortOrder:   1,
			ContentType: "image/jpeg",
			SizeBytes:   3,
			SHA256:      "other",
			Width:       1,
			Height:      1,
			Data:        []byte{0xff, 0xd8, 0xff},
		}},
	})
	if err != nil {
		t.Fatalf("SubmitFeedback duplicate: %v", err)
	}
	if first.FeedbackID != second.FeedbackID || first.ImageCount != 1 || second.ImageCount != 1 {
		t.Fatalf("unexpected idempotent results: first=%+v second=%+v", first, second)
	}

	var submissionCount int
	var imageCount int
	var payload string
	var imageData []byte
	if err := db.Pool.QueryRow(context.Background(), `select count(*) from app_user.feedback_submissions where user_id = $1`, userID).Scan(&submissionCount); err != nil {
		t.Fatalf("count submissions: %v", err)
	}
	if err := db.Pool.QueryRow(context.Background(), `select count(*) from app_user.feedback_images`).Scan(&imageCount); err != nil {
		t.Fatalf("count images: %v", err)
	}
	if err := db.Pool.QueryRow(context.Background(), `select payload::text from app_user.feedback_submissions where id = $1`, first.FeedbackID).Scan(&payload); err != nil {
		t.Fatalf("read payload: %v", err)
	}
	if err := db.Pool.QueryRow(context.Background(), `select image_data from app_user.feedback_images where submission_id = $1 and sort_order = 1`, first.FeedbackID).Scan(&imageData); err != nil {
		t.Fatalf("read image data: %v", err)
	}
	if submissionCount != 1 || imageCount != 1 || payload != `{"message": "bug"}` || len(imageData) != 3 {
		t.Fatalf("unexpected persisted data submissions=%d images=%d payload=%s image_len=%d", submissionCount, imageCount, payload, len(imageData))
	}
}
