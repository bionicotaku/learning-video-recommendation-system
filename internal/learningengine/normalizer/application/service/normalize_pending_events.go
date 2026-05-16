package service

import (
	"context"
	"fmt"
	"sort"

	"learning-video-recommendation-system/internal/learningengine/normalizer/application/dto"
	normalizerrepo "learning-video-recommendation-system/internal/learningengine/normalizer/application/repository"
	normalizerusecase "learning-video-recommendation-system/internal/learningengine/normalizer/application/usecase"
	"learning-video-recommendation-system/internal/learningengine/normalizer/domain/model"
	"learning-video-recommendation-system/internal/learningengine/normalizer/domain/rule"
	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	learningenum "learning-video-recommendation-system/internal/learningengine/reducer/domain/enum"
)

type NormalizePendingEventsUsecase struct {
	quizReader            normalizerrepo.RawQuizEventReader
	interactionReader     normalizerrepo.RawLearningInteractionReader
	learningEventRecorder normalizerrepo.LearningEventRecorder
}

var _ normalizerusecase.NormalizePendingEventsUsecase = (*NormalizePendingEventsUsecase)(nil)
var _ normalizerusecase.NormalizeLearningInteractionsByIDsUsecase = (*NormalizeLearningInteractionsByIDsUsecase)(nil)
var _ normalizerusecase.NormalizeQuizAttemptByIDUsecase = (*NormalizeQuizAttemptByIDUsecase)(nil)
var _ normalizerusecase.NormalizeSelfMarkMasteredByIDUsecase = (*NormalizeSelfMarkMasteredByIDUsecase)(nil)

func NewNormalizePendingEventsUsecase(
	quizReader normalizerrepo.RawQuizEventReader,
	interactionReader normalizerrepo.RawLearningInteractionReader,
	learningEventRecorder normalizerrepo.LearningEventRecorder,
) *NormalizePendingEventsUsecase {
	return &NormalizePendingEventsUsecase{
		quizReader:            quizReader,
		interactionReader:     interactionReader,
		learningEventRecorder: learningEventRecorder,
	}
}

type NormalizeLearningInteractionsByIDsUsecase struct {
	interactionReader     normalizerrepo.RawLearningInteractionReader
	learningEventRecorder normalizerrepo.LearningEventRecorder
}

func NewNormalizeLearningInteractionsByIDsUsecase(
	interactionReader normalizerrepo.RawLearningInteractionReader,
	learningEventRecorder normalizerrepo.LearningEventRecorder,
) *NormalizeLearningInteractionsByIDsUsecase {
	return &NormalizeLearningInteractionsByIDsUsecase{
		interactionReader:     interactionReader,
		learningEventRecorder: learningEventRecorder,
	}
}

type NormalizeQuizAttemptByIDUsecase struct {
	quizReader            normalizerrepo.RawQuizEventReader
	learningEventRecorder normalizerrepo.LearningEventRecorder
}

func NewNormalizeQuizAttemptByIDUsecase(
	quizReader normalizerrepo.RawQuizEventReader,
	learningEventRecorder normalizerrepo.LearningEventRecorder,
) *NormalizeQuizAttemptByIDUsecase {
	return &NormalizeQuizAttemptByIDUsecase{
		quizReader:            quizReader,
		learningEventRecorder: learningEventRecorder,
	}
}

type NormalizeSelfMarkMasteredByIDUsecase struct {
	interactionReader     normalizerrepo.RawLearningInteractionReader
	learningEventRecorder normalizerrepo.LearningEventRecorder
}

func NewNormalizeSelfMarkMasteredByIDUsecase(
	interactionReader normalizerrepo.RawLearningInteractionReader,
	learningEventRecorder normalizerrepo.LearningEventRecorder,
) *NormalizeSelfMarkMasteredByIDUsecase {
	return &NormalizeSelfMarkMasteredByIDUsecase{
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
	quizEvents := make([]model.RawQuizEvent, 0)
	interactions := make([]model.RawLearningInteraction, 0)

	if sourceKind == dto.SourceKindAll || sourceKind == dto.SourceKindQuiz {
		readQuizEvents, err := u.quizReader.ListPendingQuizEvents(ctx, filter)
		if err != nil {
			response.ErrorCount++
			return response, err
		}
		response.ReadRawCount += len(readQuizEvents)
		quizEvents = append(quizEvents, readQuizEvents...)
	}

	if sourceKind == dto.SourceKindAll || sourceKind == dto.SourceKindLearningInteraction {
		readInteractions, err := u.interactionReader.ListPendingLearningInteractions(ctx, filter)
		if err != nil {
			response.ErrorCount++
			return response, err
		}
		response.ReadRawCount += len(readInteractions)
		interactions = append(interactions, readInteractions...)
	}

	normalizedEvents, skippedCount, err := mapRawEvents(quizEvents, interactions)
	if err != nil {
		response.ErrorCount++
		return response, err
	}
	response.SkippedCount += skippedCount
	response.NormalizedEventCount += len(normalizedEvents)

	recordResult, err := recordNormalizedEvents(ctx, u.learningEventRecorder, normalizedEvents)
	if err != nil {
		response.ErrorCount++
		return response, err
	}
	response.RecordedEventCount += recordResult.recordedCount
	response.DuplicateEventCount += recordResult.duplicateCount
	response.RecordedUserBatchCount += recordResult.userBatchCount
	return response, nil
}

func (u *NormalizeLearningInteractionsByIDsUsecase) Execute(ctx context.Context, request dto.NormalizeLearningInteractionsByIDsRequest) (dto.NormalizeLearningInteractionsByIDsResponse, error) {
	if u.learningEventRecorder == nil {
		return dto.NormalizeLearningInteractionsByIDsResponse{}, fmt.Errorf("learning event recorder is required")
	}
	if request.UserID == "" {
		return dto.NormalizeLearningInteractionsByIDsResponse{}, fmt.Errorf("user_id is required")
	}
	if len(request.LearningInteractionEventIDs) == 0 {
		return dto.NormalizeLearningInteractionsByIDsResponse{}, fmt.Errorf("learning interaction event ids are required")
	}
	if u.interactionReader == nil {
		return dto.NormalizeLearningInteractionsByIDsResponse{}, fmt.Errorf("raw learning interaction reader is required")
	}

	response := dto.NormalizeLearningInteractionsByIDsResponse{}
	interactions := make([]model.RawLearningInteraction, 0)
	readInteractions, err := u.interactionReader.ListLearningInteractionsByIDs(ctx, request.UserID, request.LearningInteractionEventIDs)
	if err != nil {
		response.ErrorCount++
		return response, err
	}
	response.ReadRawCount += len(readInteractions)
	interactions = append(interactions, readInteractions...)
	for _, raw := range interactions {
		if raw.EventType != learningenum.EventExposure && raw.EventType != learningenum.EventLookup {
			response.ErrorCount++
			return response, fmt.Errorf("learning interaction event %s is %s, want exposure or lookup", raw.EventID, raw.EventType)
		}
	}

	normalizedEvents, skippedCount, err := mapRawEvents(nil, interactions)
	if err != nil {
		response.ErrorCount++
		return response, err
	}
	response.SkippedCount += skippedCount
	response.NormalizedEventCount += len(normalizedEvents)

	recordResult, err := recordNormalizedEvents(ctx, u.learningEventRecorder, normalizedEvents)
	if err != nil {
		response.ErrorCount++
		return response, err
	}
	response.RecordedEventCount += recordResult.recordedCount
	response.DuplicateEventCount += recordResult.duplicateCount
	response.RecordedUserBatchCount += recordResult.userBatchCount
	return response, nil
}

func (u *NormalizeQuizAttemptByIDUsecase) Execute(ctx context.Context, request dto.NormalizeQuizAttemptByIDRequest) (dto.NormalizeQuizAttemptByIDResponse, error) {
	if u.learningEventRecorder == nil {
		return dto.NormalizeQuizAttemptByIDResponse{}, fmt.Errorf("learning event recorder is required")
	}
	if request.UserID == "" {
		return dto.NormalizeQuizAttemptByIDResponse{}, fmt.Errorf("user_id is required")
	}
	if request.QuizEventID == "" {
		return dto.NormalizeQuizAttemptByIDResponse{}, fmt.Errorf("quiz_event_id is required")
	}
	if u.quizReader == nil {
		return dto.NormalizeQuizAttemptByIDResponse{}, fmt.Errorf("raw quiz event reader is required")
	}

	response := dto.NormalizeQuizAttemptByIDResponse{}
	readQuizEvents, err := u.quizReader.ListQuizEventsByIDs(ctx, request.UserID, []string{request.QuizEventID})
	if err != nil {
		response.ErrorCount++
		return response, err
	}
	response.ReadRawCount += len(readQuizEvents)

	normalizedEvents, skippedCount, err := mapRawEvents(readQuizEvents, nil)
	if err != nil {
		response.ErrorCount++
		return response, err
	}
	response.SkippedCount += skippedCount
	response.NormalizedEventCount += len(normalizedEvents)

	recordResult, err := recordNormalizedEvents(ctx, u.learningEventRecorder, normalizedEvents)
	if err != nil {
		response.ErrorCount++
		return response, err
	}
	response.RecordedEventCount += recordResult.recordedCount
	response.DuplicateEventCount += recordResult.duplicateCount
	response.RecordedUserBatchCount += recordResult.userBatchCount
	return response, nil
}

func (u *NormalizeSelfMarkMasteredByIDUsecase) Execute(ctx context.Context, request dto.NormalizeSelfMarkMasteredByIDRequest) (dto.NormalizeSelfMarkMasteredByIDResponse, error) {
	if u.learningEventRecorder == nil {
		return dto.NormalizeSelfMarkMasteredByIDResponse{}, fmt.Errorf("learning event recorder is required")
	}
	if request.UserID == "" {
		return dto.NormalizeSelfMarkMasteredByIDResponse{}, fmt.Errorf("user_id is required")
	}
	if request.LearningInteractionEventID == "" {
		return dto.NormalizeSelfMarkMasteredByIDResponse{}, fmt.Errorf("learning_interaction_event_id is required")
	}
	if u.interactionReader == nil {
		return dto.NormalizeSelfMarkMasteredByIDResponse{}, fmt.Errorf("raw learning interaction reader is required")
	}

	response := dto.NormalizeSelfMarkMasteredByIDResponse{}
	readInteractions, err := u.interactionReader.ListLearningInteractionsByIDs(ctx, request.UserID, []string{request.LearningInteractionEventID})
	if err != nil {
		response.ErrorCount++
		return response, err
	}
	response.ReadRawCount += len(readInteractions)
	if len(readInteractions) == 0 {
		return response, nil
	}
	if readInteractions[0].EventType != learningenum.EventSelfMarkMastered {
		response.ErrorCount++
		return response, fmt.Errorf("learning interaction event %s is %s, want self_mark_mastered", request.LearningInteractionEventID, readInteractions[0].EventType)
	}

	normalizedEvents, skippedCount, err := mapRawEvents(nil, readInteractions)
	if err != nil {
		response.ErrorCount++
		return response, err
	}
	response.SkippedCount += skippedCount
	response.NormalizedEventCount += len(normalizedEvents)

	recordResult, err := recordNormalizedEvents(ctx, u.learningEventRecorder, normalizedEvents)
	if err != nil {
		response.ErrorCount++
		return response, err
	}
	response.RecordedEventCount += recordResult.recordedCount
	response.DuplicateEventCount += recordResult.duplicateCount
	response.RecordedUserBatchCount += recordResult.userBatchCount
	return response, nil
}

type recordResult struct {
	recordedCount  int
	duplicateCount int
	userBatchCount int
}

func mapRawEvents(quizEvents []model.RawQuizEvent, interactions []model.RawLearningInteraction) ([]model.NormalizedLearningEvent, int, error) {
	normalizedEvents := make([]model.NormalizedLearningEvent, 0, len(quizEvents)+len(interactions))
	skippedCount := 0
	for _, raw := range quizEvents {
		result, err := rule.MapQuizEvent(raw)
		if err != nil {
			return nil, 0, err
		}
		if result.Skipped || result.Event == nil {
			skippedCount++
			continue
		}
		normalizedEvents = append(normalizedEvents, *result.Event)
	}
	for _, raw := range interactions {
		result, err := rule.MapLearningInteraction(raw)
		if err != nil {
			return nil, 0, err
		}
		if result.Skipped || result.Event == nil {
			skippedCount++
			continue
		}
		normalizedEvents = append(normalizedEvents, *result.Event)
	}
	return normalizedEvents, skippedCount, nil
}

func recordNormalizedEvents(ctx context.Context, recorder normalizerrepo.LearningEventRecorder, normalizedEvents []model.NormalizedLearningEvent) (recordResult, error) {
	if len(normalizedEvents) == 0 {
		return recordResult{}, nil
	}

	groups := groupByUser(normalizedEvents)
	userIDs := make([]string, 0, len(groups))
	for userID := range groups {
		userIDs = append(userIDs, userID)
	}
	sort.Strings(userIDs)

	result := recordResult{}
	for _, userID := range userIDs {
		recordResponse, err := recorder.Execute(ctx, learningdto.RecordLearningEventsRequest{
			UserID: userID,
			Events: toLearningInputs(groups[userID]),
		})
		if err != nil {
			return recordResult{}, err
		}
		result.recordedCount += recordResponse.RecordedCount
		result.duplicateCount += recordResponse.DuplicateCount
		result.userBatchCount++
	}

	return result, nil
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
