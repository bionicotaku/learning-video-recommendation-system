package usecase_test

import (
	"context"
	"errors"
	"testing"

	"learning-video-recommendation-system/internal/recommendation/application/dto"
	"learning-video-recommendation-system/internal/recommendation/application/usecase"
	domainassembler "learning-video-recommendation-system/internal/recommendation/domain/assembler"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

func TestGenerateVideoRecommendationsServiceExecuteReturnsEmptyResponseShell(t *testing.T) {
	service := usecase.NewGenerateVideoRecommendationsService(&stubContextAssembler{
		context: model.RecommendationContext{
			ActiveUnitStates: []model.LearningStateSnapshot{{UserID: "user-1", CoarseUnitID: 101}},
		},
	})

	response, err := service.Execute(context.Background(), dto.GenerateVideoRecommendationsRequest{
		UserID: "user-1",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if response.RunID == "" {
		t.Fatal("expected generated run id")
	}
	if !response.Underfilled {
		t.Fatal("expected underfilled when no videos are selected")
	}
	if len(response.Videos) != 0 {
		t.Fatalf("expected empty video list, got %d", len(response.Videos))
	}
}

func TestGenerateVideoRecommendationsServiceExecutePropagatesAssemblerError(t *testing.T) {
	expectedErr := errors.New("assemble failed")
	service := usecase.NewGenerateVideoRecommendationsService(&stubContextAssembler{err: expectedErr})

	_, err := service.Execute(context.Background(), dto.GenerateVideoRecommendationsRequest{UserID: "user-1"})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

type stubContextAssembler struct {
	context model.RecommendationContext
	err     error
}

func (s *stubContextAssembler) Assemble(context.Context, model.RecommendationRequest) (model.RecommendationContext, error) {
	return s.context, s.err
}

var _ domainassembler.ContextAssembler = (*stubContextAssembler)(nil)
