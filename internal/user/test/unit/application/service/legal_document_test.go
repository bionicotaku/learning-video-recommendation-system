package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	userdto "learning-video-recommendation-system/internal/user/application/dto"
	userrepo "learning-video-recommendation-system/internal/user/application/repository"
	userservice "learning-video-recommendation-system/internal/user/application/service"
	"learning-video-recommendation-system/internal/user/domain/model"
)

func TestGetLegalDocumentReturnsDocument(t *testing.T) {
	updatedAt := time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC)
	reader := &fakeLegalDocumentReader{
		document: model.LegalDocument{
			Type:      "privacy-policy",
			Title:     "隐私政策",
			Markdown:  "# 隐私政策\n",
			Version:   strPtr("2026-05-21"),
			UpdatedAt: &updatedAt,
		},
		found: true,
	}
	usecase := userservice.NewGetLegalDocumentUsecase(reader)

	result, err := usecase.Execute(context.Background(), userdto.GetLegalDocumentRequest{
		Type: "privacy-policy",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if result.Type != "privacy-policy" || result.Title != "隐私政策" || result.Markdown != "# 隐私政策\n" {
		t.Fatalf("unexpected document: %+v", result)
	}
	if result.Version == nil || *result.Version != "2026-05-21" {
		t.Fatalf("version = %v, want 2026-05-21", result.Version)
	}
	if result.UpdatedAt == nil || *result.UpdatedAt != "2026-05-21T00:00:00Z" {
		t.Fatalf("updated_at = %v, want UTC RFC3339", result.UpdatedAt)
	}
}

func TestGetLegalDocumentRejectsUnsupportedType(t *testing.T) {
	reader := &fakeLegalDocumentReader{}
	usecase := userservice.NewGetLegalDocumentUsecase(reader)

	_, err := usecase.Execute(context.Background(), userdto.GetLegalDocumentRequest{
		Type: "terms",
	})

	if !userservice.IsValidationError(err) {
		t.Fatalf("err = %v, want validation error", err)
	}
	if reader.called {
		t.Fatalf("reader should not be called")
	}
}

func TestGetLegalDocumentReturnsConfigurationErrorWhenSupportedTypeMissing(t *testing.T) {
	reader := &fakeLegalDocumentReader{found: false}
	usecase := userservice.NewGetLegalDocumentUsecase(reader)

	_, err := usecase.Execute(context.Background(), userdto.GetLegalDocumentRequest{
		Type: "user-agreement",
	})

	if !errors.Is(err, userrepo.ErrLegalDocumentNotFound) {
		t.Fatalf("err = %v, want ErrLegalDocumentNotFound", err)
	}
}

type fakeLegalDocumentReader struct {
	called   bool
	document model.LegalDocument
	found    bool
	err      error
}

func (f *fakeLegalDocumentReader) GetLegalDocument(ctx context.Context, documentType string) (model.LegalDocument, bool, error) {
	if ctx == nil {
		return model.LegalDocument{}, false, errors.New("ctx is required")
	}
	f.called = true
	if f.err != nil {
		return model.LegalDocument{}, false, f.err
	}
	return f.document, f.found, nil
}

func strPtr(value string) *string {
	return &value
}
