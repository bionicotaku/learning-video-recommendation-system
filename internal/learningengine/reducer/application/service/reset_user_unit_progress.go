package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	apprepo "learning-video-recommendation-system/internal/learningengine/reducer/application/repository"
	appusecase "learning-video-recommendation-system/internal/learningengine/reducer/application/usecase"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/aggregate"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
)

const resetUserUnitProgressSourceType = "learning_unit_reset"

type ResetUserUnitProgressUsecase struct {
	txManager TxManager
}

var _ appusecase.ResetUserUnitProgressUsecase = (*ResetUserUnitProgressUsecase)(nil)

func NewResetUserUnitProgressUsecase(txManager TxManager) *ResetUserUnitProgressUsecase {
	return &ResetUserUnitProgressUsecase{txManager: txManager}
}

func (u *ResetUserUnitProgressUsecase) Execute(ctx context.Context, request dto.ResetUserUnitProgressRequest) (dto.ResetUserUnitProgressResponse, error) {
	if request.UserID == "" {
		return dto.ResetUserUnitProgressResponse{}, fmt.Errorf("user_id is required")
	}
	if request.ClientEventID == "" {
		return dto.ResetUserUnitProgressResponse{}, validationError("client_event_id is required")
	}
	if request.CoarseUnitID <= 0 {
		return dto.ResetUserUnitProgressResponse{}, validationError("coarse_unit_id is required")
	}
	if request.SourceSurface == "" {
		return dto.ResetUserUnitProgressResponse{}, validationError("source_surface is required")
	}
	if request.OccurredAt.IsZero() {
		return dto.ResetUserUnitProgressResponse{}, validationError("occurred_at is required")
	}

	metadata, err := resetUserUnitProgressMetadata(request)
	if err != nil {
		return dto.ResetUserUnitProgressResponse{}, err
	}

	response := dto.ResetUserUnitProgressResponse{Accepted: true}
	err = u.txManager.WithinUserTx(ctx, request.UserID, func(ctx context.Context, repos TransactionalRepositories) error {
		state, err := repos.UserUnitStates().GetByUserAndUnitForUpdate(ctx, request.UserID, request.CoarseUnitID)
		if err != nil {
			return err
		}
		if state == nil {
			return ErrUserUnitStateNotFound
		}

		loadExisting := func() error {
			existing, err := repos.UnitLearningEvents().GetByUserSourceRef(ctx, request.UserID, resetUserUnitProgressSourceType, request.ClientEventID)
			if err != nil {
				return err
			}
			if existing == nil {
				return fmt.Errorf("reset learning event %q not found after duplicate append", request.ClientEventID)
			}
			response.UnitLearningEventID = existing.EventID
			response.Inserted = false
			return nil
		}

		existing, err := repos.UnitLearningEvents().GetByUserSourceRef(ctx, request.UserID, resetUserUnitProgressSourceType, request.ClientEventID)
		if err != nil {
			return err
		}
		if existing != nil {
			response.UnitLearningEventID = existing.EventID
			response.Inserted = false
			return nil
		}

		watermarks, err := repos.UnitLearningEvents().ListWatermarksByUserUnits(ctx, request.UserID, []int64{request.CoarseUnitID})
		if err != nil {
			return err
		}
		resetBoundaryAt := resetBoundaryFromWatermark(request.OccurredAt.UTC(), watermarks[request.CoarseUnitID])

		event := model.LearningEvent{
			UserID:          request.UserID,
			CoarseUnitID:    request.CoarseUnitID,
			VideoID:         request.VideoID,
			EventType:       enum.EventResetUnlearned,
			ReducerEffect:   enum.ReducerEffectResetUnlearned,
			SourceType:      resetUserUnitProgressSourceType,
			SourceRefID:     request.ClientEventID,
			Metadata:        metadata,
			OccurredAt:      request.OccurredAt.UTC(),
			ResetBoundaryAt: &resetBoundaryAt,
		}
		appendResult, err := repos.UnitLearningEvents().Append(ctx, []model.LearningEvent{event})
		if err != nil {
			if errors.Is(err, apprepo.ErrDuplicateResetClientEvent) {
				return loadExisting()
			}
			return err
		}

		if len(appendResult.InsertedEvents) == 0 {
			return loadExisting()
		}

		inserted := appendResult.InsertedEvents[0]
		nextState, err := aggregate.Reduce(state, inserted)
		if err != nil {
			return err
		}
		if _, err := repos.UserUnitStates().BatchUpsert(ctx, []*model.UserUnitState{nextState}); err != nil {
			return err
		}
		response.UnitLearningEventID = inserted.EventID
		response.Inserted = true
		return nil
	})
	if err != nil {
		return dto.ResetUserUnitProgressResponse{}, err
	}
	return response, nil
}

func resetBoundaryFromWatermark(clientOccurredAt time.Time, watermark model.UnitLearningEventWatermark) time.Time {
	boundary := clientOccurredAt
	if watermark.MaxOccurredAt != nil && watermark.MaxOccurredAt.After(boundary) {
		boundary = *watermark.MaxOccurredAt
	}
	if watermark.MaxResetBoundaryAt != nil && watermark.MaxResetBoundaryAt.After(boundary) {
		boundary = *watermark.MaxResetBoundaryAt
	}
	return boundary.UTC()
}

func resetUserUnitProgressMetadata(request dto.ResetUserUnitProgressRequest) ([]byte, error) {
	clientContext := request.ClientContext
	if len(clientContext) == 0 {
		clientContext = []byte("{}")
	}
	eventPayload := request.EventPayload
	if len(eventPayload) == 0 {
		eventPayload = []byte("{}")
	}

	metadata := map[string]any{
		"client_context":     clientContextJSON(clientContext),
		"client_occurred_at": request.OccurredAt.UTC().Format(time.RFC3339Nano),
		"event_payload":      clientContextJSON(eventPayload),
		"source_surface":     request.SourceSurface,
	}
	if request.WatchSessionID != "" {
		metadata["watch_session_id"] = request.WatchSessionID
	}
	if request.RecommendationRunID != "" {
		metadata["recommendation_run_id"] = request.RecommendationRunID
	}
	if request.RelatedQuizEventID != "" {
		metadata["related_quiz_event_id"] = request.RelatedQuizEventID
	}
	if request.TokenText != "" {
		metadata["token_text"] = request.TokenText
	}
	if request.SentenceIndex != nil {
		metadata["sentence_index"] = *request.SentenceIndex
	}
	if request.SpanIndex != nil {
		metadata["span_index"] = *request.SpanIndex
	}
	return json.Marshal(metadata)
}

func clientContextJSON(raw []byte) json.RawMessage {
	return json.RawMessage(raw)
}
