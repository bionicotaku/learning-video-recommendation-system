package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"learning-video-recommendation-system/internal/user/application/dto"
	"learning-video-recommendation-system/internal/user/application/repository"
	"learning-video-recommendation-system/internal/user/domain/model"
)

const maxFeedbackImages = 5

type SubmitFeedbackUsecase struct {
	writer repository.FeedbackWriter
}

func NewSubmitFeedbackUsecase(writer repository.FeedbackWriter) *SubmitFeedbackUsecase {
	return &SubmitFeedbackUsecase{writer: writer}
}

func (u *SubmitFeedbackUsecase) Execute(ctx context.Context, request dto.SubmitFeedbackRequest) (dto.SubmitFeedbackResponse, error) {
	if u.writer == nil {
		return dto.SubmitFeedbackResponse{}, ValidationError("feedback writer is required")
	}
	if strings.TrimSpace(request.UserID) == "" {
		return dto.SubmitFeedbackResponse{}, ValidationError("user_id is required")
	}
	if !isUUID(request.UserID) {
		return dto.SubmitFeedbackResponse{}, ValidationError("user_id must be a UUID")
	}
	if request.ClientFeedbackID != nil && !isUUID(*request.ClientFeedbackID) {
		return dto.SubmitFeedbackResponse{}, ValidationError("client_feedback_id must be a UUID")
	}
	if err := validateFeedbackPayload(request.Payload); err != nil {
		return dto.SubmitFeedbackResponse{}, err
	}
	if len(request.Images) > maxFeedbackImages {
		return dto.SubmitFeedbackResponse{}, ValidationError("images must contain at most 5 files")
	}
	images := make([]model.FeedbackImage, 0, len(request.Images))
	for index, image := range request.Images {
		if err := validateFeedbackImage(index, image); err != nil {
			return dto.SubmitFeedbackResponse{}, err
		}
		images = append(images, model.FeedbackImage{
			SortOrder:   image.SortOrder,
			ContentType: image.ContentType,
			SizeBytes:   image.SizeBytes,
			SHA256:      image.SHA256,
			Width:       image.Width,
			Height:      image.Height,
			Data:        image.Data,
		})
	}
	result, err := u.writer.SubmitFeedback(ctx, model.FeedbackSubmission{
		UserID:           request.UserID,
		ClientFeedbackID: request.ClientFeedbackID,
		Payload:          request.Payload,
		Images:           images,
	})
	if err != nil {
		return dto.SubmitFeedbackResponse{}, err
	}
	return dto.SubmitFeedbackResponse{
		FeedbackID: result.FeedbackID,
		Accepted:   true,
		ImageCount: result.ImageCount,
		CreatedAt:  result.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}, nil
}

func validateFeedbackPayload(payload []byte) error {
	if len(payload) == 0 {
		return ValidationError("payload is required")
	}
	var object map[string]any
	if err := json.Unmarshal(payload, &object); err != nil {
		return ValidationError("payload must be a JSON object")
	}
	if object == nil {
		return ValidationError("payload must be a JSON object")
	}
	return nil
}

func validateFeedbackImage(index int, image dto.FeedbackImageInput) error {
	expectedSortOrder := int32(index + 1)
	if image.SortOrder != expectedSortOrder {
		return ValidationError("images sort_order must match upload order")
	}
	if image.ContentType != "image/jpeg" {
		return ValidationError("images must be JPEG files")
	}
	if image.SizeBytes <= 0 {
		return ValidationError("images size_bytes must be positive")
	}
	if image.Width <= 0 || image.Height <= 0 {
		return ValidationError("images dimensions must be positive")
	}
	if len(image.Data) == 0 {
		return ValidationError("images data is required")
	}
	if image.SizeBytes != int32(len(image.Data)) {
		return ValidationError("images size_bytes must match image data")
	}
	hash := sha256.Sum256(image.Data)
	if image.SHA256 != hex.EncodeToString(hash[:]) {
		return ValidationError("images sha256 must match image data")
	}
	return nil
}

func isUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for index, char := range value {
		switch index {
		case 8, 13, 18, 23:
			if char != '-' {
				return false
			}
		default:
			if !isHex(char) {
				return false
			}
		}
	}
	return true
}

func isHex(char rune) bool {
	return ('0' <= char && char <= '9') ||
		('a' <= char && char <= 'f') ||
		('A' <= char && char <= 'F')
}
