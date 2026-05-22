package watchprogress

import (
	"context"
	"errors"
	"net/http"

	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
	catalogservice "learning-video-recommendation-system/internal/catalog/application/service"
)

type RecordVideoWatchProgressUsecase interface {
	Execute(ctx context.Context, request catalogdto.RecordVideoWatchProgressRequest) (catalogdto.RecordVideoWatchProgressResponse, error)
}

type Handler struct {
	recorder RecordVideoWatchProgressUsecase
}

func NewHandler(recorder RecordVideoWatchProgressUsecase) *Handler {
	return &Handler{recorder: recorder}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/video-watch-progress", h.recordVideoWatchProgress)
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

func invalidRequest(err error) error {
	if err == nil {
		return nil
	}
	return apiservice.InvalidRequestError(err.Error())
}
