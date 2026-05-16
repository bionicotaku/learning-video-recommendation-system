package learningevents

import (
	"encoding/json"
	"net/http"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
)

type selfMarkMasteredBody struct {
	ClientContext json.RawMessage `json:"client_context"`

	ClientEventID       string          `json:"client_event_id"`
	CoarseUnitID        int64           `json:"coarse_unit_id"`
	SourceSurface       string          `json:"source_surface"`
	VideoID             string          `json:"video_id"`
	WatchSessionID      string          `json:"watch_session_id"`
	RecommendationRunID string          `json:"recommendation_run_id"`
	RelatedQuizEventID  string          `json:"related_quiz_event_id"`
	TokenText           string          `json:"token_text"`
	SentenceIndex       *int32          `json:"sentence_index"`
	SpanIndex           *int32          `json:"span_index"`
	OccurredAt          string          `json:"occurred_at"`
	EventPayload        json.RawMessage `json:"event_payload"`
}

func (h *Handler) recordSelfMarkMastered(w http.ResponseWriter, r *http.Request) {
	principal, err := requiredPrincipal(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	if err := validateContentType(r); err != nil {
		writeHandlerError(w, r, err)
		return
	}

	var body selfMarkMasteredBody
	if err := request.DecodeJSONObject(r.Body, &body); err != nil {
		writeHandlerError(w, r, invalidRequest(err))
		return
	}

	command, err := mapSelfMarkMasteredBody(principal.UserID, body)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}

	result, err := h.selfMarkMastered.Execute(r.Context(), command)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}

func mapSelfMarkMasteredBody(userID string, body selfMarkMasteredBody) (apvdto.RecordSelfMarkMasteredRequest, error) {
	if len(body.ClientContext) == 0 {
		body.ClientContext = json.RawMessage(`{}`)
	}
	if err := request.ValidateJSONObject("client_context", body.ClientContext); err != nil {
		return apvdto.RecordSelfMarkMasteredRequest{}, invalidRequest(err)
	}
	if body.ClientEventID == "" {
		return apvdto.RecordSelfMarkMasteredRequest{}, apiservice.InvalidRequestError("client_event_id is required")
	}
	if body.CoarseUnitID <= 0 {
		return apvdto.RecordSelfMarkMasteredRequest{}, apiservice.InvalidRequestError("coarse_unit_id is required")
	}
	if body.SourceSurface == "" {
		return apvdto.RecordSelfMarkMasteredRequest{}, apiservice.InvalidRequestError("source_surface is required")
	}
	if err := validateOptionalUUIDs(map[string]string{
		"video_id":              body.VideoID,
		"watch_session_id":      body.WatchSessionID,
		"recommendation_run_id": body.RecommendationRunID,
		"related_quiz_event_id": body.RelatedQuizEventID,
	}); err != nil {
		return apvdto.RecordSelfMarkMasteredRequest{}, invalidRequest(err)
	}
	occurredAt, err := request.ParseRequiredTime("occurred_at", body.OccurredAt)
	if err != nil {
		return apvdto.RecordSelfMarkMasteredRequest{}, invalidRequest(err)
	}
	if len(body.EventPayload) == 0 {
		body.EventPayload = json.RawMessage(`{}`)
	}
	if err := request.ValidateJSONObject("event_payload", body.EventPayload); err != nil {
		return apvdto.RecordSelfMarkMasteredRequest{}, invalidRequest(err)
	}

	return apvdto.RecordSelfMarkMasteredRequest{
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
