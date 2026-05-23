package videointeractions

import (
	"net/http"
	"time"

	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
)

type interactionRequestBody struct {
	OccurredAt time.Time `json:"occurred_at"`
}

func (h *Handler) setVideoLike(w http.ResponseWriter, r *http.Request) {
	h.handleVideoLike(w, r, true)
}

func (h *Handler) unsetVideoLike(w http.ResponseWriter, r *http.Request) {
	h.handleVideoLike(w, r, false)
}

func (h *Handler) handleVideoLike(w http.ResponseWriter, r *http.Request, enabled bool) {
	principal, err := requiredPrincipal(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	videoID, err := pathVideoID(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	occurredAt, err := parseInteractionOccurredAt(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}

	result, err := h.setLike.Execute(r.Context(), catalogdto.SetVideoLikeRequest{
		UserID:     principal.UserID,
		VideoID:    videoID,
		Enabled:    enabled,
		OccurredAt: occurredAt,
	})
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}

func pathVideoID(r *http.Request) (string, error) {
	videoID, err := request.PathRequiredUUID(r, "video_id")
	if err != nil {
		return "", invalidRequest(err)
	}
	return videoID, nil
}

func parseInteractionOccurredAt(r *http.Request) (time.Time, error) {
	if err := request.RequireJSONContentType(r); err != nil {
		return time.Time{}, invalidRequest(err)
	}
	var body interactionRequestBody
	if err := request.DecodeJSONObject(r.Body, &body); err != nil {
		return time.Time{}, invalidRequest(err)
	}
	if body.OccurredAt.IsZero() {
		return time.Time{}, apiservice.InvalidRequestError("occurred_at is required")
	}
	return body.OccurredAt.UTC(), nil
}
