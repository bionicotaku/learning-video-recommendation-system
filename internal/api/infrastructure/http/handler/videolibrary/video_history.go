package videolibrary

import (
	"net/http"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
)

func (h *Handler) listVideoHistory(w http.ResponseWriter, r *http.Request) {
	principal, err := requiredPrincipal(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	limit, err := parseLimit(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	result, err := h.service.ListHistory(r.Context(), apvdto.ListVideoHistoryRequest{
		UserID: principal.UserID,
		Limit:  limit,
		Cursor: parseCursor(r),
	})
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}
