package unitcollections

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
	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	learningservice "learning-video-recommendation-system/internal/learningengine/reducer/application/service"
	semanticdto "learning-video-recommendation-system/internal/semantic/application/dto"
	userrepo "learning-video-recommendation-system/internal/user/application/repository"
	userservice "learning-video-recommendation-system/internal/user/application/service"
)

type ListUnitCollectionsUsecase interface {
	Execute(ctx context.Context) (semanticdto.ListUnitCollectionsResponse, error)
}

type ActivateUnitCollectionTargetUsecase interface {
	Execute(ctx context.Context, request learningdto.ActivateUnitCollectionTargetRequest) (learningdto.ActivateUnitCollectionTargetResponse, error)
}

type Handler struct {
	listCollections ListUnitCollectionsUsecase
	activateTarget  ActivateUnitCollectionTargetUsecase
}

func NewHandler(listCollections ListUnitCollectionsUsecase, activateTarget ActivateUnitCollectionTargetUsecase) *Handler {
	return &Handler{listCollections: listCollections, activateTarget: activateTarget}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/unit-collections", h.listUnitCollections)
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
