package service

import (
	"context"
	"fmt"
	"sort"

	learningdto "learning-video-recommendation-system/internal/learningengine/application/dto"
	learningusecase "learning-video-recommendation-system/internal/learningengine/application/usecase"
	"learning-video-recommendation-system/internal/learningengine/normalizer/application/dto"
	normalizerrepo "learning-video-recommendation-system/internal/learningengine/normalizer/application/repository"
	normalizerusecase "learning-video-recommendation-system/internal/learningengine/normalizer/application/usecase"
	"learning-video-recommendation-system/internal/learningengine/normalizer/domain/model"
	"learning-video-recommendation-system/internal/learningengine/normalizer/domain/rule"
)

type NormalizePendingEventsUsecase struct {
	quizReader            normalizerrepo.RawQuizEventReader
	interactionReader     normalizerrepo.RawLearningInteractionReader
	learningEventRecorder learningusecase.RecordLearningEventsUsecase
}

var _ normalizerusecase.NormalizePendingEventsUsecase = (*NormalizePendingEventsUsecase)(nil)

func NewNormalizePendingEventsUsecase(
	quizReader normalizerrepo.RawQuizEventReader,
	interactionReader normalizerrepo.RawLearningInteractionReader,
	learningEventRecorder learningusecase.RecordLearningEventsUsecase,
) *NormalizePendingEventsUsecase {
	return &NormalizePendingEventsUsecase{
		quizReader:            quizReader,
		interactionReader:     interactionReader,
		learningEventRecorder: learningEventRecorder,
	}
}

func (u *NormalizePendingEventsUsecase) Execute(ctx context.Context, request dto.NormalizePendingEventsRequest) (dto.NormalizePendingEventsResponse, error) {
	if u.learningEventRecorder == nil {
		return dto.NormalizePendingEventsResponse{}, fmt.Errorf("learning event recorder is required")
	}

	sourceKind, err := normalizeSourceKind(request.SourceKind)
	if err != nil {
		return dto.NormalizePendingEventsResponse{}, err
	}
	if (sourceKind == dto.SourceKindAll || sourceKind == dto.SourceKindQuiz) && u.quizReader == nil {
		return dto.NormalizePendingEventsResponse{}, fmt.Errorf("raw quiz event reader is required")
	}
	if (sourceKind == dto.SourceKindAll || sourceKind == dto.SourceKindLearningInteraction) && u.interactionReader == nil {
		return dto.NormalizePendingEventsResponse{}, fmt.Errorf("raw learning interaction reader is required")
	}
	limit := normalizeLimit(request.Limit)
	filter := normalizerrepo.PendingRawEventFilter{
		UserID:         request.UserID,
		Limit:          limit,
		OccurredBefore: request.OccurredBefore,
	}

	response := dto.NormalizePendingEventsResponse{}
	normalizedEvents := make([]model.NormalizedLearningEvent, 0)

	if sourceKind == dto.SourceKindAll || sourceKind == dto.SourceKindQuiz {
		quizEvents, err := u.quizReader.ListPendingQuizEvents(ctx, filter)
		if err != nil {
			response.ErrorCount++
			return response, err
		}
		response.ReadRawCount += len(quizEvents)
		for _, raw := range quizEvents {
			result, err := rule.MapQuizEvent(raw)
			if err != nil {
				response.ErrorCount++
				return response, err
			}
			collectResult(&response, &normalizedEvents, result)
		}
	}

	if sourceKind == dto.SourceKindAll || sourceKind == dto.SourceKindLearningInteraction {
		interactions, err := u.interactionReader.ListPendingLearningInteractions(ctx, filter)
		if err != nil {
			response.ErrorCount++
			return response, err
		}
		response.ReadRawCount += len(interactions)
		for _, raw := range interactions {
			result, err := rule.MapLearningInteraction(raw)
			if err != nil {
				response.ErrorCount++
				return response, err
			}
			collectResult(&response, &normalizedEvents, result)
		}
	}

	if len(normalizedEvents) == 0 {
		return response, nil
	}

	groups := groupByUser(normalizedEvents)
	userIDs := make([]string, 0, len(groups))
	for userID := range groups {
		userIDs = append(userIDs, userID)
	}
	sort.Strings(userIDs)

	for _, userID := range userIDs {
		recordResponse, err := u.learningEventRecorder.Execute(ctx, learningdto.RecordLearningEventsRequest{
			UserID: userID,
			Events: toLearningInputs(groups[userID]),
		})
		if err != nil {
			response.ErrorCount++
			return response, err
		}
		response.RecordedEventCount += recordResponse.RecordedCount
		response.RecordedUserBatchCount++
	}

	return response, nil
}

func normalizeSourceKind(value string) (string, error) {
	if value == "" {
		return dto.SourceKindAll, nil
	}
	switch value {
	case dto.SourceKindAll, dto.SourceKindQuiz, dto.SourceKindLearningInteraction:
		return value, nil
	default:
		return "", fmt.Errorf("unsupported source_kind: %s", value)
	}
}

func normalizeLimit(value int) int {
	if value <= 0 {
		return dto.DefaultNormalizeLimit
	}
	if value > dto.MaxNormalizeLimit {
		return dto.MaxNormalizeLimit
	}
	return value
}

func collectResult(response *dto.NormalizePendingEventsResponse, events *[]model.NormalizedLearningEvent, result model.NormalizationResult) {
	if result.Skipped {
		response.SkippedCount++
		return
	}
	if result.Event == nil {
		response.SkippedCount++
		return
	}
	response.NormalizedEventCount++
	*events = append(*events, *result.Event)
}

func groupByUser(events []model.NormalizedLearningEvent) map[string][]model.NormalizedLearningEvent {
	groups := make(map[string][]model.NormalizedLearningEvent)
	for _, event := range events {
		groups[event.UserID] = append(groups[event.UserID], event)
	}
	return groups
}

func toLearningInputs(events []model.NormalizedLearningEvent) []learningdto.LearningEventInput {
	inputs := make([]learningdto.LearningEventInput, 0, len(events))
	for _, event := range events {
		inputs = append(inputs, learningdto.LearningEventInput{
			CoarseUnitID:    event.CoarseUnitID,
			VideoID:         event.VideoID,
			EventType:       event.EventType,
			ReducerEffect:   event.ReducerEffect,
			SourceType:      event.SourceType,
			SourceRefID:     event.SourceRefID,
			IsCorrect:       event.IsCorrect,
			ProgressQuality: event.ProgressQuality,
			Metadata:        event.Metadata,
			OccurredAt:      event.OccurredAt,
		})
	}
	return inputs
}
