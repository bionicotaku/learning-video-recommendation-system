package wordfavorites

import (
	"net/http"

	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
)

func (h *Handler) listFavorites(w http.ResponseWriter, r *http.Request) {
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
	result, err := h.list.Execute(r.Context(), catalogdto.ListWordFavoritesRequest{
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
