package usecase

import (
	"context"
	"time"

	"learning-video-recommendation-system/internal/recommendation/application/command"
	"learning-video-recommendation-system/internal/recommendation/application/dto"
	appquery "learning-video-recommendation-system/internal/recommendation/application/query"
	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	domainservice "learning-video-recommendation-system/internal/recommendation/domain/service"
)

type GenerateLearningUnitRecommendationsUseCase struct {
	txManager         apprepo.TxManager
	stateRepo         apprepo.UserUnitStateReadRepository
	servingStateRepo  apprepo.UserUnitServingStateRepository
	runRepo           apprepo.SchedulerRunRepository
	backlogCalculator domainservice.BacklogCalculator
	quotaAllocator    domainservice.QuotaAllocator
	reviewScorer      domainservice.ReviewScorer
	newScorer         domainservice.NewScorer
	priorityExtractor domainservice.PriorityZeroExtractor
	assembler         domainservice.RecommendationAssembler
	defaults          model.RecommendationDefaults
}

func NewGenerateLearningUnitRecommendationsUseCase(
	txManager apprepo.TxManager,
	stateRepo apprepo.UserUnitStateReadRepository,
	servingStateRepo apprepo.UserUnitServingStateRepository,
	runRepo apprepo.SchedulerRunRepository,
	backlogCalculator domainservice.BacklogCalculator,
	quotaAllocator domainservice.QuotaAllocator,
	reviewScorer domainservice.ReviewScorer,
	newScorer domainservice.NewScorer,
	priorityExtractor domainservice.PriorityZeroExtractor,
	assembler domainservice.RecommendationAssembler,
) GenerateLearningUnitRecommendationsUseCase {
	return GenerateLearningUnitRecommendationsUseCase{
		txManager:         txManager,
		stateRepo:         stateRepo,
		servingStateRepo:  servingStateRepo,
		runRepo:           runRepo,
		backlogCalculator: backlogCalculator,
		quotaAllocator:    quotaAllocator,
		reviewScorer:      reviewScorer,
		newScorer:         newScorer,
		priorityExtractor: priorityExtractor,
		assembler:         assembler,
		defaults:          model.DefaultRecommendationDefaults(),
	}
}

func (uc GenerateLearningUnitRecommendationsUseCase) Execute(ctx context.Context, cmd command.GenerateRecommendationsCommand) (dto.GenerateRecommendationsResult, error) {
	now := cmd.Now
	if now.IsZero() {
		now = time.Now()
	}

	requestedLimit := cmd.RequestedLimit
	if requestedLimit <= 0 {
		requestedLimit = uc.defaults.SessionDefaultLimit
	}

	reviewCandidates, err := uc.stateRepo.FindDueReviewCandidates(ctx, cmd.UserID, now)
	if err != nil {
		return dto.GenerateRecommendationsResult{}, err
	}
	newCandidates, err := uc.stateRepo.FindNewCandidates(ctx, cmd.UserID)
	if err != nil {
		return dto.GenerateRecommendationsResult{}, err
	}

	reviewBacklog := uc.backlogCalculator.Compute(len(reviewCandidates))
	quotas := uc.quotaAllocator.Allocate(reviewBacklog, requestedLimit, uc.defaults)

	scoredReviews := make([]appquery.ScoredReviewCandidate, 0, len(reviewCandidates))
	for _, candidate := range reviewCandidates {
		scoredReviews = append(scoredReviews, uc.reviewScorer.Score(candidate, now))
	}

	scoredNews := make([]appquery.ScoredNewCandidate, 0, len(newCandidates))
	for _, candidate := range newCandidates {
		scoredNews = append(scoredNews, uc.newScorer.Score(candidate, now))
	}

	priorityZero := uc.priorityExtractor.Extract(scoredReviews)
	batch := uc.assembler.Assemble(cmd.UserID, now, priorityZero, scoredReviews, scoredNews, quotas)
	batch.DueReviewCount = reviewBacklog

	err = uc.txManager.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.runRepo.SaveRun(ctx, batch); err != nil {
			return err
		}
		if err := uc.runRepo.SaveRunItems(ctx, batch); err != nil {
			return err
		}
		if err := uc.servingStateRepo.TouchRecommendedAt(ctx, batch.UserID, batch.RunID, coarseUnitIDs(batch), batch.GeneratedAt); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return dto.GenerateRecommendationsResult{}, err
	}

	return dto.GenerateRecommendationsResult{Batch: batch}, nil
}

func coarseUnitIDs(batch model.RecommendationBatch) []int64 {
	ids := make([]int64, 0, len(batch.Items))
	for _, item := range batch.Items {
		ids = append(ids, item.CoarseUnitID)
	}

	return ids
}
