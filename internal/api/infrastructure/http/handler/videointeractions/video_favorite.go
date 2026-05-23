package videointeractions

import (
	"net/http"

	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
)

func (h *Handler) setVideoFavorite(w http.ResponseWriter, r *http.Request) {
	h.handleVideoFavorite(w, r, true)
}

func (h *Handler) unsetVideoFavorite(w http.ResponseWriter, r *http.Request) {
	h.handleVideoFavorite(w, r, false)
}

func (h *Handler) handleVideoFavorite(w http.ResponseWriter, r *http.Request, enabled bool) {
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

	result, err := h.setFavorite.Execute(r.Context(), catalogdto.SetVideoFavoriteRequest{
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
