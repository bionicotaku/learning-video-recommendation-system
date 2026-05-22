package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	userdto "learning-video-recommendation-system/internal/user/application/dto"
	userservice "learning-video-recommendation-system/internal/user/application/service"
	"learning-video-recommendation-system/internal/user/domain/model"
)

func TestSubmitFeedbackRejectsNonObjectPayload(t *testing.T) {
	writer := &fakeFeedbackWriter{}
	usecase := userservice.NewSubmitFeedbackUsecase(writer)

	_, err := usecase.Execute(context.Background(), userdto.SubmitFeedbackRequest{
		UserID:  "11111111-1111-4111-8111-111111111111",
		Payload: json.RawMessage(`[]`),
	})

	if !userservice.IsValidationError(err) {
		t.Fatalf("err = %v, want validation error", err)
	}
	if writer.called {
		t.Fatalf("writer should not be called")
	}
}

func TestSubmitFeedbackRejectsMoreThanFiveImages(t *testing.T) {
	writer := &fakeFeedbackWriter{}
	usecase := userservice.NewSubmitFeedbackUsecase(writer)

	request := userdto.SubmitFeedbackRequest{
		UserID:  "11111111-1111-4111-8111-111111111111",
		Payload: json.RawMessage(`{"message":"bug"}`),
	}
	for i := 0; i < 6; i++ {
		request.Images = append(request.Images, userdto.FeedbackImageInput{
			SortOrder:   int32(i + 1),
			ContentType: "image/jpeg",
			SizeBytes:   3,
			SHA256:      "sha",
			Width:       1,
			Height:      1,
			Data:        []byte{0xff, 0xd8, 0xff},
		})
	}

	_, err := usecase.Execute(context.Background(), request)

	if !userservice.IsValidationError(err) {
		t.Fatalf("err = %v, want validation error", err)
	}
	if writer.called {
		t.Fatalf("writer should not be called")
	}
}

func TestSubmitFeedbackPassesValidatedSubmissionToWriter(t *testing.T) {
	createdAt := time.Date(2026, 5, 22, 18, 30, 0, 0, time.UTC)
	writer := &fakeFeedbackWriter{
		result: model.FeedbackSubmissionResult{
			FeedbackID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
			ImageCount: 1,
			CreatedAt:  createdAt,
		},
	}
	usecase := userservice.NewSubmitFeedbackUsecase(writer)
	clientID := "22222222-2222-4222-8222-222222222222"

	result, err := usecase.Execute(context.Background(), userdto.SubmitFeedbackRequest{
		UserID:           "11111111-1111-4111-8111-111111111111",
		ClientFeedbackID: &clientID,
		Payload:          json.RawMessage(`{"message":"bug"}`),
		Images: []userdto.FeedbackImageInput{{
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
		t.Fatalf("Execute: %v", err)
	}
	if result.FeedbackID != "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" || !result.Accepted || result.ImageCount != 1 || result.CreatedAt != "2026-05-22T18:30:00Z" {
		t.Fatalf("unexpected response: %+v", result)
	}
	if !writer.called || writer.submission.ClientFeedbackID == nil || *writer.submission.ClientFeedbackID != clientID {
		t.Fatalf("writer submission not mapped: %+v", writer.submission)
	}
}

type fakeFeedbackWriter struct {
	called     bool
	submission model.FeedbackSubmission
	result     model.FeedbackSubmissionResult
	err        error
}

func (f *fakeFeedbackWriter) SubmitFeedback(ctx context.Context, submission model.FeedbackSubmission) (model.FeedbackSubmissionResult, error) {
	if ctx == nil {
		return model.FeedbackSubmissionResult{}, errors.New("ctx is required")
	}
	f.called = true
	f.submission = submission
	return f.result, f.err
}
