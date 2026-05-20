package unitprogress

import (
	"net/http"
	"strconv"
	"strings"

	apiservice "learning-video-recommendation-system/internal/api/application/service"
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

	limit, err := parseLimit(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}

	result, err := h.usecase.Execute(r.Context(), learningdto.ListUserUnitProgressRequest{
		UserID: principal.UserID,
		Bucket: bucket,
		Limit:  limit,
		Cursor: strings.TrimSpace(r.URL.Query().Get("cursor")),
	})
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}

func parseLimit(r *http.Request) (int, error) {
	rawLimit := strings.TrimSpace(r.URL.Query().Get("limit"))
	if rawLimit == "" {
		return 0, nil
	}
	limit, err := strconv.Atoi(rawLimit)
	if err != nil {
		return 0, apiservice.InvalidRequestError("limit must be an integer")
	}
	if limit < 1 || limit > 100 {
		return 0, apiservice.InvalidRequestError("limit must be between 1 and 100")
	}
	return limit, nil
}
