package learningtargets

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
	userrepo "learning-video-recommendation-system/internal/user/application/repository"
	userservice "learning-video-recommendation-system/internal/user/application/service"
)

type ActivateUnitCollectionTargetUsecase interface {
	Execute(ctx context.Context, request learningdto.ActivateUnitCollectionTargetRequest) (learningdto.ActivateUnitCollectionTargetResponse, error)
}

type GetActiveLearningTargetCoarseUnitIDsUsecase interface {
	Execute(ctx context.Context, request learningdto.GetActiveLearningTargetCoarseUnitIDsRequest) (learningdto.GetActiveLearningTargetCoarseUnitIDsResponse, error)
}

type Handler struct {
	activateTarget      ActivateUnitCollectionTargetUsecase
	activeTargetUnitIDs GetActiveLearningTargetCoarseUnitIDsUsecase
}

func NewHandler(
	activateTarget ActivateUnitCollectionTargetUsecase,
	activeTargetUnitIDs GetActiveLearningTargetCoarseUnitIDsUsecase,
) *Handler {
	return &Handler{
		activateTarget:      activateTarget,
		activeTargetUnitIDs: activeTargetUnitIDs,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/learning-targets/active-coarse-unit-ids", h.getActiveLearningTargetCoarseUnitIDs)
	mux.HandleFunc("PUT /api/learning-targets/active-collection", h.activateUnitCollection)
}

func writeHandlerError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.RequestIDFromContext(r.Context())
	switch {
	case errors.Is(err, auth.ErrMissingPrincipal):
		response.WriteError(w, requestID, response.Unauthorized("trusted principal is required"))
	case apiservice.IsInvalidRequest(err), learningservice.IsValidationError(err), userservice.IsValidationError(err):
		response.WriteError(w, requestID, response.InvalidRequest(err.Error()))
	case errors.Is(err, userrepo.ErrAuthUserNotFound):
		response.WriteError(w, requestID, response.Unauthorized("trusted principal is required"))
	case errors.Is(err, learningservice.ErrUnitCollectionNotFound):
		response.WriteError(w, requestID, response.NotFound("unit collection not found"))
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
