package legaldocuments

import (
	"context"
	"errors"
	"net/http"
	"strings"

	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	userdto "learning-video-recommendation-system/internal/user/application/dto"
	userservice "learning-video-recommendation-system/internal/user/application/service"
)

type GetLegalDocumentUsecase interface {
	Execute(ctx context.Context, request userdto.GetLegalDocumentRequest) (userdto.GetLegalDocumentResponse, error)
}

type Handler struct {
	getDocument GetLegalDocumentUsecase
}

func NewHandler(getDocument GetLegalDocumentUsecase) *Handler {
	return &Handler{getDocument: getDocument}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/legal-documents/{type}", h.getLegalDocument)
}

func (h *Handler) getLegalDocument(w http.ResponseWriter, r *http.Request) {
	result, err := h.getDocument.Execute(r.Context(), userdto.GetLegalDocumentRequest{
		Type: strings.TrimSpace(r.PathValue("type")),
	})
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}

func writeHandlerError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.RequestIDFromContext(r.Context())
	switch {
	case apiservice.IsInvalidRequest(err), userservice.IsValidationError(err):
		response.WriteError(w, requestID, response.InvalidRequest(err.Error()))
	case apiservice.IsServiceUnavailable(err), errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		response.WriteError(w, requestID, response.ServiceUnavailable("request canceled or timed out"))
	default:
		response.WriteError(w, requestID, response.InternalError())
	}
}
