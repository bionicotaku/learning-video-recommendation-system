//go:build e2e

package e2e

import (
	"context"
	"errors"
	"testing"
	"time"

	learningdto "learning-video-recommendation-system/internal/learningengine/application/dto"
	recommendationservice "learning-video-recommendation-system/internal/recommendation/application/service"
	recommendationusecase "learning-video-recommendation-system/internal/recommendation/application/usecase"
	recommendationaggregator "learning-video-recommendation-system/internal/recommendation/domain/aggregator"
	recommendationexplain "learning-video-recommendation-system/internal/recommendation/domain/explain"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	recommendationplanner "learning-video-recommendation-system/internal/recommendation/domain/planner"
	recommendationranking "learning-video-recommendation-system/internal/recommendation/domain/ranking"
	recommendationselector "learning-video-recommendation-system/internal/recommendation/domain/selector"
	recommendationrepo "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/repository"
	recommendationtx "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/tx"
	"learning-video-recommendation-system/internal/test/e2e/testutil"
)

func TestE2E_RecommendationWriteSideRunMetadataAndRanksStayConsistent(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecase()

	userID := h.NewUserID()
	hardUnit := h.NewUnitID()
	newUnit := h.NewUnitID()
	softUnit := h.NewUnitID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, hardUnit, newUnit, softUnit)

	h.SeedCatalogVideo(t, strongSupplyVideo(h.NewVideoID(), hardUnit, 1_000, 2_200, 0, "audit-hard", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(h.NewVideoID(), newUnit, 3_000, 4_100, 2, "audit-new", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(h.NewVideoID(), softUnit, 5_000, 6_200, 4, "audit-soft", 90_000))
	h.RefreshRecommendationViews(t)

	testutil.MustEnsureTarget(t, learning, userID,
		targetSpec(hardUnit, 0.95, "hard"),
		targetSpec(newUnit, 0.90, "new"),
		targetSpec(softUnit, 0.75, "soft"),
	)

	now := time.Now().UTC()
	q4 := int16(4)
	mustRecordEvents(t, learning, userID,
		learningdto.LearningEventInput{CoarseUnitID: hardUnit, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: mustTimeAdd(now, -48*time.Hour)},
		learningdto.LearningEventInput{CoarseUnitID: softUnit, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: mustTimeAdd(now, -12*time.Hour)},
	)

	response := mustRecommendN(t, recommendation, userID, 3)
	run := h.LoadRecommendationRun(t, response.RunID)
	if run.SelectorMode != response.SelectorMode {
		t.Fatalf("run selector_mode = %q, want %q", run.SelectorMode, response.SelectorMode)
	}
	if run.Underfilled != response.Underfilled {
		t.Fatalf("run underfilled = %v, want %v", run.Underfilled, response.Underfilled)
	}
	if int(run.ResultCount) != len(response.Videos) {
		t.Fatalf("run result_count = %d, want %d", run.ResultCount, len(response.Videos))
	}

	items := h.LoadRecommendationItems(t, response.RunID)
	assertContiguousRanks(t, items)
}

func TestE2E_RecommendationWriteSideFailureRollsBackAuditAndServing(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()

	userID := h.NewUserID()
	unitID := h.NewUnitID()
	videoID := h.NewVideoID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, unitID)
	h.SeedCatalogVideo(t, strongSupplyVideo(videoID, unitID, 1_000, 2_200, 0, "rollback", 90_000))
	h.RefreshRecommendationViews(t)
	testutil.MustEnsureTarget(t, learning, userID, targetSpec(unitID, 0.95, "rollback"))

	failing := newFailingRecommendationUsecase(h)
	beforeRuns := h.CountRecommendationRuns(t, userID)

	response, err := failing.Execute(context.Background(), recommendRequest(userID, 1))
	if err == nil {
		t.Fatalf("expected write-side error, got response %+v", response)
	}
	if !errors.Is(err, errForcedAuditFailure) {
		t.Fatalf("error = %v, want %v", err, errForcedAuditFailure)
	}
	if got := h.CountRecommendationRuns(t, userID); got != beforeRuns {
		t.Fatalf("run count after rollback = %d, want %d", got, beforeRuns)
	}
	if got := h.LoadUnitServingCountOrZero(t, userID, unitID); got != 0 {
		t.Fatalf("unit served_count after rollback = %d, want 0", got)
	}
	if got := h.LoadVideoServingCountOrZero(t, userID, videoID); got != 0 {
		t.Fatalf("video served_count after rollback = %d, want 0", got)
	}
}

func TestE2E_ReplayKeepsRecommendationOwnStateAccumulating(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecase()

	userID := h.NewUserID()
	unitID := h.NewUnitID()
	videoID := h.NewVideoID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, unitID)
	h.SeedCatalogVideo(t, strongSupplyVideo(videoID, unitID, 1_000, 2_200, 0, "replay-serving", 90_000))
	h.RefreshRecommendationViews(t)
	testutil.MustEnsureTarget(t, learning, userID, targetSpec(unitID, 0.95, "replay"))

	first := mustRecommendN(t, recommendation, userID, 1)
	assertContainsVideo(t, first.Videos, videoID)
	mustReplay(t, learning, userID)
	second := mustRecommendN(t, recommendation, userID, 1)
	assertContainsVideo(t, second.Videos, videoID)

	if got := h.CountRecommendationRuns(t, userID); got != 2 {
		t.Fatalf("run count = %d, want 2", got)
	}
	if got := h.LoadUnitServingCount(t, userID, unitID); got != 2 {
		t.Fatalf("unit served_count = %d, want 2", got)
	}
	if got := h.LoadVideoServingCount(t, userID, videoID); got != 2 {
		t.Fatalf("video served_count = %d, want 2", got)
	}
}

var errForcedAuditFailure = errors.New("forced audit failure")

type failingAuditWriter struct {
	inner recommendationservice.AuditWriter
}

func (w failingAuditWriter) Write(ctx context.Context, run model.RecommendationRun, items []model.RecommendationItem) error {
	if err := w.inner.Write(ctx, run, items); err != nil {
		return err
	}
	return errForcedAuditFailure
}

func newFailingRecommendationUsecase(h *testutil.Harness) recommendationusecase.GenerateVideoRecommendationsUsecase {
	learningStates := recommendationrepo.NewLearningStateReader(h.Pool)
	inventory := recommendationrepo.NewUnitInventoryReader(h.Pool)
	unitServing := recommendationrepo.NewUnitServingStateRepository(h.Pool)
	videoServing := recommendationrepo.NewVideoServingStateRepository(h.Pool)
	videoUserState := recommendationrepo.NewVideoUserStateReader(h.Pool)
	recommendable := recommendationrepo.NewRecommendableVideoUnitReader(h.Pool)
	semanticSpans := recommendationrepo.NewSemanticSpanReader(h.Pool)
	transcriptSentences := recommendationrepo.NewTranscriptSentenceReader(h.Pool)
	auditRepo := recommendationrepo.NewRecommendationAuditRepository(h.Pool)

	assembler := recommendationservice.NewDefaultContextAssembler(learningStates, inventory, unitServing)
	videoStateEnricher := recommendationservice.NewDefaultVideoStateEnricher(videoServing, videoUserState)
	resolver := recommendationservice.NewDefaultEvidenceResolver(semanticSpans, transcriptSentences)
	resultWriter := recommendationservice.NewDefaultRecommendationResultWriter(
		recommendationtx.NewManager(h.Pool),
		failingAuditWriter{inner: recommendationservice.NewDefaultAuditWriter(auditRepo)},
		recommendationservice.NewDefaultServingStateManager(unitServing, videoServing),
	)

	usecase, err := recommendationusecase.NewGenerateVideoRecommendationsPipeline(
		assembler,
		recommendationplanner.NewDefaultDemandPlanner(),
		recommendationservice.NewDefaultCandidateGenerator(recommendable),
		resolver,
		recommendationaggregator.NewDefaultVideoEvidenceAggregator(),
		recommendationranking.NewDefaultVideoRanker(),
		recommendationselector.NewDefaultVideoSelector(),
		recommendationexplain.NewDefaultExplanationBuilder(),
		videoStateEnricher,
		resultWriter,
	)
	if err != nil {
		panic(err)
	}
	return usecase
}
