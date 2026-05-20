package unitprogress

import (
	"context"
	"errors"
	"net/http"

	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	learningservice "learning-video-recommendation-system/internal/learningengine/reducer/application/service"
)

type ListUserUnitProgressUsecase interface {
	Execute(ctx context.Context, request learningdto.ListUserUnitProgressRequest) (learningdto.ListUserUnitProgressResponse, error)
}

type Handler struct {
	usecase ListUserUnitProgressUsecase
}

func NewHandler(usecase ListUserUnitProgressUsecase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/learning/unit-progress/mastered", h.listMastered)
	mux.HandleFunc("GET /api/learning/unit-progress/unmastered", h.listUnmastered)
}

func writeHandlerError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.RequestIDFromContext(r.Context())
	switch {
	case errors.Is(err, auth.ErrMissingPrincipal):
		response.WriteError(w, requestID, response.Unauthorized("trusted principal is required"))
	case apiservice.IsInvalidRequest(err), learningservice.IsValidationError(err):
		response.WriteError(w, requestID, response.InvalidRequest(err.Error()))
	case apiservice.IsServiceUnavailable(err), errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		response.WriteError(w, requestID, response.ServiceUnavailable("request canceled or timed out"))
	default:
		response.WriteError(w, requestID, response.InternalError())
	}
}

func requiredPrincipal(r *http.Request) (auth.Principal, error) {
	return auth.RequirePrincipal(r.Context())
}
