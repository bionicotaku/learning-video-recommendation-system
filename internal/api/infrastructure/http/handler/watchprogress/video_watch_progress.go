package watchprogress

import (
	"encoding/json"
	"net/http"

	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
)

type videoWatchProgressBody struct {
	VideoID        string          `json:"video_id"`
	WatchSessionID string          `json:"watch_session_id"`
	PositionMS     int32           `json:"position_ms"`
	ActiveWatchMS  int64           `json:"active_watch_ms"`
	OccurredAt     string          `json:"occurred_at"`
	SourceSurface  string          `json:"source_surface"`
	ClientContext  json.RawMessage `json:"client_context"`
	Metadata       json.RawMessage `json:"metadata"`
}

func (h *Handler) recordVideoWatchProgress(w http.ResponseWriter, r *http.Request) {
	principal, err := requiredPrincipal(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	if err := request.RequireJSONContentType(r); err != nil {
		writeHandlerError(w, r, invalidRequest(err))
		return
	}

	var body videoWatchProgressBody
	if err := request.DecodeJSONObject(r.Body, &body); err != nil {
		writeHandlerError(w, r, invalidRequest(err))
		return
	}

	command, err := mapVideoWatchProgressBody(principal.UserID, body)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}

	result, err := h.recorder.Execute(r.Context(), command)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}

func mapVideoWatchProgressBody(userID string, body videoWatchProgressBody) (catalogdto.RecordVideoWatchProgressRequest, error) {
	if err := request.ValidateRequiredUUID("video_id", body.VideoID); err != nil {
		return catalogdto.RecordVideoWatchProgressRequest{}, invalidRequest(err)
	}
	if err := request.ValidateRequiredUUID("watch_session_id", body.WatchSessionID); err != nil {
		return catalogdto.RecordVideoWatchProgressRequest{}, invalidRequest(err)
	}
	if body.PositionMS < 0 {
		return catalogdto.RecordVideoWatchProgressRequest{}, apiservice.InvalidRequestError("position_ms must be non-negative")
	}
	if body.ActiveWatchMS < 0 {
		return catalogdto.RecordVideoWatchProgressRequest{}, apiservice.InvalidRequestError("active_watch_ms must be non-negative")
	}
	occurredAt, err := request.ParseOptionalTime("occurred_at", body.OccurredAt)
	if err != nil {
		return catalogdto.RecordVideoWatchProgressRequest{}, invalidRequest(err)
	}
	if len(body.ClientContext) == 0 {
		body.ClientContext = json.RawMessage(`{}`)
	}
	if err := request.ValidateJSONObject("client_context", body.ClientContext); err != nil {
		return catalogdto.RecordVideoWatchProgressRequest{}, invalidRequest(err)
	}
	if len(body.Metadata) == 0 {
		body.Metadata = json.RawMessage(`{}`)
	}
	if err := request.ValidateJSONObject("metadata", body.Metadata); err != nil {
		return catalogdto.RecordVideoWatchProgressRequest{}, invalidRequest(err)
	}

	return catalogdto.RecordVideoWatchProgressRequest{
		UserID:         userID,
		VideoID:        body.VideoID,
		WatchSessionID: body.WatchSessionID,
		PositionMS:     body.PositionMS,
		ActiveWatchMS:  body.ActiveWatchMS,
		OccurredAt:     occurredAt,
		SourceSurface:  body.SourceSurface,
		ClientContext:  body.ClientContext,
		Metadata:       body.Metadata,
	}, nil
}
