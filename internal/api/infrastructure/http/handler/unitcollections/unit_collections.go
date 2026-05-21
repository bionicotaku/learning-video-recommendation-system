package unitcollections

import (
	"net/http"

	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
)

func (h *Handler) listUnitCollections(w http.ResponseWriter, r *http.Request) {
	result, err := h.listCollections.Execute(r.Context())
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}
