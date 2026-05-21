//go:build e2e

package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	learningservice "learning-video-recommendation-system/internal/learningengine/reducer/application/service"
	learningusecase "learning-video-recommendation-system/internal/learningengine/reducer/application/usecase"
	learningrepo "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/repository"
	learningtx "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/tx"
	"learning-video-recommendation-system/internal/platform/postgres/pgtest"
	recommendationdto "learning-video-recommendation-system/internal/recommendation/application/dto"
	recommendationservice "learning-video-recommendation-system/internal/recommendation/application/service"
	recommendationusecase "learning-video-recommendation-system/internal/recommendation/application/usecase"
	recommendationaggregator "learning-video-recommendation-system/internal/recommendation/domain/aggregator"
	recommendationexplain "learning-video-recommendation-system/internal/recommendation/domain/explain"
	recommendationplanner "learning-video-recommendation-system/internal/recommendation/domain/planner"
	recommendationranking "learning-video-recommendation-system/internal/recommendation/domain/ranking"
	recommendationselector "learning-video-recommendation-system/internal/recommendation/domain/selector"
	recommendationrepo "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/repository"
	recommendationtx "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/tx"
)

type Harness struct {
	suite *pgtest.Suite
	db    *pgtest.Database
	Pool  *pgxpool.Pool

	mu          sync.Mutex
	nextUUIDSeq int64
	nextUnitID  int64
}

type LearningSuite struct {
	EnsureTargetUnits learningusecase.EnsureTargetUnitsUsecase
	SetTargetInactive learningusecase.SetTargetInactiveUsecase
	SuspendTargetUnit learningusecase.SuspendTargetUnitUsecase
	ResumeTargetUnit  learningusecase.ResumeTargetUnitUsecase
	RecordEvents      learningusecase.RecordLearningEventsUsecase
	ReplayUserStates  learningusecase.ReplayUserStatesUsecase
	ListUserUnitState learningusecase.ListUserUnitStatesUsecase
}

type RecommendationPlanItemView = recommendationdto.RecommendationPlanItem

type RecommendationRunSummary struct {
	SelectorMode string
	Underfilled  bool
	ResultCount  int32
}

type RecommendationItemSummary struct {
	Rank           int
	VideoID        string
	PrimaryLane    string
	DominantRole   string
	DominantUnitID *int64
	ReasonCodes    []string
	LearningUnits  []recommendationdto.ExpectedLearningUnit
}

type CatalogVideoFixture struct {
	VideoID           string
	DurationMs        int32
	Status            string
	VisibilityStatus  string
	PublishAt         *time.Time
	MappedSpanRatio   float64
	TranscriptEntries []TranscriptSentenceFixture
	SemanticSpans     []SemanticSpanFixture
	UnitIndexes       []VideoUnitIndexFixture
}

type TranscriptSentenceFixture struct {
	SentenceIndex int32
	StartMs       int32
	EndMs         int32
}

type SemanticSpanFixture struct {
	SentenceIndex int32
	SpanIndex     int32
	CoarseUnitID  *int64
	StartMs       int32
	EndMs         int32
}

type VideoUnitIndexFixture struct {
	CoarseUnitID              int64
	MentionCount              int32
	SentenceCount             int32
	CoverageMs                int32
	CoverageRatio             float64
	SentenceIndexes           []int32
	BestEvidenceSentenceIndex int32
	BestEvidenceSpanIndex     int32
}

func OpenHarness() (*Harness, error) {
	suite, err := pgtest.OpenSuite(pgtest.Options{
		TempDirPrefix:        "learning-recommendation-e2e-*",
		TemplateDatabaseName: "cross_module_e2e_template",
		DatabaseNamePrefix:   "cross_module_e2e",
		SchemaPlan:           e2eSchemaPlan(),
	})
	if err != nil {
		return nil, err
	}

	db, err := suite.OpenDatabase(context.Background(), "cross_module_e2e")
	if err != nil {
		_ = suite.Close()
		return nil, err
	}

	harness := &Harness{
		suite:       suite,
		db:          db,
		Pool:        db.Pool,
		nextUUIDSeq: 1000,
		nextUnitID:  100,
	}
	return harness, nil
}

func (h *Harness) Close() error {
	if h.db != nil {
		if err := h.db.Close(context.Background()); err != nil {
			return err
		}
	}
	if h.suite != nil {
		return h.suite.Close()
	}
	return nil
}

func (h *Harness) LearningSuite() *LearningSuite {
	stateRepo := learningrepo.NewUserUnitStateRepository(h.Pool)
	txManager := learningtx.NewManager(h.Pool)

	return &LearningSuite{
		EnsureTargetUnits: learningservice.NewEnsureTargetUnitsUsecase(txManager),
		SetTargetInactive: learningservice.NewSetTargetInactiveUsecase(txManager),
		SuspendTargetUnit: learningservice.NewSuspendTargetUnitUsecase(txManager),
		ResumeTargetUnit:  learningservice.NewResumeTargetUnitUsecase(txManager),
		RecordEvents:      learningservice.NewRecordLearningEventsUsecase(txManager),
		ReplayUserStates:  learningservice.NewReplayUserStatesUsecase(txManager),
		ListUserUnitState: learningservice.NewListUserUnitStatesUsecase(stateRepo),
	}
}

func (h *Harness) RecommendationUsecase() recommendationusecase.GenerateVideoRecommendationsUsecase {
	return h.RecommendationUsecaseWithResultWriter(nil)
}

func (h *Harness) RecommendationUsecaseWithResultWriter(resultWriter recommendationservice.RecommendationResultWriter) recommendationusecase.GenerateVideoRecommendationsUsecase {
	learningStates := recommendationrepo.NewLearningStateReader(h.Pool)
	inventory := recommendationrepo.NewUnitInventoryReader(h.Pool)
	unitServing := recommendationrepo.NewUnitServingStateRepository(h.Pool)
	videoServing := recommendationrepo.NewVideoServingStateRepository(h.Pool)
	videoUserState := recommendationrepo.NewVideoUserStateReader(h.Pool)
	recommendable := recommendationrepo.NewRecommendableVideoUnitReader(h.Pool)
	semanticSpans := recommendationrepo.NewSemanticSpanReader(h.Pool)
	transcriptSentences := recommendationrepo.NewTranscriptSentenceReader(h.Pool)
	auditRepo := recommendationrepo.NewRecommendationAuditRepository(h.Pool)

	assembler := recommendationservice.NewDefaultContextAssembler(
		learningStates,
		inventory,
		unitServing,
	)
	videoStateEnricher := recommendationservice.NewDefaultVideoStateEnricher(videoServing, videoUserState)
	resolver := recommendationservice.NewDefaultEvidenceResolver(semanticSpans, transcriptSentences)
	if resultWriter == nil {
		resultWriter = recommendationservice.NewDefaultRecommendationResultWriter(
			recommendationtx.NewManager(h.Pool),
			recommendationservice.NewDefaultAuditWriter(auditRepo),
			recommendationservice.NewDefaultServingStateManager(unitServing, videoServing),
		)
	}

	usecase, err := recommendationusecase.NewGenerateVideoRecommendationsPipeline(
		assembler,
		recommendationplanner.NewDefaultDemandPlanner(),
		recommendationservice.NewDefaultCandidateGenerator(recommendable),
		resolver,
		recommendationaggregator.NewDefaultVideoEvidenceAggregator(),
		recommendationranking.NewDefaultVideoRanker(),
		recommendationselector.NewDefaultVideoSelector(),
		recommendationservice.NewDefaultVideoFillService(recommendationrepo.NewVideoFillCandidateReader(h.Pool)),
		recommendationexplain.NewDefaultExplanationBuilder(),
		videoStateEnricher,
		resultWriter,
	)
	if err != nil {
		panic(err)
	}
	return usecase
}

func (h *Harness) SeedUser(t *testing.T, userID string) {
	if t != nil {
		t.Helper()
	}
	if _, err := h.Pool.Exec(context.Background(), `insert into auth.users (id) values ($1) on conflict (id) do nothing`, userID); err != nil {
		failNow(t, "seed auth.users: %v", err)
	}
}

func (h *Harness) SeedCoarseUnits(t *testing.T, unitIDs ...int64) {
	if t != nil {
		t.Helper()
	}
	for _, unitID := range unitIDs {
		if _, err := h.Pool.Exec(context.Background(), `
			insert into semantic.coarse_unit (
				id,
				kind,
				label,
				lang,
				chinese_label,
				english_label,
				status,
				version,
				fine_unit_ids,
				original_defs
			) values (
				$1::bigint,
				'word',
				'unit-' || $1::text,
				'en',
				'unit-cn-' || $1::text,
				'unit ' || $1::text,
				'active',
				1,
				'{}'::bigint[],
				'{}'::text[]
			) on conflict (id) do nothing`, unitID); err != nil {
			failNow(t, "seed semantic.coarse_unit %d: %v", unitID, err)
		}
	}
}

func (h *Harness) SeedCatalogVideo(t *testing.T, fixture CatalogVideoFixture) {
	if t != nil {
		t.Helper()
	}

	status := fixture.Status
	if status == "" {
		status = "active"
	}
	visibility := fixture.VisibilityStatus
	if visibility == "" {
		visibility = "public"
	}

	if _, err := h.Pool.Exec(
		context.Background(),
		`insert into catalog.videos (
			video_id,
			duration_ms,
			status,
			visibility_status,
			publish_at,
			title,
			description,
			video_object_path,
			thumbnail_url
		)
		 values ($1::uuid, $2, $3, $4, $5, 'E2E Video ' || $1::text, 'E2E description', 'videos/' || $1::text || '/master.m3u8', 'covers/' || $1::text || '.webp')
		 on conflict (video_id) do update
		 set duration_ms = excluded.duration_ms,
		     status = excluded.status,
		     visibility_status = excluded.visibility_status,
		     publish_at = excluded.publish_at,
		     title = excluded.title,
		     description = excluded.description,
		     video_object_path = excluded.video_object_path,
		     thumbnail_url = excluded.thumbnail_url`,
		fixture.VideoID,
		fixture.DurationMs,
		status,
		visibility,
		fixture.PublishAt,
	); err != nil {
		failNow(t, "seed catalog.videos: %v", err)
	}

	if _, err := h.Pool.Exec(
		context.Background(),
		`insert into catalog.video_transcripts (video_id, mapped_span_ratio)
		 values ($1, $2)
		 on conflict (video_id) do update set mapped_span_ratio = excluded.mapped_span_ratio`,
		fixture.VideoID,
		fixture.MappedSpanRatio,
	); err != nil {
		failNow(t, "seed catalog.video_transcripts: %v", err)
	}

	for _, sentence := range fixture.TranscriptEntries {
		if _, err := h.Pool.Exec(
			context.Background(),
			`insert into catalog.video_transcript_sentences (video_id, sentence_index, start_ms, end_ms, text, translation)
			 values ($1, $2, $3, $4, $5, $6)`,
			fixture.VideoID,
			sentence.SentenceIndex,
			sentence.StartMs,
			sentence.EndMs,
			fmt.Sprintf("sentence %d", sentence.SentenceIndex),
			fmt.Sprintf("句子 %d", sentence.SentenceIndex),
		); err != nil {
			failNow(t, "seed catalog.video_transcript_sentences: %v", err)
		}
	}

	for _, span := range fixture.SemanticSpans {
		if _, err := h.Pool.Exec(
			context.Background(),
			`insert into catalog.video_semantic_spans (
					video_id,
					sentence_index,
					span_index,
					coarse_unit_id,
					start_ms,
					end_ms,
					surface_text,
					explanation,
					base_form,
					translation,
					dictionary,
					mapping_reason
				) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			fixture.VideoID,
			span.SentenceIndex,
			span.SpanIndex,
			span.CoarseUnitID,
			span.StartMs,
			span.EndMs,
			fmt.Sprintf("span %d", span.SpanIndex),
			"test explanation",
			"span",
			"跨度",
			"test dictionary",
			"test mapping reason",
		); err != nil {
			failNow(t, "seed catalog.video_semantic_spans: %v", err)
		}
	}

	for _, entry := range fixture.UnitIndexes {
		if _, err := h.Pool.Exec(
			context.Background(),
			`insert into catalog.video_unit_index (
					video_id,
					coarse_unit_id,
					mention_count,
					sentence_count,
					coverage_ms,
					coverage_ratio,
					sentence_indexes,
					best_evidence_sentence_index,
					best_evidence_span_index,
					best_evidence_scores,
					best_evidence_question_reject_reason,
					best_evidence_selection_reason,
					best_evidence_candidate_score,
					best_evidence_target_text
				) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, '{}'::jsonb, null, 'test fixture', 8.3500, 'test target')`,
			fixture.VideoID,
			entry.CoarseUnitID,
			entry.MentionCount,
			entry.SentenceCount,
			entry.CoverageMs,
			entry.CoverageRatio,
			entry.SentenceIndexes,
			entry.BestEvidenceSentenceIndex,
			entry.BestEvidenceSpanIndex,
		); err != nil {
			failNow(t, "seed catalog.video_unit_index: %v", err)
		}
	}
}

func (h *Harness) SeedVideoUserState(t *testing.T, userID, videoID string, lastWatchedAt *time.Time, watchCount, completedCount int32, lastPositionMs, maxPositionMs int32, totalWatchMs int64) {
	if t != nil {
		t.Helper()
	}
	if _, err := h.Pool.Exec(
		context.Background(),
		`insert into catalog.video_user_states (
			user_id, video_id, last_watched_at, watch_count, completed_count, last_position_ms, max_position_ms, total_watch_ms
		) values ($1, $2, $3, $4, $5, $6, $7, $8)`,
		userID,
		videoID,
		lastWatchedAt,
		watchCount,
		completedCount,
		lastPositionMs,
		maxPositionMs,
		totalWatchMs,
	); err != nil {
		failNow(t, "seed catalog.video_user_states: %v", err)
	}
}

func (h *Harness) RefreshRecommendationViews(t *testing.T) {
	if t != nil {
		t.Helper()
	}
	ctx := context.Background()
	if _, err := h.Pool.Exec(ctx, `refresh materialized view recommendation.v_recommendable_video_units`); err != nil {
		failNow(t, "refresh recommendation.v_recommendable_video_units: %v", err)
	}
	if _, err := h.Pool.Exec(ctx, `refresh materialized view recommendation.v_unit_video_inventory`); err != nil {
		failNow(t, "refresh recommendation.v_unit_video_inventory: %v", err)
	}
}

func (h *Harness) CountRecommendationRuns(t *testing.T, userID string) int {
	t.Helper()
	var count int
	if err := h.Pool.QueryRow(context.Background(), `select count(*) from recommendation.video_recommendation_runs where user_id = $1`, userID).Scan(&count); err != nil {
		t.Fatalf("count recommendation runs: %v", err)
	}
	return count
}

func (h *Harness) CountRecommendationItems(t *testing.T, runID string) int {
	t.Helper()
	var count int
	if err := h.Pool.QueryRow(context.Background(), `select count(*) from recommendation.video_recommendation_items where run_id = $1`, runID).Scan(&count); err != nil {
		t.Fatalf("count recommendation items: %v", err)
	}
	return count
}

func (h *Harness) LoadVideoServingCount(t *testing.T, userID, videoID string) int {
	t.Helper()
	var servedCount int
	if err := h.Pool.QueryRow(context.Background(), `select served_count from recommendation.user_video_serving_states where user_id = $1 and video_id = $2`, userID, videoID).Scan(&servedCount); err != nil {
		t.Fatalf("load user_video_serving_states: %v", err)
	}
	return servedCount
}

func (h *Harness) LoadVideoServingCountOrZero(t *testing.T, userID, videoID string) int {
	t.Helper()
	var servedCount int
	err := h.Pool.QueryRow(context.Background(), `select served_count from recommendation.user_video_serving_states where user_id = $1 and video_id = $2`, userID, videoID).Scan(&servedCount)
	if err != nil {
		return 0
	}
	return servedCount
}

func (h *Harness) LoadUnitServingCount(t *testing.T, userID string, unitID int64) int {
	t.Helper()
	var servedCount int
	if err := h.Pool.QueryRow(context.Background(), `select served_count from recommendation.user_unit_serving_states where user_id = $1 and coarse_unit_id = $2`, userID, unitID).Scan(&servedCount); err != nil {
		t.Fatalf("load user_unit_serving_states: %v", err)
	}
	return servedCount
}

func (h *Harness) LoadUnitServingCountOrZero(t *testing.T, userID string, unitID int64) int {
	t.Helper()
	var servedCount int
	err := h.Pool.QueryRow(context.Background(), `select served_count from recommendation.user_unit_serving_states where user_id = $1 and coarse_unit_id = $2`, userID, unitID).Scan(&servedCount)
	if err != nil {
		return 0
	}
	return servedCount
}

func (h *Harness) LoadRecommendationRun(t *testing.T, runID string) RecommendationRunSummary {
	t.Helper()
	run := RecommendationRunSummary{}
	if err := h.Pool.QueryRow(
		context.Background(),
		`select selector_mode, underfilled, result_count
		 from recommendation.video_recommendation_runs
		 where run_id = $1`,
		runID,
	).Scan(&run.SelectorMode, &run.Underfilled, &run.ResultCount); err != nil {
		t.Fatalf("load recommendation run: %v", err)
	}
	return run
}

func (h *Harness) LoadRecommendationItems(t *testing.T, runID string) []RecommendationItemSummary {
	t.Helper()
	rows, err := h.Pool.Query(
		context.Background(),
		`select rank, video_id, primary_lane, dominant_role, dominant_unit_id, reason_codes, learning_units
			 from recommendation.video_recommendation_items
			 where run_id = $1
			 order by rank asc`,
		runID,
	)
	if err != nil {
		t.Fatalf("query recommendation items: %v", err)
	}
	defer rows.Close()

	items := make([]RecommendationItemSummary, 0)
	for rows.Next() {
		var item RecommendationItemSummary
		var reasonCodes []string
		var learningUnits []byte
		if err := rows.Scan(&item.Rank, &item.VideoID, &item.PrimaryLane, &item.DominantRole, &item.DominantUnitID, &reasonCodes, &learningUnits); err != nil {
			t.Fatalf("scan recommendation item: %v", err)
		}
		item.ReasonCodes = reasonCodes
		if err := json.Unmarshal(learningUnits, &item.LearningUnits); err != nil {
			t.Fatalf("decode recommendation item learning_units: %v", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate recommendation items: %v", err)
	}
	return items
}

func (h *Harness) LoadSupplyGrade(t *testing.T, unitID int64) string {
	t.Helper()
	var supplyGrade string
	if err := h.Pool.QueryRow(
		context.Background(),
		`select supply_grade
		 from recommendation.v_unit_video_inventory
		 where coarse_unit_id = $1`,
		unitID,
	).Scan(&supplyGrade); err != nil {
		t.Fatalf("load unit supply grade: %v", err)
	}
	return supplyGrade
}

func (h *Harness) LoadAuditLearningUnits(t *testing.T, runID string, rank int) []recommendationdto.ExpectedLearningUnit {
	t.Helper()
	var learningUnits []byte
	if err := h.Pool.QueryRow(
		context.Background(),
		`select learning_units
		 from recommendation.video_recommendation_items
		 where run_id = $1 and rank = $2`,
		runID,
		rank,
	).Scan(&learningUnits); err != nil {
		t.Fatalf("load recommendation audit learning_units: %v", err)
	}
	var result []recommendationdto.ExpectedLearningUnit
	if err := json.Unmarshal(learningUnits, &result); err != nil {
		t.Fatalf("decode recommendation audit learning_units: %v", err)
	}
	return result
}

func (h *Harness) NewUserID() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nextUUIDSeq++
	return fmt.Sprintf("00000000-0000-0000-0000-%012d", h.nextUUIDSeq)
}

func (h *Harness) NewVideoID() string {
	return h.NewUserID()
}

func (h *Harness) NewUnitID() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nextUnitID++
	return h.nextUnitID
}

func happyPathVideo(videoID string, unitID int64, startMs, endMs int32, sentenceIndex int32, surface string) CatalogVideoFixture {
	return CatalogVideoFixture{
		VideoID:         videoID,
		DurationMs:      120_000,
		MappedSpanRatio: 0.82,
		TranscriptEntries: []TranscriptSentenceFixture{
			{SentenceIndex: sentenceIndex, StartMs: startMs, EndMs: endMs},
			{SentenceIndex: sentenceIndex + 1, StartMs: endMs, EndMs: endMs + 1_500},
		},
		SemanticSpans: []SemanticSpanFixture{
			{SentenceIndex: sentenceIndex, SpanIndex: 0, CoarseUnitID: &unitID, StartMs: startMs, EndMs: endMs},
		},
		UnitIndexes: []VideoUnitIndexFixture{
			{
				CoarseUnitID:              unitID,
				MentionCount:              3,
				SentenceCount:             2,
				CoverageMs:                endMs + 1_500 - startMs,
				CoverageRatio:             0.08,
				SentenceIndexes:           []int32{sentenceIndex, sentenceIndex + 1},
				BestEvidenceSentenceIndex: sentenceIndex,
				BestEvidenceSpanIndex:     0,
			},
		},
	}
}

func e2eSchemaPlan() pgtest.SchemaPlan {
	return pgtest.NewSchemaPlan(
		pgtest.SQLFile(pgtest.RepoPath(
			"internal",
			"learningengine",
			"normalizer",
			"infrastructure",
			"persistence",
			"schema",
			"000000_external_refs.sql",
		)),
		pgtest.MigrationDir(pgtest.RepoPath(
			"internal",
			"analytics",
			"infrastructure",
			"migration",
		)),
		pgtest.MigrationDir(pgtest.RepoPath(
			"internal",
			"learningengine",
			"reducer",
			"infrastructure",
			"migration",
		)),
		pgtest.SQLText("e2e supplemental external catalog columns", supplementalExternalCatalogSQL()),
		pgtest.SQLFile(pgtest.RepoPath(
			"internal",
			"recommendation",
			"infrastructure",
			"persistence",
			"schema",
			"000000_external_refs.sql",
		)),
		pgtest.SQLText("drop placeholder recommendation materialized views", supplementalDropPlaceholderRecommendationViewsSQL()),
		pgtest.MigrationDir(pgtest.RepoPath(
			"internal",
			"recommendation",
			"infrastructure",
			"migration",
		)),
	)
}

func supplementalExternalCatalogSQL() string {
	return `-- supplemental-sql --
alter table if exists catalog.videos add column if not exists duration_ms integer not null default 0;
alter table if exists catalog.videos add column if not exists status text not null default 'active';
alter table if exists catalog.videos add column if not exists visibility_status text not null default 'public';
alter table if exists catalog.videos add column if not exists publish_at timestamptz;
alter table if exists catalog.videos add column if not exists title text not null default 'E2E Video';
alter table if exists catalog.videos add column if not exists description text;
alter table if exists catalog.videos add column if not exists video_object_path text not null default 'videos/e2e/master.m3u8';
alter table if exists catalog.videos add column if not exists thumbnail_url text;

alter table if exists catalog.questions add column if not exists scope_type text not null default 'unit';
alter table if exists catalog.questions add column if not exists question_type text not null default 'unit_meaning_choice';
alter table if exists catalog.questions add column if not exists coarse_unit_id bigint not null default 0;
alter table if exists catalog.questions add column if not exists target_text text not null default '';
alter table if exists catalog.questions add column if not exists video_id uuid;
alter table if exists catalog.questions add column if not exists context_sentence_index integer;
alter table if exists catalog.questions add column if not exists context_span_index integer;
alter table if exists catalog.questions add column if not exists context_start_ms integer;
alter table if exists catalog.questions add column if not exists context_end_ms integer;
alter table if exists catalog.questions add column if not exists content_payload jsonb not null default '{}'::jsonb;
alter table if exists catalog.questions add column if not exists status text not null default 'active';
alter table if exists catalog.questions add column if not exists created_at timestamptz not null default now();
alter table if exists catalog.questions add column if not exists updated_at timestamptz not null default now();

create table if not exists catalog.video_user_states (
  user_id uuid not null references auth.users(id) on delete cascade,
  video_id uuid not null references catalog.videos(video_id) on delete cascade,
  has_liked boolean not null default false,
  has_bookmarked boolean not null default false,
  has_watched boolean not null default false,
  liked_at timestamptz,
  bookmarked_at timestamptz,
  first_watched_at timestamptz,
  last_watched_at timestamptz,
  watch_count integer not null default 0,
  completed_count integer not null default 0,
  last_position_ms integer not null default 0,
  max_position_ms integer not null default 0,
  total_watch_ms bigint not null default 0,
  updated_at timestamptz not null default now(),
  primary key (user_id, video_id)
);

create table if not exists catalog.video_engagement_stats (
  video_id uuid primary key references catalog.videos(video_id) on delete cascade,
  view_count bigint not null default 0,
  like_count bigint not null default 0,
  favorite_count bigint not null default 0,
  completed_count bigint not null default 0,
  total_watch_ms bigint not null default 0,
  updated_at timestamptz not null default now()
);
`
}

func supplementalDropPlaceholderRecommendationViewsSQL() string {
	return `-- supplemental-sql --
drop materialized view if exists recommendation.v_unit_video_inventory;
drop materialized view if exists recommendation.v_recommendable_video_units;
`
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func failNow(t *testing.T, format string, args ...any) {
	if t != nil {
		t.Fatalf(format, args...)
	}
	panic(fmt.Sprintf(format, args...))
}

func MustEnsureTarget(t *testing.T, suite *LearningSuite, userID string, specs ...learningdto.TargetUnitSpec) {
	t.Helper()
	if _, err := suite.EnsureTargetUnits.Execute(context.Background(), learningdto.EnsureTargetUnitsRequest{
		UserID:  userID,
		Targets: specs,
	}); err != nil {
		t.Fatalf("EnsureTargetUnits.Execute(): %v", err)
	}
}

func MustRecommend(t *testing.T, usecase recommendationusecase.GenerateVideoRecommendationsUsecase, userID string, targetCount int) recommendationdto.GenerateVideoRecommendationsResponse {
	t.Helper()
	response, err := usecase.Execute(context.Background(), recommendationdto.GenerateVideoRecommendationsRequest{
		UserID:           userID,
		TargetVideoCount: targetCount,
		RequestContext:   []byte(`{"source":"e2e"}`),
	})
	if err != nil {
		t.Fatalf("GenerateVideoRecommendations.Execute(): %v", err)
	}
	return response
}
