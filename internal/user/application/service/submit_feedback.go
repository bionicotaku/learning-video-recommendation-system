package service

import (
	"context"
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
	if strings.TrimSpace(image.SHA256) == "" {
		return ValidationError("images sha256 is required")
	}
	if image.Width <= 0 || image.Height <= 0 {
		return ValidationError("images dimensions must be positive")
	}
	if len(image.Data) == 0 {
		return ValidationError("images data is required")
	}
	return nil
}
