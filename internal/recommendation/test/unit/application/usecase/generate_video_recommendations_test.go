package usecase_test

import (
	"context"
	"errors"
	"testing"

	"learning-video-recommendation-system/internal/recommendation/application/dto"
	appservice "learning-video-recommendation-system/internal/recommendation/application/service"
	"learning-video-recommendation-system/internal/recommendation/application/usecase"
	domainaggregator "learning-video-recommendation-system/internal/recommendation/domain/aggregator"
	domainassembler "learning-video-recommendation-system/internal/recommendation/domain/assembler"
	domaincandidate "learning-video-recommendation-system/internal/recommendation/domain/candidate"
	domainexplain "learning-video-recommendation-system/internal/recommendation/domain/explain"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	domainplanner "learning-video-recommendation-system/internal/recommendation/domain/planner"
	domainranking "learning-video-recommendation-system/internal/recommendation/domain/ranking"
	domainresolver "learning-video-recommendation-system/internal/recommendation/domain/resolver"
	domainselector "learning-video-recommendation-system/internal/recommendation/domain/selector"
)

func TestNewGenerateVideoRecommendationsPipelineRejectsIncompleteDependencies(t *testing.T) {
	_, err := usecase.NewGenerateVideoRecommendationsPipeline(
		&constructorStubContextAssembler{},
		nil,
		&constructorStubCandidateGenerator{},
		&constructorStubResolver{},
		&constructorStubAggregator{},
		&constructorStubRanker{},
		&constructorStubSelector{},
		&constructorStubExplainer{},
		nil,
		nil,
	)
	if !errors.Is(err, usecase.ErrIncompletePipeline) {
		t.Fatalf("expected ErrIncompletePipeline, got %v", err)
	}
}

func TestGenerateVideoRecommendationsPipelinePropagatesAssemblerError(t *testing.T) {
	expectedErr := errors.New("assemble failed")
	service, err := usecase.NewGenerateVideoRecommendationsPipeline(
		&constructorStubContextAssembler{err: expectedErr},
		&constructorStubPlanner{},
		&constructorStubCandidateGenerator{},
		&constructorStubResolver{},
		&constructorStubAggregator{},
		&constructorStubRanker{},
		&constructorStubSelector{},
		&constructorStubExplainer{},
		&constructorStubVideoStateEnricher{},
		nil,
	)
	if err != nil {
		t.Fatalf("NewGenerateVideoRecommendationsPipeline() error = %v", err)
	}

	_, execErr := service.Execute(context.Background(), dto.GenerateVideoRecommendationsRequest{UserID: "user-1"})
	if !errors.Is(execErr, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, execErr)
	}
}

type constructorStubContextAssembler struct {
	context model.RecommendationContext
	err     error
}

func (s *constructorStubContextAssembler) Assemble(context.Context, model.RecommendationRequest) (model.RecommendationContext, error) {
	return s.context, s.err
}

var _ domainassembler.ContextAssembler = (*constructorStubContextAssembler)(nil)

type constructorStubPlanner struct{ bundle model.DemandBundle }

func (s *constructorStubPlanner) Plan(model.RecommendationContext) (model.DemandBundle, error) {
	return s.bundle, nil
}

var _ domainplanner.DemandPlanner = (*constructorStubPlanner)(nil)

type constructorStubCandidateGenerator struct{ candidates []model.VideoUnitCandidate }

func (s *constructorStubCandidateGenerator) Generate(context.Context, model.RecommendationContext, model.DemandBundle) ([]model.VideoUnitCandidate, error) {
	return s.candidates, nil
}

var _ domaincandidate.CandidateGenerator = (*constructorStubCandidateGenerator)(nil)

type constructorStubResolver struct {
	windows []model.ResolvedEvidenceWindow
}

func (s *constructorStubResolver) Resolve(context.Context, model.RecommendationContext, []model.VideoUnitCandidate, model.DemandBundle) ([]model.ResolvedEvidenceWindow, error) {
	return s.windows, nil
}

var _ domainresolver.EvidenceResolver = (*constructorStubResolver)(nil)

type constructorStubAggregator struct{ videos []model.VideoCandidate }

func (s *constructorStubAggregator) Aggregate(model.RecommendationContext, []model.ResolvedEvidenceWindow, model.DemandBundle) ([]model.VideoCandidate, error) {
	return s.videos, nil
}

var _ domainaggregator.VideoEvidenceAggregator = (*constructorStubAggregator)(nil)

type constructorStubRanker struct{ ranked []model.VideoCandidate }

func (s *constructorStubRanker) Rank(model.RecommendationContext, []model.VideoCandidate, model.DemandBundle) ([]model.VideoCandidate, error) {
	return s.ranked, nil
}

var _ domainranking.VideoRanker = (*constructorStubRanker)(nil)

type constructorStubSelector struct{ selected []model.VideoCandidate }

func (s *constructorStubSelector) Select(model.RecommendationContext, []model.VideoCandidate, model.DemandBundle) ([]model.VideoCandidate, error) {
	return s.selected, nil
}

var _ domainselector.VideoSelector = (*constructorStubSelector)(nil)

type constructorStubExplainer struct {
	items []model.FinalRecommendationItem
}

func (s *constructorStubExplainer) Build(model.RecommendationContext, []model.VideoCandidate, model.DemandBundle) ([]model.FinalRecommendationItem, error) {
	return s.items, nil
}

var _ domainexplain.ExplanationBuilder = (*constructorStubExplainer)(nil)

type constructorStubVideoStateEnricher struct{}

func (s *constructorStubVideoStateEnricher) Enrich(_ context.Context, contextModel model.RecommendationContext, _ []model.VideoCandidate) (model.RecommendationContext, error) {
	return contextModel, nil
}

var _ appservice.VideoStateEnricher = (*constructorStubVideoStateEnricher)(nil)
