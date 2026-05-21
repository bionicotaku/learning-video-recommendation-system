package usecase

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"learning-video-recommendation-system/internal/recommendation/application/dto"
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
	videoFillService   appservice.VideoFillService
	explainer          domainexplain.ExplanationBuilder
	videoStateEnricher appservice.VideoStateEnricher
	resultWriter       appservice.RecommendationResultWriter
}

var _ GenerateVideoRecommendationsUsecase = (*GenerateVideoRecommendationsService)(nil)

var ErrIncompletePipeline = errors.New("recommendation pipeline requires all dependencies")

func NewGenerateVideoRecommendationsPipeline(
	assembler domainassembler.ContextAssembler,
	planner domainplanner.DemandPlanner,
	candidateGenerator domaincandidate.CandidateGenerator,
	resolver domainresolver.EvidenceResolver,
	aggregator domainaggregator.VideoEvidenceAggregator,
	ranker domainranking.VideoRanker,
	selector domainselector.VideoSelector,
	videoFillService appservice.VideoFillService,
	explainer domainexplain.ExplanationBuilder,
	videoStateEnricher appservice.VideoStateEnricher,
	resultWriter appservice.RecommendationResultWriter,
) (*GenerateVideoRecommendationsService, error) {
	if assembler == nil || planner == nil || candidateGenerator == nil || resolver == nil || aggregator == nil || ranker == nil || selector == nil || videoFillService == nil || explainer == nil || videoStateEnricher == nil {
		return nil, ErrIncompletePipeline
	}

	return &GenerateVideoRecommendationsService{
		assembler:          assembler,
		planner:            planner,
		candidateGenerator: candidateGenerator,
		resolver:           resolver,
		aggregator:         aggregator,
		ranker:             ranker,
		selector:           selector,
		videoFillService:   videoFillService,
		explainer:          explainer,
		videoStateEnricher: videoStateEnricher,
		resultWriter:       resultWriter,
	}, nil
}

func (u *GenerateVideoRecommendationsService) Execute(ctx context.Context, request dto.GenerateVideoRecommendationsRequest) (dto.GenerateVideoRecommendationsResponse, error) {
	timer := NewPipelineTimer()
	var contextModel model.RecommendationContext
	if err := timer.Observe("context_assemble", func() error {
		var err error
		contextModel, err = u.assembler.Assemble(ctx, model.RecommendationRequest{
			UserID:           request.UserID,
			TargetVideoCount: request.TargetVideoCount,
			RequestContext:   request.RequestContext,
		})
		return err
	}); err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	var demandBundle model.DemandBundle
	if err := timer.Observe("plan", func() error {
		var err error
		demandBundle, err = u.planner.Plan(contextModel)
		return err
	}); err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	var candidates []model.VideoUnitCandidate
	if err := timer.Observe("candidate_generate", func() error {
		var err error
		candidates, err = u.candidateGenerator.Generate(ctx, contextModel, demandBundle)
		return err
	}); err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	var resolvedEvidence []model.ResolvedEvidenceWindow
	if err := timer.Observe("evidence_resolve", func() error {
		var err error
		resolvedEvidence, err = u.resolver.Resolve(ctx, contextModel, candidates, demandBundle)
		return err
	}); err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	var videoCandidates []model.VideoCandidate
	if err := timer.Observe("aggregate", func() error {
		var err error
		videoCandidates, err = u.aggregator.Aggregate(contextModel, resolvedEvidence, demandBundle)
		return err
	}); err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}
	contextModel.RecallScope.AggregatedVideoCandidateCount = len(videoCandidates)
	contextModel.RecallScope.VideoStateLookupCount = len(uniqueCandidateVideoIDs(videoCandidates))

	if err := timer.Observe("video_state_enrich", func() error {
		var err error
		contextModel, err = u.videoStateEnricher.Enrich(ctx, contextModel, videoCandidates)
		return err
	}); err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	var rankedVideos []model.VideoCandidate
	if err := timer.Observe("rank", func() error {
		var err error
		rankedVideos, err = u.ranker.Rank(contextModel, videoCandidates, demandBundle)
		return err
	}); err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	var selectedVideos []model.VideoCandidate
	if err := timer.Observe("select", func() error {
		var err error
		selectedVideos, err = u.selector.Select(contextModel, rankedVideos, demandBundle)
		return err
	}); err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	targetCount := targetVideoCount(contextModel.Request, demandBundle)
	if err := timer.Observe("fill", func() error {
		var err error
		selectedVideos, err = u.videoFillService.Fill(ctx, contextModel, selectedVideos, targetCount)
		return err
	}); err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	var finalItems []model.FinalRecommendationItem
	if err := timer.Observe("final_item_build", func() error {
		var err error
		finalItems, err = u.explainer.Build(contextModel, selectedVideos, demandBundle)
		return err
	}); err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	runID, err := newRunID()
	if err != nil {
		return dto.GenerateVideoRecommendationsResponse{}, err
	}

	underfilled := len(finalItems) < targetCount
	demandBundle.Flags.ExtremeSparse = hasDemand(demandBundle) && underfilled
	selectorMode := selectorModeForDemand(demandBundle)
	response := dto.GenerateVideoRecommendationsResponse{
		RunID: runID,
		Items: mapFinalItems(finalItems),
	}

	if u.resultWriter != nil {
		run, items, err := buildAuditPayload(contextModel, runID, contextModel.Request.UserID, contextModel.Request.RequestContext, selectorMode, demandBundle, candidates, selectedVideos, finalItems, underfilled, timer.Snapshot())
		if err != nil {
			return dto.GenerateVideoRecommendationsResponse{}, err
		}
		if err := u.resultWriter.Persist(ctx, run, items, contextModel.Request.UserID, finalItems); err != nil {
			return dto.GenerateVideoRecommendationsResponse{}, err
		}
	}

	return response, nil
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

func selectorModeForDemand(demand model.DemandBundle) string {
	if demand.Flags.ExtremeSparse {
		return string(policy.SelectorModeExtremeSparse)
	}
	if demand.Flags.HardReviewLowSupply {
		return string(policy.SelectorModeLowSupply)
	}
	return string(policy.SelectorModeNormal)
}

func targetVideoCount(request model.RecommendationRequest, demand model.DemandBundle) int {
	if request.TargetVideoCount > 0 {
		return request.TargetVideoCount
	}
	if demand.TargetVideoCount > 0 {
		return demand.TargetVideoCount
	}
	return 8
}

func hasDemand(demand model.DemandBundle) bool {
	return len(demand.HardReview)+len(demand.NewNow)+len(demand.SoftReview)+len(demand.NearFuture) > 0
}

func mapFinalItems(items []model.FinalRecommendationItem) []dto.RecommendationPlanItem {
	result := make([]dto.RecommendationPlanItem, 0, len(items))
	for _, item := range items {
		result = append(result, dto.RecommendationPlanItem{
			VideoID:       item.VideoID,
			DurationMs:    item.DurationMs,
			LearningUnits: mapLearningUnits(item.LearningUnits),
		})
	}
	return result
}

func mapLearningUnits(units []model.ExpectedLearningUnit) []dto.ExpectedLearningUnit {
	result := make([]dto.ExpectedLearningUnit, 0, len(units))
	for _, unit := range units {
		var evidence *dto.LearningUnitEvidence
		if unit.Evidence != nil {
			evidence = &dto.LearningUnitEvidence{
				SentenceIndex: unit.Evidence.SentenceIndex,
				SpanIndex:     unit.Evidence.SpanIndex,
				StartMs:       unit.Evidence.StartMs,
				EndMs:         unit.Evidence.EndMs,
			}
		}
		result = append(result, dto.ExpectedLearningUnit{
			CoarseUnitID: unit.CoarseUnitID,
			Role:         string(unit.Role),
			IsPrimary:    unit.IsPrimary,
			Evidence:     evidence,
		})
	}
	return result
}

func buildAuditPayload(
	contextModel model.RecommendationContext,
	runID string,
	userID string,
	requestContext []byte,
	selectorMode string,
	demand model.DemandBundle,
	candidates []model.VideoUnitCandidate,
	selectedVideos []model.VideoCandidate,
	finalItems []model.FinalRecommendationItem,
	underfilled bool,
	pipelineTimingMs map[string]int64,
) (model.RecommendationRun, []model.RecommendationItem, error) {
	plannerSnapshot, err := json.Marshal(demand)
	if err != nil {
		return model.RecommendationRun{}, nil, err
	}
	laneBudgetSnapshot, err := json.Marshal(demand.LaneBudget)
	if err != nil {
		return model.RecommendationRun{}, nil, err
	}
	candidateSummary, err := json.Marshal(candidateSummary(contextModel, candidates, selectedVideos, pipelineTimingMs))
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
	for index, finalItem := range finalItems {
		selected := selectedByVideo[finalItem.VideoID]
		items = append(items, model.RecommendationItem{
			RunID:          runID,
			Rank:           int32(index + 1),
			VideoID:        finalItem.VideoID,
			Score:          finalItem.Score,
			PrimaryLane:    primaryLane(selected.LaneSources),
			DominantRole:   selected.DominantRole,
			DominantUnitID: selected.DominantUnitID,
			ReasonCodes:    finalItem.ReasonCodes,
			LearningUnits:  append([]model.ExpectedLearningUnit(nil), finalItem.LearningUnits...),
		})
	}

	return run, items, nil
}

func candidateSummary(contextModel model.RecommendationContext, candidates []model.VideoUnitCandidate, selectedVideos []model.VideoCandidate, pipelineTimingMs map[string]int64) map[string]any {
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
	learningSelectedCount := 0
	masteredFillCount := 0
	popularFillCount := 0
	for _, selected := range selectedVideos {
		switch primaryLane(selected.LaneSources) {
		case string(policy.LaneMasteredTargetFill):
			masteredFillCount++
		case string(policy.LanePopularFill):
			popularFillCount++
		default:
			learningSelectedCount++
		}
	}

	return map[string]any{
		"lane_counts":                        laneCounts,
		"lane_distinct_videos":               laneDistinctCounts,
		"distinct_video_count":               len(distinctVideos),
		"learning_selected_count":            learningSelectedCount,
		"mastered_target_fill_count":         masteredFillCount,
		"popular_fill_count":                 popularFillCount,
		"fill_triggered":                     masteredFillCount+popularFillCount > 0,
		"active_target_unit_count":           contextModel.RecallScope.ActiveTargetUnitCount,
		"queue_rebuilt":                      contextModel.RecallScope.QueueRebuilt,
		"queue_candidate_count":              contextModel.RecallScope.QueueCandidateCount,
		"planner_scope_unit_count":           contextModel.RecallScope.PlannerScopeUnitCount,
		"planner_scope_unit_count_by_bucket": contextModel.RecallScope.PlannerScopeUnitCountByBucket,
		"no_supply_scope_unit_count":         contextModel.RecallScope.NoSupplyScopeUnitCount,
		"recall_fetch_unit_count":            contextModel.RecallScope.RecallFetchUnitCount,
		"per_unit_recall_limit":              contextModel.RecallScope.PerUnitRecallLimit,
		"max_possible_recall_rows":           contextModel.RecallScope.MaxPossibleRecallRows,
		"actual_recall_row_count":            contextModel.RecallScope.ActualRecallRowCount,
		"aggregated_video_candidate_count":   contextModel.RecallScope.AggregatedVideoCandidateCount,
		"video_state_lookup_count":           contextModel.RecallScope.VideoStateLookupCount,
		"pipeline_timing_ms":                 pipelineTimingMs,
	}
}

func uniqueCandidateVideoIDs(candidates []model.VideoCandidate) []string {
	seen := make(map[string]struct{}, len(candidates))
	result := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if _, ok := seen[candidate.VideoID]; ok {
			continue
		}
		seen[candidate.VideoID] = struct{}{}
		result = append(result, candidate.VideoID)
	}
	return result
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
	case string(policy.LaneQualityFallback):
		return 3
	case string(policy.LaneMasteredTargetFill):
		return 4
	case string(policy.LanePopularFill):
		return 5
	default:
		return 6
	}
}
