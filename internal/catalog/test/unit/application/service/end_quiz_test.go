package service_test

import (
	"context"
	"errors"
	"testing"

	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
	catalogservice "learning-video-recommendation-system/internal/catalog/application/service"
	"learning-video-recommendation-system/internal/catalog/domain/model"
)

func TestEndQuizQuestionLookupPrefersVideoContextAndFallsBackToGeneric(t *testing.T) {
	reader := &fakeEndQuizQuestionReader{
		videoVisible: true,
		videoQuestions: []model.EndQuizQuestionCandidate{
			candidate("11111111-1111-1111-1111-111111111111", 101, "video_unit", "context_meaning_choice", "target 101", validPayload("Video question 101")),
			candidate("22222222-2222-2222-2222-222222222222", 102, "video_unit", "context_meaning_choice", "target 102", []byte(`{"question":"","options":[{"id":"correct","text":"ok"}]}`)),
		},
		unitQuestions: []model.EndQuizQuestionCandidate{
			candidate("33333333-3333-3333-3333-333333333333", 102, "unit", "unit_meaning_choice", "target 102", validPayload("Unit question 102")),
			candidate("44444444-4444-4444-4444-444444444444", 103, "unit", "unit_meaning_choice", "target 103", validPayload("Unit question 103")),
		},
	}
	usecase := catalogservice.NewEndQuizQuestionLookupUsecase(reader)

	response, err := usecase.Execute(context.Background(), catalogdto.EndQuizQuestionLookupRequest{
		VideoID:       "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		CoarseUnitIDs: []int64{101, 102, 101, 999},
	})
	if err != nil {
		t.Fatalf("lookup end quiz questions: %v", err)
	}

	if response.VideoID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("video_id = %q", response.VideoID)
	}
	if len(response.Items) != 2 {
		t.Fatalf("expected 2 items, got %d: %+v", len(response.Items), response.Items)
	}
	if response.Items[0].CoarseUnitID != 101 || response.Items[0].QuestionID != "11111111-1111-1111-1111-111111111111" || response.Items[0].Source != "video_context" {
		t.Fatalf("unit 101 should use video question: %+v", response.Items[0])
	}
	if response.Items[0].Question != "Video question 101" || response.Items[0].Options[0].OptionID != "correct" || response.Items[0].Explanation == nil || *response.Items[0].Explanation != "正确解释" {
		t.Fatalf("unit 101 payload not mapped: %+v", response.Items[0])
	}
	if response.Items[0].ContextSentenceIndex == nil || *response.Items[0].ContextSentenceIndex != 7 {
		t.Fatalf("context indexes not mapped: %+v", response.Items[0])
	}
	if response.Items[1].CoarseUnitID != 102 || response.Items[1].QuestionID != "33333333-3333-3333-3333-333333333333" || response.Items[1].Source != "unit_generic" {
		t.Fatalf("unit 102 should fall back to generic question after invalid video candidate: %+v", response.Items[1])
	}
	if len(response.MissingCoarseUnitIDs) != 1 || response.MissingCoarseUnitIDs[0] != 999 {
		t.Fatalf("unexpected missing units: %+v", response.MissingCoarseUnitIDs)
	}
	if len(reader.requestedUnitIDs) != 3 || reader.requestedUnitIDs[0] != 101 || reader.requestedUnitIDs[1] != 102 || reader.requestedUnitIDs[2] != 999 {
		t.Fatalf("unit ids should be deduped preserving order: %+v", reader.requestedUnitIDs)
	}
}

func TestEndQuizQuestionLookupRejectsInvalidRequestAndMissingVideo(t *testing.T) {
	cases := []struct {
		name    string
		request catalogdto.EndQuizQuestionLookupRequest
		reader  *fakeEndQuizQuestionReader
		wantErr func(error) bool
	}{
		{
			name:    "missing video id",
			request: catalogdto.EndQuizQuestionLookupRequest{CoarseUnitIDs: []int64{101}},
			reader:  &fakeEndQuizQuestionReader{videoVisible: true},
			wantErr: catalogservice.IsValidationError,
		},
		{
			name:    "empty unit ids",
			request: catalogdto.EndQuizQuestionLookupRequest{VideoID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"},
			reader:  &fakeEndQuizQuestionReader{videoVisible: true},
			wantErr: catalogservice.IsValidationError,
		},
		{
			name:    "too many unit ids",
			request: catalogdto.EndQuizQuestionLookupRequest{VideoID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", CoarseUnitIDs: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9}},
			reader:  &fakeEndQuizQuestionReader{videoVisible: true},
			wantErr: catalogservice.IsValidationError,
		},
		{
			name:    "non-positive unit id",
			request: catalogdto.EndQuizQuestionLookupRequest{VideoID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", CoarseUnitIDs: []int64{0}},
			reader:  &fakeEndQuizQuestionReader{videoVisible: true},
			wantErr: catalogservice.IsValidationError,
		},
		{
			name:    "video not visible",
			request: catalogdto.EndQuizQuestionLookupRequest{VideoID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", CoarseUnitIDs: []int64{101}},
			reader:  &fakeEndQuizQuestionReader{videoVisible: false},
			wantErr: catalogservice.IsNotFoundError,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			usecase := catalogservice.NewEndQuizQuestionLookupUsecase(tt.reader)
			_, err := usecase.Execute(context.Background(), tt.request)
			if err == nil || !tt.wantErr(err) {
				t.Fatalf("expected classified error, got %v", err)
			}
		})
	}
}

func TestEndQuizQuestionLookupPropagatesReaderError(t *testing.T) {
	reader := &fakeEndQuizQuestionReader{err: errors.New("db down")}
	usecase := catalogservice.NewEndQuizQuestionLookupUsecase(reader)

	_, err := usecase.Execute(context.Background(), catalogdto.EndQuizQuestionLookupRequest{
		VideoID:       "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		CoarseUnitIDs: []int64{101},
	})
	if !errors.Is(err, reader.err) {
		t.Fatalf("expected reader error, got %v", err)
	}
}

type fakeEndQuizQuestionReader struct {
	videoVisible     bool
	err              error
	requestedUnitIDs []int64
	videoQuestions   []model.EndQuizQuestionCandidate
	unitQuestions    []model.EndQuizQuestionCandidate
}

func (f *fakeEndQuizQuestionReader) HasVisibleVideoForEndQuiz(ctx context.Context, videoID string) (bool, error) {
	return f.videoVisible, f.err
}

func (f *fakeEndQuizQuestionReader) ListVideoUnitQuizQuestionCandidates(ctx context.Context, videoID string, coarseUnitIDs []int64) ([]model.EndQuizQuestionCandidate, error) {
	f.requestedUnitIDs = append([]int64(nil), coarseUnitIDs...)
	return f.videoQuestions, f.err
}

func (f *fakeEndQuizQuestionReader) ListUnitQuizQuestionCandidates(ctx context.Context, coarseUnitIDs []int64) ([]model.EndQuizQuestionCandidate, error) {
	return f.unitQuestions, f.err
}

func candidate(questionID string, coarseUnitID int64, scopeType string, questionType string, targetText string, payload []byte) model.EndQuizQuestionCandidate {
	contextSentenceIndex := int32(7)
	contextSpanIndex := int32(2)
	contextStartMS := int32(1200)
	contextEndMS := int32(1800)
	return model.EndQuizQuestionCandidate{
		QuestionID:           questionID,
		ScopeType:            scopeType,
		QuestionType:         questionType,
		CoarseUnitID:         coarseUnitID,
		TargetText:           targetText,
		ContextSentenceIndex: &contextSentenceIndex,
		ContextSpanIndex:     &contextSpanIndex,
		ContextStartMS:       &contextStartMS,
		ContextEndMS:         &contextEndMS,
		ContentPayload:       payload,
	}
}

func validPayload(question string) []byte {
	return []byte(`{
		"question": "` + question + `",
		"context_text": "上下文句子",
		"options": [
			{"id": "correct", "text": "正确选项"},
			{"id": "wrong_1", "text": "错误选项"}
		],
		"explanation": "正确解释"
	}`)
}
