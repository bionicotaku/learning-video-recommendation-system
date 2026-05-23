package learningtargets

import (
	"context"
	"net/http"

	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/httperror"
	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
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
	httperror.Write(w, r, err,
		httperror.LearningValidation,
		httperror.UserValidation,
		httperror.AuthUserNotFound,
		httperror.UnitCollectionNotFound,
	)
}

func requiredPrincipal(r *http.Request) (auth.Principal, error) {
	return auth.RequirePrincipal(r.Context())
}

func invalidRequest(err error) error {
	if err == nil {
		return nil
	}
	return httperror.InvalidRequest(err)
}
