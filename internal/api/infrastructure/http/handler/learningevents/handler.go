package learningevents

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"strings"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
)

type RecordLearningInteractionsBatchService interface {
	Execute(ctx context.Context, request apvdto.RecordLearningInteractionsBatchRequest) (apvdto.RecordLearningInteractionsBatchResponse, error)
}

type RecordQuizAttemptService interface {
	Execute(ctx context.Context, request apvdto.RecordQuizAttemptRequest) (apvdto.RecordQuizAttemptResponse, error)
}

type RecordSelfMarkMasteredService interface {
	Execute(ctx context.Context, request apvdto.RecordSelfMarkMasteredRequest) (apvdto.RecordSelfMarkMasteredResponse, error)
}

type Handler struct {
	learningInteractions RecordLearningInteractionsBatchService
	quizAttempts         RecordQuizAttemptService
	selfMarkMastered     RecordSelfMarkMasteredService
}

func NewHandler(
	learningInteractions RecordLearningInteractionsBatchService,
	quizAttempts RecordQuizAttemptService,
	selfMarkMastered RecordSelfMarkMasteredService,
) *Handler {
	return &Handler{
		learningInteractions: learningInteractions,
		quizAttempts:         quizAttempts,
		selfMarkMastered:     selfMarkMastered,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/learning-interactions:batch", h.recordLearningInteractionsBatch)
	mux.HandleFunc("POST /api/quiz-attempts", h.recordQuizAttempt)
	mux.HandleFunc("POST /api/learning-units:mark-mastered", h.recordSelfMarkMastered)
}

func writeHandlerError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.RequestIDFromContext(r.Context())
	switch {
	case errors.Is(err, auth.ErrMissingPrincipal):
		response.WriteError(w, requestID, response.Unauthorized("trusted principal is required"))
	case apiservice.IsInvalidRequest(err):
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

func validateOptionalUUIDs(values map[string]string) error {
	for field, value := range values {
		if err := request.ValidateOptionalUUID(field, value); err != nil {
			return err
		}
	}
	return nil
}
