package main

import (
	recommendationservice "learning-video-recommendation-system/internal/recommendation/application/service"
	recommendationusecase "learning-video-recommendation-system/internal/recommendation/application/usecase"
	recommendationaggregator "learning-video-recommendation-system/internal/recommendation/domain/aggregator"
	recommendationexplain "learning-video-recommendation-system/internal/recommendation/domain/explain"
	recommendationplanner "learning-video-recommendation-system/internal/recommendation/domain/planner"
	recommendationranking "learning-video-recommendation-system/internal/recommendation/domain/ranking"
	recommendationselector "learning-video-recommendation-system/internal/recommendation/domain/selector"
	recommendationrepo "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/repository"
	recommendationtx "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/tx"

	"github.com/jackc/pgx/v5/pgxpool"
)

func buildRecommendationUsecase(pool *pgxpool.Pool) (*recommendationusecase.GenerateVideoRecommendationsService, error) {
	unitServing := recommendationrepo.NewUnitServingStateRepository(pool)
	videoServing := recommendationrepo.NewVideoServingStateRepository(pool)
	recommendable := recommendationrepo.NewRecommendableVideoUnitReader(pool)

	return recommendationusecase.NewGenerateVideoRecommendationsPipeline(
		recommendationservice.NewDefaultContextAssembler(
			recommendationrepo.NewLearningStateReader(pool),
			recommendationrepo.NewUnitInventoryReader(pool),
			unitServing,
			recommendationservice.NewRecallQueueService(recommendationrepo.NewRecallQueueRepository(pool)),
			recommendable,
		),
		recommendationplanner.NewDefaultDemandPlanner(),
		recommendationservice.NewDefaultCandidateGenerator(recommendable),
		recommendationservice.NewDefaultEvidenceResolver(),
		recommendationaggregator.NewDefaultVideoEvidenceAggregator(),
		recommendationranking.NewDefaultVideoRanker(),
		recommendationselector.NewDefaultVideoSelector(),
		recommendationservice.NewDefaultVideoFillService(recommendationrepo.NewVideoFillCandidateReader(pool)),
		recommendationexplain.NewDefaultExplanationBuilder(),
		recommendationservice.NewDefaultVideoStateEnricher(
			videoServing,
			recommendationrepo.NewVideoUserStateReader(pool),
		),
		recommendationservice.NewDefaultRecommendationResultWriter(
			recommendationtx.NewManager(pool),
			recommendationservice.NewDefaultAuditWriter(recommendationrepo.NewRecommendationAuditRepository(pool)),
			recommendationservice.NewDefaultServingStateManager(unitServing, videoServing),
		),
	)
}
