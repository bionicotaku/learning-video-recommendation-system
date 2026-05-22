package unitcollections

import (
	"context"
	"errors"
	"net/http"

	apivdto "learning-video-recommendation-system/internal/api/application/dto"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	learningservice "learning-video-recommendation-system/internal/learningengine/reducer/application/service"
	userservice "learning-video-recommendation-system/internal/user/application/service"
)

type ListUnitCollectionsUsecase interface {
	Execute(ctx context.Context, request apivdto.ListUnitCollectionsRequest) (apivdto.UnitCollectionsResponse, error)
}

type Handler struct {
	listCollections ListUnitCollectionsUsecase
}

func NewHandler(listCollections ListUnitCollectionsUsecase) *Handler {
	return &Handler{listCollections: listCollections}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/unit-collections", h.listUnitCollections)
}

func writeHandlerError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.RequestIDFromContext(r.Context())
	switch {
	case errors.Is(err, auth.ErrMissingPrincipal):
		response.WriteError(w, requestID, response.Unauthorized("trusted principal is required"))
	case apiservice.IsInvalidRequest(err), learningservice.IsValidationError(err), userservice.IsValidationError(err):
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

func invalidRequest(err error) error {
	if err == nil {
		return nil
	}
	return apiservice.InvalidRequestError(err.Error())
}
