package videodetail

import (
	"net/http"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
)

func (h *Handler) getVideoDetail(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.service.Execute(r.Context(), apvdto.GetVideoDetailRequest{
		UserID:  principal.UserID,
		VideoID: videoID,
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
