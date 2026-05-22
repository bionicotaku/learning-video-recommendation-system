package feedback

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image/jpeg"
	"io"
	"math"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	userdto "learning-video-recommendation-system/internal/user/application/dto"
	userservice "learning-video-recommendation-system/internal/user/application/service"
)

const MaxRequestBytes int64 = 5 << 20

type SubmitFeedbackUsecase interface {
	Execute(ctx context.Context, request userdto.SubmitFeedbackRequest) (userdto.SubmitFeedbackResponse, error)
}

type Handler struct {
	submitter SubmitFeedbackUsecase
}

func NewHandler(submitter SubmitFeedbackUsecase) *Handler {
	return &Handler{submitter: submitter}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/feedback", h.submitFeedback)
}

func (h *Handler) submitFeedback(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	if err := validateMultipartContentType(r); err != nil {
		writeHandlerError(w, r, err)
		return
	}
	command, err := decodeMultipartFeedback(r, principal.UserID)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	result, err := h.submitter.Execute(r.Context(), command)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}

func decodeMultipartFeedback(r *http.Request, userID string) (userdto.SubmitFeedbackRequest, error) {
	if err := r.ParseMultipartForm(MaxRequestBytes); err != nil {
		if isPayloadTooLargeError(err) {
			return userdto.SubmitFeedbackRequest{}, payloadTooLargeError()
		}
		return userdto.SubmitFeedbackRequest{}, apiservice.InvalidRequestError("invalid multipart form")
	}
	if r.MultipartForm == nil {
		return userdto.SubmitFeedbackRequest{}, apiservice.InvalidRequestError("multipart form is required")
	}
	defer func(form *multipart.Form) { _ = form.RemoveAll() }(r.MultipartForm)

	for field := range r.MultipartForm.Value {
		if field != "payload" && field != "client_feedback_id" {
			return userdto.SubmitFeedbackRequest{}, apiservice.InvalidRequestError("unexpected form field")
		}
	}
	for field := range r.MultipartForm.File {
		if field != "images" {
			return userdto.SubmitFeedbackRequest{}, apiservice.InvalidRequestError("unexpected file field")
		}
	}

	payload, err := singleFormValue(r.MultipartForm, "payload", true)
	if err != nil {
		return userdto.SubmitFeedbackRequest{}, err
	}
	payloadBytes := []byte(payload)
	if err := request.ValidateJSONObject("payload", json.RawMessage(payloadBytes)); err != nil {
		return userdto.SubmitFeedbackRequest{}, apiservice.InvalidRequestError(err.Error())
	}

	clientFeedbackIDValue, err := singleFormValue(r.MultipartForm, "client_feedback_id", false)
	if err != nil {
		return userdto.SubmitFeedbackRequest{}, err
	}
	var clientFeedbackID *string
	if clientFeedbackIDValue != "" {
		if err := request.ValidateOptionalUUID("client_feedback_id", clientFeedbackIDValue); err != nil {
			return userdto.SubmitFeedbackRequest{}, apiservice.InvalidRequestError(err.Error())
		}
		clientFeedbackID = &clientFeedbackIDValue
	}

	files := r.MultipartForm.File["images"]
	if len(files) > 5 {
		return userdto.SubmitFeedbackRequest{}, apiservice.InvalidRequestError("images must contain at most 5 files")
	}
	images := make([]userdto.FeedbackImageInput, 0, len(files))
	for index, file := range files {
		image, err := decodeFeedbackImage(index, file)
		if err != nil {
			return userdto.SubmitFeedbackRequest{}, err
		}
		images = append(images, image)
	}
	return userdto.SubmitFeedbackRequest{
		UserID:           userID,
		ClientFeedbackID: clientFeedbackID,
		Payload:          payloadBytes,
		Images:           images,
	}, nil
}

func singleFormValue(form *multipart.Form, field string, required bool) (string, error) {
	values := form.Value[field]
	if len(values) == 0 {
		if required {
			return "", apiservice.InvalidRequestError(field + " is required")
		}
		return "", nil
	}
	if len(values) != 1 {
		return "", apiservice.InvalidRequestError(field + " must be provided once")
	}
	return values[0], nil
}

func decodeFeedbackImage(index int, fileHeader *multipart.FileHeader) (userdto.FeedbackImageInput, error) {
	contentType, err := normalizedJPEGContentType(fileHeader.Header.Get("Content-Type"))
	if err != nil {
		return userdto.FeedbackImageInput{}, err
	}
	file, err := fileHeader.Open()
	if err != nil {
		return userdto.FeedbackImageInput{}, apiservice.InvalidRequestError("image file is invalid")
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(file)
	if err != nil {
		if isPayloadTooLargeError(err) {
			return userdto.FeedbackImageInput{}, payloadTooLargeError()
		}
		return userdto.FeedbackImageInput{}, apiservice.InvalidRequestError("image file is invalid")
	}
	if len(data) > math.MaxInt32 {
		return userdto.FeedbackImageInput{}, apiservice.InvalidRequestError("image file is too large")
	}
	if len(data) < 3 || data[0] != 0xff || data[1] != 0xd8 || data[2] != 0xff {
		return userdto.FeedbackImageInput{}, apiservice.InvalidRequestError("images must be JPEG files")
	}
	config, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return userdto.FeedbackImageInput{}, apiservice.InvalidRequestError("images must be valid JPEG files")
	}
	hash := sha256.Sum256(data)
	return userdto.FeedbackImageInput{
		SortOrder:   int32(index + 1),
		ContentType: contentType,
		SizeBytes:   int32(len(data)),
		SHA256:      hex.EncodeToString(hash[:]),
		Width:       int32(config.Width),
		Height:      int32(config.Height),
		Data:        data,
	}, nil
}

func normalizedJPEGContentType(contentType string) (string, error) {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", apiservice.InvalidRequestError("images must be JPEG files")
	}
	if mediaType != "image/jpeg" && mediaType != "image/jpg" {
		return "", apiservice.InvalidRequestError("images must be JPEG files")
	}
	return "image/jpeg", nil
}

func validateMultipartContentType(r *http.Request) error {
	contentType := r.Header.Get("Content-Type")
	if strings.TrimSpace(contentType) == "" {
		return apiservice.InvalidRequestError("content-type must be multipart/form-data")
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "multipart/form-data" {
		return apiservice.InvalidRequestError("content-type must be multipart/form-data")
	}
	return nil
}

func writeHandlerError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.RequestIDFromContext(r.Context())
	switch {
	case errors.Is(err, auth.ErrMissingPrincipal):
		response.WriteError(w, requestID, response.Unauthorized("trusted principal is required"))
	case isPayloadTooLargeError(err):
		response.WriteError(w, requestID, response.PayloadTooLarge("request body must not exceed 5 MiB"))
	case apiservice.IsInvalidRequest(err), userservice.IsValidationError(err):
		response.WriteError(w, requestID, response.InvalidRequest(err.Error()))
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled), apiservice.IsServiceUnavailable(err):
		response.WriteError(w, requestID, response.ServiceUnavailable("request canceled or timed out"))
	default:
		response.WriteError(w, requestID, response.InternalError())
	}
}

func payloadTooLargeError() error {
	return response.PayloadTooLarge("request body must not exceed 5 MiB")
}

func isPayloadTooLargeError(err error) bool {
	if err == nil {
		return false
	}
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError) ||
		strings.Contains(strings.ToLower(err.Error()), "request body too large") ||
		strings.Contains(strings.ToLower(err.Error()), "multipart: message too large") ||
		isResponsePayloadTooLarge(err)
}

func isResponsePayloadTooLarge(err error) bool {
	var responseError *response.Error
	return errors.As(err, &responseError) && responseError.Code == "payload_too_large"
}
