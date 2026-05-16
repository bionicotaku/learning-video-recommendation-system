package endquiz

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"strings"

	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
	catalogservice "learning-video-recommendation-system/internal/catalog/application/service"
)

type EndQuizQuestionLookupUsecase interface {
	Execute(ctx context.Context, request catalogdto.EndQuizQuestionLookupRequest) (catalogdto.EndQuizQuestionLookupResponse, error)
}

type Handler struct {
	lookup EndQuizQuestionLookupUsecase
}

func NewHandler(lookup EndQuizQuestionLookupUsecase) *Handler {
	return &Handler{lookup: lookup}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/videos/end-quiz", h.getEndQuiz)
}

func writeHandlerError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.RequestIDFromContext(r.Context())
	switch {
	case errors.Is(err, auth.ErrMissingPrincipal):
		response.WriteError(w, requestID, response.Unauthorized("trusted principal is required"))
	case apiservice.IsInvalidRequest(err), catalogservice.IsValidationError(err):
		response.WriteError(w, requestID, response.InvalidRequest(err.Error()))
	case catalogservice.IsNotFoundError(err):
		response.WriteError(w, requestID, response.NotFound(err.Error()))
	case catalogservice.IsConflictError(err):
		response.WriteError(w, requestID, response.Conflict(err.Error()))
	case catalogservice.IsUnprocessableError(err):
		response.WriteError(w, requestID, response.UnprocessableEntity(err.Error()))
	case apiservice.IsServiceUnavailable(err), errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		response.WriteError(w, requestID, response.ServiceUnavailable("request canceled or timed out"))
	default:
		response.WriteError(w, requestID, response.InternalError())
	}
}

func requiredPrincipal(r *http.Request) (auth.Principal, error) {
	return auth.RequirePrincipal(r.Context())
}

func validateContentType(r *http.Request) error {
	contentType := r.Header.Get("Content-Type")
	if strings.TrimSpace(contentType) == "" {
		return apiservice.InvalidRequestError("content-type must be application/json")
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil && mediaType == "application/json" {
		return nil
	}
	return apiservice.InvalidRequestError("content-type must be application/json")
}

func invalidRequest(err error) error {
	if err == nil {
		return nil
	}
	return apiservice.InvalidRequestError(err.Error())
}
