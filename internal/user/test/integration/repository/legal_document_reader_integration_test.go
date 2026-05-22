//go:build integration

package repository_test

import (
	"context"
	"testing"

	userrepo "learning-video-recommendation-system/internal/user/infrastructure/persistence/repository"
)

func TestRepositoryReadsSeededLegalDocuments(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	repository := userrepo.NewRepository(db.Pool)

	document, found, err := repository.GetLegalDocument(context.Background(), "privacy-policy")
	if err != nil {
		t.Fatalf("GetLegalDocument privacy-policy: %v", err)
	}
	if !found {
		t.Fatalf("privacy-policy not found")
	}
	if document.Type != "privacy-policy" || document.Title != "隐私政策" || document.Markdown == "" {
		t.Fatalf("unexpected privacy-policy document: %+v", document)
	}
	if document.Version == nil || *document.Version != "2026-05-21" {
		t.Fatalf("version = %v, want 2026-05-21", document.Version)
	}
	if document.UpdatedAt == nil || !document.UpdatedAt.Equal(document.UpdatedAt.UTC()) {
		t.Fatalf("updated_at should be non-nil UTC: %v", document.UpdatedAt)
	}

	agreement, found, err := repository.GetLegalDocument(context.Background(), "user-agreement")
	if err != nil {
		t.Fatalf("GetLegalDocument user-agreement: %v", err)
	}
	if !found || agreement.Type != "user-agreement" || agreement.Title != "用户协议" || agreement.Markdown == "" {
		t.Fatalf("unexpected user-agreement document: found=%v document=%+v", found, agreement)
	}
}

func TestRepositoryReturnsNotFoundForMissingLegalDocument(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	repository := userrepo.NewRepository(db.Pool)

	_, found, err := repository.GetLegalDocument(context.Background(), "missing")
	if err != nil {
		t.Fatalf("GetLegalDocument missing: %v", err)
	}
	if found {
		t.Fatalf("missing legal document should not be found")
	}
}
