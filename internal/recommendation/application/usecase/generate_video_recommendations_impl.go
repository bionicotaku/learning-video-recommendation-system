package usecase

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sort"

	"learning-video-recommendation-system/internal/recommendation/application/dto"
	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	appservice "learning-video-recommendation-system/internal/recommendation/application/service"
	domainaggregator "learning-video-recommendation-system/internal/recommendation/domain/aggregator"
	domainassembler "learning-video-recommendation-system/internal/recommendation/domain/assembler"
	domaincandidate "learning-video-recommendation-system/internal/recommendation/domain/candidate"
	domainexplain "learning-video-recommendation-system/internal/recommendation/domain/explain"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	domainplanner "learning-video-recommendation-system/internal/recommendation/domain/planner"
	"learning-video-recommendation-system/internal/recommendation/domain/policy"
	domainranking "learning-video-recommendation-system/internal/recommendation/domain/ranking"
	domainresolver "learning-video-recommendation-system/internal/recommendation/domain/resolver"
	domainselector "learning-video-recommendation-system/internal/recommendation/domain/selector"
)

type GenerateVideoRecommendationsService struct {
	assembler          domainassembler.ContextAssembler
	planner            domainplanner.DemandPlanner
	candidateGenerator domaincandidate.CandidateGenerator
	resolver           domainresolver.EvidenceResolver
	aggregator         domainaggregator.VideoEvidenceAggregator
	ranker             domainranking.VideoRanker
	selector           domainselector.VideoSelector
	explainer          domainexplain.ExplanationBuilder
	videoServing       apprepo.VideoServingStateRepository
	videoUserStates    apprepo.VideoUserStateReader
	resultWriter       appservice.RecommendationResultWriter
}

var _ GenerateVideoRecommendationsUsecase = (*GenerateVideoRecommendationsService)(nil)

func NewGenerateVideoRecommendationsService(assembler domainassembler.ContextAssembler) *GenerateVideoRecommendationsService {
	return &GenerateVideoRecommendationsService{assembler: assembler}
}

func NewGenerateVideoRecommendationsPipeline(
	assembler domainassembler.ContextAssembler,
	planner domainplanner.DemandPlanner,
	candidateGenerator domaincandidate.CandidateGenerator,
	resolver domainresolver.EvidenceResolver,
	aggregator domainaggregator.VideoEvidenceAggregator,
	ranker domainranking.VideoRanker,
	selector domainselector.VideoSelector,
	explainer domainexplain.ExplanationBuilder,
	videoServing apprepo.VideoServingStateRepository,
	videoUserStates apprepo.VideoUserStateReader,
	resultWriter appservice.RecommendationResultWriter,
) *GenerateVideoRecommendationsService {
	return &GenerateVideoRecommendationsService{
		assembler:          assembler,
		planner:            planner,
		candidateGenerator: candidateGenerator,
		resolver:           resolver,
		aggregator:         aggregator,
		ranker:             ranker,
		selector:           selector,
		explainer:          explainer,
		videoServing:       videoServing,
		videoUserStates:    videoUserStates,
		resultWriter:       resultWriter,
	}
}

func (u *GenerateVideoRecommendationsService) Execute(ctx context.Context, request dto.GenerateVideoRecommendationsRequest) (dto.GenerateVideoRecommendationsResponse, error) {
	contextModel, err := u.assembler.Assemble(ctx, model.RecommendationRequest{
		UserID:               request.UserID,
		TargetVideoCount:     request.TargetVideoCount,
		PreferredDurationSec: request.PreferredDurationSec,
		SessionHint:          request.SessionHint,
		RequestContext:       request.RequestContext,
	})
	if err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	if u.planner == nil || u.candidateGenerator == nil || u.resolver == nil || u.aggregator == nil || u.ranker == nil || u.selector == nil || u.explainer == nil {
		return shellResponse(contextModel)
	}

	demandBundle, err := u.planner.Plan(contextModel)
	if err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	candidates, err := u.candidateGenerator.Generate(contextModel, demandBundle)
	if err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	resolvedEvidence, err := u.resolver.Resolve(contextModel, candidates, demandBundle)
	if err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	videoCandidates, err := u.aggregator.Aggregate(contextModel, resolvedEvidence, demandBundle)
	if err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	contextModel, err = u.enrichVideoStates(ctx, contextModel, videoCandidates)
	if err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	rankedVideos, err := u.ranker.Rank(contextModel, videoCandidates, demandBundle)
	if err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	selectedVideos, err := u.selector.Select(contextModel, rankedVideos, demandBundle)
	if err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	finalItems, err := u.explainer.Build(contextModel, selectedVideos, demandBundle)
	if err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	runID, err := newRunID()
	if err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	selectorMode := selectorModeForDemand(demandBundle)
	response := dto.GenerateVideoRecommendationsResponse{
		RunID:        runID,
		SelectorMode: selectorMode,
		Underfilled:  len(finalItems) < contextModel.Request.TargetVideoCount,
		Videos:       mapFinalItems(finalItems),
	}

	if u.resultWriter != nil {
		run, items, err := buildAuditPayload(runID, contextModel.Request.UserID, contextModel.Request.RequestContext, selectorMode, demandBundle, candidates, selectedVideos, finalItems, response.Underfilled)
		if err != nil {
			return dto.GenerateVideoRecommendationsResponse{}, err
		}
		if err := u.resultWriter.Persist(ctx, run, items, contextModel.Request.UserID, finalItems); err != nil {
			return dto.GenerateVideoRecommendationsResponse{}, err
		}
	}

	return response, nil
}

func shellResponse(contextModel model.RecommendationContext) (dto.GenerateVideoRecommendationsResponse, error) {
	runID, err := newRunID()
	if err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	return dto.GenerateVideoRecommendationsResponse{
		RunID:        runID,
		SelectorMode: "",
		Underfilled:  len(contextModel.ActiveUnitStates) > 0,
		Videos:       []dto.RecommendationVideo{},
	}, nil
}

func newRunID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		buf[0:4],
		buf[4:6],
		buf[6:8],
		buf[8:10],
		buf[10:16],
	), nil
}

func (u *GenerateVideoRecommendationsService) enrichVideoStates(ctx context.Context, contextModel model.RecommendationContext, videos []model.VideoCandidate) (model.RecommendationContext, error) {
	if len(videos) == 0 {
		return contextModel, nil
	}

	videoIDs := uniqueVideoIDs(videos)
	if u.videoServing != nil {
		videoServingStates, err := u.videoServing.ListByUserAndVideoIDs(ctx, contextModel.Request.UserID, videoIDs)
		if err != nil {
			return model.RecommendationContext{}, err
		}
		contextModel.VideoServingStates = videoServingStates
	}
	if u.videoUserStates != nil {
		videoUserStates, err := u.videoUserStates.ListByUserAndVideoIDs(ctx, contextModel.Request.UserID, videoIDs)
		if err != nil {
			return model.RecommendationContext{}, err
		}
		contextModel.VideoUserStates = videoUserStates
	}
	return contextModel, nil
}

func uniqueVideoIDs(videos []model.VideoCandidate) []string {
	seen := make(map[string]struct{}, len(videos))
	result := make([]string, 0, len(videos))
	for _, video := range videos {
		if _, ok := seen[video.VideoID]; ok {
			continue
		}
		seen[video.VideoID] = struct{}{}
		result = append(result, video.VideoID)
	}
	sort.Strings(result)
	return result
}

func selectorModeForDemand(demand model.DemandBundle) string {
	if demand.Flags.ExtremeSparse {
		return string(policy.SelectorModeExtremeSparse)
	}
	if demand.Flags.HardReviewLowSupply {
		return string(policy.SelectorModeLowSupply)
	}
	return string(policy.SelectorModeNormal)
}

func mapFinalItems(items []model.FinalRecommendationItem) []dto.RecommendationVideo {
	result := make([]dto.RecommendationVideo, 0, len(items))
	for _, item := range items {
		result = append(result, dto.RecommendationVideo{
			VideoID:                   item.VideoID,
			Rank:                      item.Rank,
			Score:                     item.Score,
			ReasonCodes:               item.ReasonCodes,
			CoveredUnits:              item.CoveredUnits,
			CoveredHardReviewUnits:    item.CoveredHardReviewUnits,
			CoveredNewNowUnits:        item.CoveredNewNowUnits,
			CoveredSoftReviewUnits:    item.CoveredSoftReviewUnits,
			CoveredNearFutureUnits:    item.CoveredNearFutureUnits,
			BestEvidenceSentenceIndex: item.BestEvidenceSentenceIndex,
			BestEvidenceSpanIndex:     item.BestEvidenceSpanIndex,
			BestEvidenceStartMs:       item.BestEvidenceStartMs,
			BestEvidenceEndMs:         item.BestEvidenceEndMs,
			Explanation:               item.Explanation,
		})
	}
	return result
}

func buildAuditPayload(
	runID string,
	userID string,
	requestContext []byte,
	selectorMode string,
	demand model.DemandBundle,
	candidates []model.VideoUnitCandidate,
	selectedVideos []model.VideoCandidate,
	finalItems []model.FinalRecommendationItem,
	underfilled bool,
) (model.RecommendationRun, []model.RecommendationItem, error) {
	plannerSnapshot, err := json.Marshal(demand)
	if err != nil {
		return model.RecommendationRun{}, nil, err
	}
	laneBudgetSnapshot, err := json.Marshal(demand.LaneBudget)
	if err != nil {
		return model.RecommendationRun{}, nil, err
	}
	candidateSummary, err := json.Marshal(candidateSummary(candidates))
	if err != nil {
		return model.RecommendationRun{}, nil, err
	}

	run := model.RecommendationRun{
		RunID:              runID,
		UserID:             userID,
		RequestContext:     requestContext,
		SessionMode:        demand.SessionMode,
		SelectorMode:       selectorMode,
		PlannerSnapshot:    plannerSnapshot,
		LaneBudgetSnapshot: laneBudgetSnapshot,
		CandidateSummary:   candidateSummary,
		Underfilled:        underfilled,
		ResultCount:        int32(len(finalItems)),
	}

	selectedByVideo := make(map[string]model.VideoCandidate, len(selectedVideos))
	for _, selected := range selectedVideos {
		selectedByVideo[selected.VideoID] = selected
	}

	items := make([]model.RecommendationItem, 0, len(finalItems))
	for _, finalItem := range finalItems {
		selected := selectedByVideo[finalItem.VideoID]
		items = append(items, model.RecommendationItem{
			RunID:                     runID,
			Rank:                      int32(finalItem.Rank),
			VideoID:                   finalItem.VideoID,
			Score:                     finalItem.Score,
			PrimaryLane:               primaryLane(selected.LaneSources),
			DominantBucket:            selected.DominantBucket,
			DominantUnitID:            selected.DominantUnitID,
			ReasonCodes:               finalItem.ReasonCodes,
			CoveredHardReviewCount:    int32(len(finalItem.CoveredHardReviewUnits)),
			CoveredNewNowCount:        int32(len(finalItem.CoveredNewNowUnits)),
			CoveredSoftReviewCount:    int32(len(finalItem.CoveredSoftReviewUnits)),
			CoveredNearFutureCount:    int32(len(finalItem.CoveredNearFutureUnits)),
			BestEvidenceSentenceIndex: finalItem.BestEvidenceSentenceIndex,
			BestEvidenceSpanIndex:     finalItem.BestEvidenceSpanIndex,
			BestEvidenceStartMs:       finalItem.BestEvidenceStartMs,
			BestEvidenceEndMs:         finalItem.BestEvidenceEndMs,
		})
	}

	return run, items, nil
}

func candidateSummary(candidates []model.VideoUnitCandidate) map[string]any {
	laneCounts := make(map[string]int)
	distinctVideos := make(map[string]struct{})
	laneDistinctVideos := make(map[string]map[string]struct{})
	for _, candidate := range candidates {
		laneCounts[candidate.Lane]++
		distinctVideos[candidate.VideoID] = struct{}{}
		if _, ok := laneDistinctVideos[candidate.Lane]; !ok {
			laneDistinctVideos[candidate.Lane] = map[string]struct{}{}
		}
		laneDistinctVideos[candidate.Lane][candidate.VideoID] = struct{}{}
	}

	laneDistinctCounts := make(map[string]int, len(laneDistinctVideos))
	for lane, videos := range laneDistinctVideos {
		laneDistinctCounts[lane] = len(videos)
	}

	return map[string]any{
		"lane_counts":          laneCounts,
		"lane_distinct_videos": laneDistinctCounts,
		"distinct_video_count": len(distinctVideos),
	}
}

func primaryLane(laneSources []string) string {
	if len(laneSources) == 0 {
		return ""
	}
	sort.SliceStable(laneSources, func(i, j int) bool {
		return lanePriority(laneSources[i]) < lanePriority(laneSources[j])
	})
	return laneSources[0]
}

func lanePriority(lane string) int {
	switch lane {
	case string(policy.LaneExactCore):
		return 0
	case string(policy.LaneBundle):
		return 1
	case string(policy.LaneSoftFuture):
		return 2
	default:
		return 3
	}
}
