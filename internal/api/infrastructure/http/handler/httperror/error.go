package httperror

import (
	"context"
	"errors"
	"net/http"
	"strings"

	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	catalogservice "learning-video-recommendation-system/internal/catalog/application/service"
	learningservice "learning-video-recommendation-system/internal/learningengine/reducer/application/service"
	userrepo "learning-video-recommendation-system/internal/user/application/repository"
	userservice "learning-video-recommendation-system/internal/user/application/service"
)

const defaultPayloadTooLargeMessage = "request body must not exceed 1 MiB"

type Mapper func(error) *response.Error

func Write(w http.ResponseWriter, r *http.Request, err error, mappers ...Mapper) {
	requestID := middleware.RequestIDFromContext(r.Context())
	response.WriteError(w, requestID, Map(err, mappers...))
}

func Map(err error, mappers ...Mapper) *response.Error {
	if err == nil {
		return response.InternalError()
	}
	var responseErr *response.Error
	if errors.As(err, &responseErr) {
		return responseErr
	}
	switch {
	case errors.Is(err, auth.ErrMissingPrincipal):
		return response.Unauthorized("trusted principal is required")
	case apiservice.IsInvalidRequest(err):
		return response.InvalidRequest(err.Error())
	case apiservice.IsServiceUnavailable(err), errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return response.ServiceUnavailable("request canceled or timed out")
	}
	for _, mapper := range mappers {
		if mapper == nil {
			continue
		}
		if mapped := mapper(err); mapped != nil {
			return mapped
		}
	}
	return response.InternalError()
}

func InvalidRequest(err error) *response.Error {
	if IsPayloadTooLarge(err) {
		return response.PayloadTooLarge(defaultPayloadTooLargeMessage)
	}
	if err == nil {
		return response.InvalidRequest("")
	}
	return response.InvalidRequest(err.Error())
}

func PayloadTooLarge(message string) *response.Error {
	return response.PayloadTooLarge(message)
}

func IsPayloadTooLarge(err error) bool {
	if err == nil {
		return false
	}
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		return true
	}
	var responseErr *response.Error
	if errors.As(err, &responseErr) && responseErr.Code == "payload_too_large" {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "request body too large") ||
		strings.Contains(message, "multipart: message too large")
}

func CatalogValidation(err error) *response.Error {
	if catalogservice.IsValidationError(err) {
		return response.InvalidRequest(err.Error())
	}
	return nil
}

func CatalogNotFound(err error) *response.Error {
	if catalogservice.IsNotFoundError(err) {
		return response.NotFound(err.Error())
	}
	return nil
}

func CatalogConflict(err error) *response.Error {
	if catalogservice.IsConflictError(err) {
		return response.Conflict(err.Error())
	}
	return nil
}

func CatalogUnprocessable(err error) *response.Error {
	if catalogservice.IsUnprocessableError(err) {
		return response.UnprocessableEntity(err.Error())
	}
	return nil
}

func LearningValidation(err error) *response.Error {
	if learningservice.IsValidationError(err) {
		return response.InvalidRequest(err.Error())
	}
	return nil
}

func UserValidation(err error) *response.Error {
	if userservice.IsValidationError(err) {
		return response.InvalidRequest(err.Error())
	}
	return nil
}

func AuthUserNotFound(err error) *response.Error {
	if errors.Is(err, userrepo.ErrAuthUserNotFound) {
		return response.Unauthorized("trusted principal is required")
	}
	return nil
}

func UnitCollectionNotFound(err error) *response.Error {
	if errors.Is(err, learningservice.ErrUnitCollectionNotFound) {
		return response.NotFound("unit collection not found")
	}
	return nil
}
