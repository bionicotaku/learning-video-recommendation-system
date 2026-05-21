package unitcollections

import (
	"net/http"

	apivdto "learning-video-recommendation-system/internal/api/application/dto"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
)

func (h *Handler) listUnitCollections(w http.ResponseWriter, r *http.Request) {
	principal, err := requiredPrincipal(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	result, err := h.listCollections.Execute(r.Context(), apivdto.ListUnitCollectionsRequest{
		UserID: principal.UserID,
	})
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}
