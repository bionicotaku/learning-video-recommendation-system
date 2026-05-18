package videointeractions

import (
	"net/http"

	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
)

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

	result, err := h.setLike.Execute(r.Context(), catalogdto.SetVideoLikeRequest{
		UserID:  principal.UserID,
		VideoID: videoID,
		Enabled: enabled,
	})
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}

func pathVideoID(r *http.Request) (string, error) {
	videoID := r.PathValue("video_id")
	if err := request.ValidateRequiredUUID("video_id", videoID); err != nil {
		return "", invalidRequest(err)
	}
	return videoID, nil
}
