package unitprogress

import (
	"net/http"

	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
)

func (h *Handler) listMastered(w http.ResponseWriter, r *http.Request) {
	h.listUnitProgress(w, r, learningdto.UnitProgressBucketMastered)
}

func (h *Handler) listUnmastered(w http.ResponseWriter, r *http.Request) {
	h.listUnitProgress(w, r, learningdto.UnitProgressBucketUnmastered)
}

func (h *Handler) listUnitProgress(w http.ResponseWriter, r *http.Request, bucket string) {
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

	result, err := h.usecase.Execute(r.Context(), learningdto.ListUserUnitProgressRequest{
		UserID: principal.UserID,
		Bucket: bucket,
		Limit:  limit,
		Cursor: request.ParseCursor(r),
	})
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}
