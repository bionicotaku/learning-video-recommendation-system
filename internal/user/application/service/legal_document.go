package service

import (
	"context"
	"strings"

	"learning-video-recommendation-system/internal/user/application/dto"
	"learning-video-recommendation-system/internal/user/application/repository"
	"learning-video-recommendation-system/internal/user/domain/model"
)

type GetLegalDocumentUsecase struct {
	reader repository.LegalDocumentReader
}

func NewGetLegalDocumentUsecase(reader repository.LegalDocumentReader) *GetLegalDocumentUsecase {
	return &GetLegalDocumentUsecase{reader: reader}
}

func (u *GetLegalDocumentUsecase) Execute(ctx context.Context, request dto.GetLegalDocumentRequest) (dto.GetLegalDocumentResponse, error) {
	if u.reader == nil {
		return dto.GetLegalDocumentResponse{}, ValidationError("legal document reader is required")
	}
	documentType := strings.TrimSpace(request.Type)
	if !model.IsLegalDocumentType(documentType) {
		return dto.GetLegalDocumentResponse{}, ValidationError("unsupported legal document type")
	}
	document, found, err := u.reader.GetLegalDocument(ctx, documentType)
	if err != nil {
		return dto.GetLegalDocumentResponse{}, err
	}
	if !found {
		return dto.GetLegalDocumentResponse{}, repository.ErrLegalDocumentNotFound
	}
	return legalDocumentResponse(document), nil
}

func legalDocumentResponse(document model.LegalDocument) dto.GetLegalDocumentResponse {
	var updatedAt *string
	if document.UpdatedAt != nil {
		value := document.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z")
		updatedAt = &value
	}
	return dto.GetLegalDocumentResponse{
		Type:      document.Type,
		Title:     document.Title,
		Markdown:  document.Markdown,
		UpdatedAt: updatedAt,
		Version:   document.Version,
	}
}
