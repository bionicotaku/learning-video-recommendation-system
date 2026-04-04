package usecase

import (
	"context"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/application/command"
	"learning-video-recommendation-system/internal/recommendation/scheduler/application/dto"
	appquery "learning-video-recommendation-system/internal/recommendation/scheduler/application/query"
	apprepo "learning-video-recommendation-system/internal/recommendation/scheduler/application/repository"
	domainservice "learning-video-recommendation-system/internal/recommendation/scheduler/domain/service"
)

type GenerateLearningUnitRecommendationsUseCase struct {
	stateRepo         apprepo.UserUnitStateRepository
	settingsRepo      apprepo.UserSchedulerSettingsRepository
	runRepo           apprepo.SchedulerRunRepository
	backlogCalculator domainservice.BacklogCalculator
	quotaAllocator    domainservice.QuotaAllocator
	reviewScorer      domainservice.ReviewScorer
	newScorer         domainservice.NewScorer
	priorityExtractor domainservice.PriorityZeroExtractor
	assembler         domainservice.RecommendationAssembler
}

func NewGenerateLearningUnitRecommendationsUseCase(
	stateRepo apprepo.UserUnitStateRepository,
	settingsRepo apprepo.UserSchedulerSettingsRepository,
	runRepo apprepo.SchedulerRunRepository,
	backlogCalculator domainservice.BacklogCalculator,
	quotaAllocator domainservice.QuotaAllocator,
	reviewScorer domainservice.ReviewScorer,
	newScorer domainservice.NewScorer,
	priorityExtractor domainservice.PriorityZeroExtractor,
	assembler domainservice.RecommendationAssembler,
) GenerateLearningUnitRecommendationsUseCase {
	return GenerateLearningUnitRecommendationsUseCase{
		stateRepo:         stateRepo,
		settingsRepo:      settingsRepo,
		runRepo:           runRepo,
		backlogCalculator: backlogCalculator,
		quotaAllocator:    quotaAllocator,
		reviewScorer:      reviewScorer,
		newScorer:         newScorer,
		priorityExtractor: priorityExtractor,
		assembler:         assembler,
	}
}

func (uc GenerateLearningUnitRecommendationsUseCase) Execute(ctx context.Context, cmd command.GenerateRecommendationsCommand) (dto.GenerateRecommendationsResult, error) {
	settings, err := uc.settingsRepo.GetOrDefault(ctx, cmd.UserID)
	if err != nil {
		return dto.GenerateRecommendationsResult{}, err
	}

	now := cmd.Now
	if now.IsZero() {
		now = time.Now()
	}

	requestedLimit := cmd.RequestedLimit
	if requestedLimit <= 0 {
		requestedLimit = settings.SessionDefaultLimit
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
	quotas := uc.quotaAllocator.Allocate(reviewBacklog, requestedLimit, *settings)

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

	if persistSnapshot(cmd.RequestContext) && uc.runRepo != nil {
		if err := uc.runRepo.SaveRun(ctx, batch); err != nil {
			return dto.GenerateRecommendationsResult{}, err
		}
		if err := uc.runRepo.SaveRunItems(ctx, batch); err != nil {
			return dto.GenerateRecommendationsResult{}, err
		}
	}

	return dto.GenerateRecommendationsResult{Batch: batch}, nil
}

func persistSnapshot(requestContext map[string]any) bool {
	if requestContext == nil {
		return false
	}

	value, ok := requestContext["persist_snapshot"]
	if !ok {
		return false
	}

	flag, ok := value.(bool)
	return ok && flag
}
