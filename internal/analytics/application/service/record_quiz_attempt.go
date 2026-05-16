package service

import (
	"context"
	"fmt"

	"learning-video-recommendation-system/internal/analytics/application/dto"
	apprepo "learning-video-recommendation-system/internal/analytics/application/repository"
	appusecase "learning-video-recommendation-system/internal/analytics/application/usecase"
	"learning-video-recommendation-system/internal/analytics/domain/model"
)

type RecordQuizAttemptUsecase struct {
	writer apprepo.RawEventWriter
}

var _ appusecase.RecordQuizAttemptUsecase = (*RecordQuizAttemptUsecase)(nil)

func NewRecordQuizAttemptUsecase(writer apprepo.RawEventWriter) *RecordQuizAttemptUsecase {
	return &RecordQuizAttemptUsecase{writer: writer}
}

func (u *RecordQuizAttemptUsecase) Execute(ctx context.Context, request dto.RecordQuizAttemptRequest) (dto.RecordQuizAttemptResponse, error) {
	if u.writer == nil {
		return dto.RecordQuizAttemptResponse{}, fmt.Errorf("raw event writer is required")
	}
	event, err := mapQuizAttemptRequest(request)
	if err != nil {
		return dto.RecordQuizAttemptResponse{}, err
	}

	result, err := u.writer.UpsertQuizEvent(ctx, event)
	if err != nil {
		return dto.RecordQuizAttemptResponse{}, err
	}
	return dto.RecordQuizAttemptResponse{
		Accepted:    true,
		QuizEventID: result.EventID,
		Inserted:    result.Inserted,
	}, nil
}

func mapQuizAttemptRequest(request dto.RecordQuizAttemptRequest) (model.RawQuizEvent, error) {
	if request.UserID == "" {
		return model.RawQuizEvent{}, fmt.Errorf("user_id is required")
	}
	if request.ClientEventID == "" {
		return model.RawQuizEvent{}, fmt.Errorf("client_event_id is required")
	}
	if request.QuestionID == "" {
		return model.RawQuizEvent{}, fmt.Errorf("question_id is required")
	}
	if request.CoarseUnitID == 0 {
		return model.RawQuizEvent{}, fmt.Errorf("coarse_unit_id is required")
	}
	if request.TriggerType == "" {
		return model.RawQuizEvent{}, fmt.Errorf("trigger_type is required")
	}
	if len(request.SelectedOptionIDs) == 0 {
		return model.RawQuizEvent{}, fmt.Errorf("selected_option_ids is required")
	}
	if len(request.SelectedOptionIDs) != len(request.SelectionIntervalMS) {
		return model.RawQuizEvent{}, fmt.Errorf("selection_interval_ms must match selected_option_ids")
	}
	if request.TotalElapsedMS < 0 {
		return model.RawQuizEvent{}, fmt.Errorf("total_elapsed_ms must be non-negative")
	}
	if request.ShownAt.IsZero() {
		return model.RawQuizEvent{}, fmt.Errorf("shown_at is required")
	}
	if request.CompletedAt.IsZero() {
		return model.RawQuizEvent{}, fmt.Errorf("completed_at is required")
	}
	if request.CompletedAt.Before(request.ShownAt) {
		return model.RawQuizEvent{}, fmt.Errorf("completed_at must be >= shown_at")
	}
	lastOption := request.SelectedOptionIDs[len(request.SelectedOptionIDs)-1]
	if lastOption != "correct" {
		return model.RawQuizEvent{}, fmt.Errorf("selected_option_ids must end with correct")
	}
	if request.IsFirstTryCorrect != (request.SelectedOptionIDs[0] == "correct") {
		return model.RawQuizEvent{}, fmt.Errorf("is_first_try_correct does not match selected_option_ids")
	}
	for index, interval := range request.SelectionIntervalMS {
		if interval < 0 {
			return model.RawQuizEvent{}, fmt.Errorf("selection_interval_ms[%d] must be non-negative", index)
		}
	}
	clientContext, err := normalizeJSONObject(request.ClientContext, "client_context")
	if err != nil {
		return model.RawQuizEvent{}, err
	}

	return model.RawQuizEvent{
		ClientEventID:       request.ClientEventID,
		UserID:              request.UserID,
		ClientContext:       clientContext,
		QuestionID:          request.QuestionID,
		CoarseUnitID:        request.CoarseUnitID,
		VideoID:             request.VideoID,
		RecommendationRunID: request.RecommendationRunID,
		TriggerType:         request.TriggerType,
		SelectedOptionIDs:   request.SelectedOptionIDs,
		SelectionIntervalMS: request.SelectionIntervalMS,
		IsFirstTryCorrect:   request.IsFirstTryCorrect,
		TotalElapsedMS:      request.TotalElapsedMS,
		ShownAt:             request.ShownAt,
		CompletedAt:         request.CompletedAt,
	}, nil
}
