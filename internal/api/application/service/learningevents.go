package service

import (
	"context"
	"errors"
	"log/slog"

	analyticsdto "learning-video-recommendation-system/internal/analytics/application/dto"
	analyticsservice "learning-video-recommendation-system/internal/analytics/application/service"
	analyticsusecase "learning-video-recommendation-system/internal/analytics/application/usecase"
	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	normalizerdto "learning-video-recommendation-system/internal/learningengine/normalizer/application/dto"
	normalizerusecase "learning-video-recommendation-system/internal/learningengine/normalizer/application/usecase"

	"github.com/jackc/pgx/v5/pgconn"
)

type RecordLearningInteractionsBatchService struct {
	rawWriter  analyticsusecase.RecordLearningInteractionsBatchUsecase
	normalizer normalizerusecase.NormalizeLearningInteractionsByIDsUsecase
	logger     *slog.Logger
}

func NewRecordLearningInteractionsBatchService(
	rawWriter analyticsusecase.RecordLearningInteractionsBatchUsecase,
	normalizer normalizerusecase.NormalizeLearningInteractionsByIDsUsecase,
	logger *slog.Logger,
) *RecordLearningInteractionsBatchService {
	return &RecordLearningInteractionsBatchService{
		rawWriter:  rawWriter,
		normalizer: normalizer,
		logger:     logger,
	}
}

func (s *RecordLearningInteractionsBatchService) Execute(ctx context.Context, request apvdto.RecordLearningInteractionsBatchRequest) (apvdto.RecordLearningInteractionsBatchResponse, error) {
	if s.rawWriter == nil {
		return apvdto.RecordLearningInteractionsBatchResponse{}, errors.New("learning interaction raw writer is required")
	}

	rawResponse, err := s.rawWriter.Execute(ctx, toAnalyticsLearningInteractionsRequest(request))
	if err != nil {
		return apvdto.RecordLearningInteractionsBatchResponse{}, classifyOwnerError(err)
	}

	response := apvdto.RecordLearningInteractionsBatchResponse{
		AcceptedCount:  rawResponse.AcceptedCount,
		InsertedCount:  rawResponse.InsertedCount,
		DuplicateCount: rawResponse.DuplicateCount,
		Events:         make([]apvdto.AcceptedLearningInteractionEvent, 0, len(rawResponse.AcceptedEvents)),
	}
	ids := make([]string, 0, len(rawResponse.AcceptedEvents))
	for _, event := range rawResponse.AcceptedEvents {
		response.Events = append(response.Events, apvdto.AcceptedLearningInteractionEvent{
			ClientEventID:              event.ClientEventID,
			LearningInteractionEventID: event.LearningInteractionEventID,
			Inserted:                   event.Inserted,
		})
		ids = append(ids, event.LearningInteractionEventID)
	}

	if s.normalizer != nil && len(ids) > 0 {
		if _, err := s.normalizer.Execute(ctx, normalizerdto.NormalizeLearningInteractionsByIDsRequest{
			UserID:                      request.UserID,
			LearningInteractionEventIDs: ids,
		}); err != nil {
			logger := s.logger
			if logger == nil {
				logger = slog.Default()
			}
			logger.WarnContext(ctx, "normalize learning interactions failed", "error", err)
		}
	}

	return response, nil
}

type RecordQuizAttemptService struct {
	rawWriter  analyticsusecase.RecordQuizAttemptUsecase
	normalizer normalizerusecase.NormalizeQuizAttemptByIDUsecase
	logger     *slog.Logger
}

func NewRecordQuizAttemptService(
	rawWriter analyticsusecase.RecordQuizAttemptUsecase,
	normalizer normalizerusecase.NormalizeQuizAttemptByIDUsecase,
	logger *slog.Logger,
) *RecordQuizAttemptService {
	return &RecordQuizAttemptService{
		rawWriter:  rawWriter,
		normalizer: normalizer,
		logger:     logger,
	}
}

func (s *RecordQuizAttemptService) Execute(ctx context.Context, request apvdto.RecordQuizAttemptRequest) (apvdto.RecordQuizAttemptResponse, error) {
	if s.rawWriter == nil {
		return apvdto.RecordQuizAttemptResponse{}, errors.New("quiz attempt raw writer is required")
	}

	rawResponse, err := s.rawWriter.Execute(ctx, analyticsdto.RecordQuizAttemptRequest{
		UserID:              request.UserID,
		ClientContext:       request.ClientContext,
		ClientEventID:       request.ClientEventID,
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
	})
	if err != nil {
		return apvdto.RecordQuizAttemptResponse{}, classifyOwnerError(err)
	}

	if s.normalizer != nil && rawResponse.QuizEventID != "" {
		if _, err := s.normalizer.Execute(ctx, normalizerdto.NormalizeQuizAttemptByIDRequest{
			UserID:      request.UserID,
			QuizEventID: rawResponse.QuizEventID,
		}); err != nil {
			logger := s.logger
			if logger == nil {
				logger = slog.Default()
			}
			logger.WarnContext(ctx, "normalize quiz attempt failed", "error", err)
		}
	}

	return apvdto.RecordQuizAttemptResponse{
		Accepted:    rawResponse.Accepted,
		QuizEventID: rawResponse.QuizEventID,
		Inserted:    rawResponse.Inserted,
	}, nil
}

type RecordSelfMarkMasteredService struct {
	rawWriter  analyticsusecase.RecordSelfMarkMasteredUsecase
	normalizer normalizerusecase.NormalizeSelfMarkMasteredByIDUsecase
	logger     *slog.Logger
}

func NewRecordSelfMarkMasteredService(
	rawWriter analyticsusecase.RecordSelfMarkMasteredUsecase,
	normalizer normalizerusecase.NormalizeSelfMarkMasteredByIDUsecase,
	logger *slog.Logger,
) *RecordSelfMarkMasteredService {
	return &RecordSelfMarkMasteredService{
		rawWriter:  rawWriter,
		normalizer: normalizer,
		logger:     logger,
	}
}

func (s *RecordSelfMarkMasteredService) Execute(ctx context.Context, request apvdto.RecordSelfMarkMasteredRequest) (apvdto.RecordSelfMarkMasteredResponse, error) {
	if s.rawWriter == nil {
		return apvdto.RecordSelfMarkMasteredResponse{}, errors.New("self mark raw writer is required")
	}

	rawResponse, err := s.rawWriter.Execute(ctx, analyticsdto.RecordSelfMarkMasteredRequest{
		UserID:              request.UserID,
		ClientContext:       request.ClientContext,
		ClientEventID:       request.ClientEventID,
		CoarseUnitID:        request.CoarseUnitID,
		SourceSurface:       request.SourceSurface,
		VideoID:             request.VideoID,
		WatchSessionID:      request.WatchSessionID,
		RecommendationRunID: request.RecommendationRunID,
		RelatedQuizEventID:  request.RelatedQuizEventID,
		TokenText:           request.TokenText,
		SentenceIndex:       request.SentenceIndex,
		SpanIndex:           request.SpanIndex,
		OccurredAt:          request.OccurredAt,
		EventPayload:        request.EventPayload,
	})
	if err != nil {
		return apvdto.RecordSelfMarkMasteredResponse{}, classifyOwnerError(err)
	}

	if s.normalizer != nil && rawResponse.LearningInteractionEventID != "" {
		if _, err := s.normalizer.Execute(ctx, normalizerdto.NormalizeSelfMarkMasteredByIDRequest{
			UserID:                     request.UserID,
			LearningInteractionEventID: rawResponse.LearningInteractionEventID,
		}); err != nil {
			logger := s.logger
			if logger == nil {
				logger = slog.Default()
			}
			logger.WarnContext(ctx, "normalize self mark mastered failed", "error", err)
		}
	}

	return apvdto.RecordSelfMarkMasteredResponse{
		Accepted:                   rawResponse.Accepted,
		LearningInteractionEventID: rawResponse.LearningInteractionEventID,
		Inserted:                   rawResponse.Inserted,
	}, nil
}

func toAnalyticsLearningInteractionsRequest(request apvdto.RecordLearningInteractionsBatchRequest) analyticsdto.RecordLearningInteractionsBatchRequest {
	events := make([]analyticsdto.LearningInteractionEventInput, 0, len(request.Events))
	for _, event := range request.Events {
		events = append(events, analyticsdto.LearningInteractionEventInput{
			ClientEventID:                  event.ClientEventID,
			EventType:                      event.EventType,
			SourceSurface:                  event.SourceSurface,
			CoarseUnitID:                   event.CoarseUnitID,
			TokenText:                      event.TokenText,
			SentenceIndex:                  event.SentenceIndex,
			SpanIndex:                      event.SpanIndex,
			OccurredAt:                     event.OccurredAt,
			ExposureStartMS:                event.ExposureStartMS,
			ExposureEndMS:                  event.ExposureEndMS,
			ExposureCount:                  event.ExposureCount,
			LookupVisibleMS:                event.LookupVisibleMS,
			LookupSentenceAudioReplayCount: event.LookupSentenceAudioReplayCount,
			LookupWordAudioPlayCount:       event.LookupWordAudioPlayCount,
			LookupPracticeNowClicked:       event.LookupPracticeNowClicked,
			EventPayload:                   event.EventPayload,
		})
	}

	return analyticsdto.RecordLearningInteractionsBatchRequest{
		UserID:              request.UserID,
		ClientContext:       request.ClientContext,
		VideoID:             request.VideoID,
		WatchSessionID:      request.WatchSessionID,
		RecommendationRunID: request.RecommendationRunID,
		Events:              events,
	}
}

func classifyOwnerError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return ServiceUnavailableError("request canceled or timed out")
	}
	var appErr *Error
	if errors.As(err, &appErr) {
		return err
	}
	if analyticsservice.IsValidationError(err) {
		return InvalidRequestError(err.Error())
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return err
	}
	return err
}
