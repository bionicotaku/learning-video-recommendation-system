package learningevents

import (
	"encoding/json"
	"net/http"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
)

type resetUserUnitProgressBody = selfMarkMasteredBody

func (h *Handler) resetUserUnitProgress(w http.ResponseWriter, r *http.Request) {
	principal, err := requiredPrincipal(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	if err := request.RequireJSONContentType(r); err != nil {
		writeHandlerError(w, r, invalidRequest(err))
		return
	}

	var body resetUserUnitProgressBody
	if err := request.DecodeJSONObject(r.Body, &body); err != nil {
		writeHandlerError(w, r, invalidRequest(err))
		return
	}

	command, err := mapResetUserUnitProgressBody(principal.UserID, body)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}

	result, err := h.resetProgress.Execute(r.Context(), command)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}

func mapResetUserUnitProgressBody(userID string, body resetUserUnitProgressBody) (apvdto.ResetUserUnitProgressRequest, error) {
	if len(body.ClientContext) == 0 {
		body.ClientContext = json.RawMessage(`{}`)
	}
	if err := request.ValidateJSONObject("client_context", body.ClientContext); err != nil {
		return apvdto.ResetUserUnitProgressRequest{}, invalidRequest(err)
	}
	if body.ClientEventID == "" {
		return apvdto.ResetUserUnitProgressRequest{}, apiservice.InvalidRequestError("client_event_id is required")
	}
	if body.CoarseUnitID <= 0 {
		return apvdto.ResetUserUnitProgressRequest{}, apiservice.InvalidRequestError("coarse_unit_id is required")
	}
	if body.SourceSurface == "" {
		return apvdto.ResetUserUnitProgressRequest{}, apiservice.InvalidRequestError("source_surface is required")
	}
	if err := validateOptionalUUIDs(map[string]string{
		"video_id":              body.VideoID,
		"watch_session_id":      body.WatchSessionID,
		"recommendation_run_id": body.RecommendationRunID,
		"related_quiz_event_id": body.RelatedQuizEventID,
	}); err != nil {
		return apvdto.ResetUserUnitProgressRequest{}, invalidRequest(err)
	}
	occurredAt, err := request.ParseRequiredTime("occurred_at", body.OccurredAt)
	if err != nil {
		return apvdto.ResetUserUnitProgressRequest{}, invalidRequest(err)
	}
	if len(body.EventPayload) == 0 {
		body.EventPayload = json.RawMessage(`{}`)
	}
	if err := request.ValidateJSONObject("event_payload", body.EventPayload); err != nil {
		return apvdto.ResetUserUnitProgressRequest{}, invalidRequest(err)
	}

	return apvdto.ResetUserUnitProgressRequest{
		UserID:              userID,
		ClientContext:       body.ClientContext,
		ClientEventID:       body.ClientEventID,
		CoarseUnitID:        body.CoarseUnitID,
		SourceSurface:       body.SourceSurface,
		VideoID:             body.VideoID,
		WatchSessionID:      body.WatchSessionID,
		RecommendationRunID: body.RecommendationRunID,
		RelatedQuizEventID:  body.RelatedQuizEventID,
		TokenText:           body.TokenText,
		SentenceIndex:       body.SentenceIndex,
		SpanIndex:           body.SpanIndex,
		OccurredAt:          occurredAt,
		EventPayload:        body.EventPayload,
	}, nil
}
