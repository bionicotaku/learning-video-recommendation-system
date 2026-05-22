package videolibrary

import (
	"net/http"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
)

func (h *Handler) listVideoHistory(w http.ResponseWriter, r *http.Request) {
	principal, err := requiredPrincipal(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	limit, err := request.ParseOptionalLimit(r, 1, 100)
	if err != nil {
		writeHandlerError(w, r, apiservice.InvalidRequestError(err.Error()))
		return
	}
	result, err := h.service.ListHistory(r.Context(), apvdto.ListVideoHistoryRequest{
		UserID: principal.UserID,
		Limit:  limit,
		Cursor: request.ParseCursor(r),
	})
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}
