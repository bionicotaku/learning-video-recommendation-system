package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgxpool"

	learningdto "learning-video-recommendation-system/internal/learningengine/application/dto"
	learningservice "learning-video-recommendation-system/internal/learningengine/application/service"
	learningusecase "learning-video-recommendation-system/internal/learningengine/application/usecase"
	learningrepo "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/repository"
	learningtx "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/tx"
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
	postgres *embeddedpostgres.EmbeddedPostgres
	Pool     *pgxpool.Pool

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

type RecommendationVideoView = recommendationdto.RecommendationVideo

type RecommendationRunSummary struct {
	SelectorMode string
	Underfilled  bool
	ResultCount  int32
}

type RecommendationItemSummary struct {
	Rank           int
	VideoID        string
	PrimaryLane    string
	DominantBucket string
	DominantUnitID *int64
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
	Text          string
	StartMs       int32
	EndMs         int32
	Explanation   string
}

type SemanticSpanFixture struct {
	SentenceIndex int32
	SpanIndex     int32
	CoarseUnitID  *int64
	StartMs       int32
	EndMs         int32
	Text          string
	Explanation   string
}

type EvidenceRefFixture struct {
	SentenceIndex int32 `json:"sentence_index"`
	SpanIndex     int32 `json:"span_index"`
}

type VideoUnitIndexFixture struct {
	CoarseUnitID       int64
	MentionCount       int32
	SentenceCount      int32
	FirstStartMs       int32
	LastEndMs          int32
	CoverageMs         int32
	CoverageRatio      float64
	SentenceIndexes    []int32
	EvidenceSpanRefs   []EvidenceRefFixture
	SampleSurfaceForms []string
}

func StartHarness(t *testing.T) *Harness {
	t.Helper()

	harness, err := OpenHarness(t.TempDir())
	if err != nil {
		t.Fatalf("open e2e harness: %v", err)
	}
	t.Cleanup(func() {
		if err := harness.Close(); err != nil {
			t.Fatalf("close e2e harness: %v", err)
		}
	})
	return harness
}

func OpenHarness(baseDir string) (*Harness, error) {
	port := freePort(nil)
	config := embeddedpostgres.DefaultConfig().
		Port(uint32(port)).
		Database("cross_module_e2e").
		Username("postgres").
		Password("postgres").
		RuntimePath(filepath.Join(baseDir, "runtime")).
		DataPath(filepath.Join(baseDir, "data")).
		BinariesPath(filepath.Join(baseDir, "bin"))

	postgres := embeddedpostgres.NewDatabase(config)
	if err := postgres.Start(); err != nil {
		return nil, fmt.Errorf("start embedded postgres: %w", err)
	}

	dsn := fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/cross_module_e2e?sslmode=disable", port)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		_ = postgres.Stop()
		return nil, fmt.Errorf("connect pgx pool: %w", err)
	}

	harness := &Harness{
		postgres:    postgres,
		Pool:        pool,
		nextUUIDSeq: 1000,
		nextUnitID:  100,
	}
	return harness, nil
}

func (h *Harness) applySchema(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := applySchemaSequence(ctx, func(ctx context.Context, sql string) error {
		_, err := h.Pool.Exec(ctx, sql)
		return err
	}, repoRoot(t), migrationFiles(t, filepath.Join(repoRoot(t), "internal", "learningengine", "infrastructure", "migration")), migrationFiles(t, filepath.Join(repoRoot(t), "internal", "recommendation", "infrastructure", "migration"))); err != nil {
		t.Fatalf("apply schema sequence: %v", err)
	}
}

func (h *Harness) ApplySchema(t *testing.T) {
	t.Helper()
	h.applySchema(t)
}

func (h *Harness) ApplySchemaForMain() error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	root := repoRootFromRuntime()
	return applySchemaSequence(
		ctx,
		func(ctx context.Context, sql string) error {
			_, err := h.Pool.Exec(ctx, sql)
			return err
		},
		root,
		migrationFilesForMain(filepath.Join(root, "internal", "learningengine", "infrastructure", "migration")),
		migrationFilesForMain(filepath.Join(root, "internal", "recommendation", "infrastructure", "migration")),
	)
}

func (h *Harness) Close() error {
	if h.Pool != nil {
		h.Pool.Close()
	}
	if h.postgres != nil {
		return h.postgres.Stop()
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
		if _, err := h.Pool.Exec(context.Background(), `insert into semantic.coarse_unit (id) values ($1) on conflict (id) do nothing`, unitID); err != nil {
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
		`insert into catalog.videos (video_id, duration_ms, status, visibility_status, publish_at)
		 values ($1, $2, $3, $4, $5)
		 on conflict (video_id) do update
		 set duration_ms = excluded.duration_ms,
		     status = excluded.status,
		     visibility_status = excluded.visibility_status,
		     publish_at = excluded.publish_at`,
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
			`insert into catalog.video_transcript_sentences (video_id, sentence_index, text, start_ms, end_ms, explanation)
			 values ($1, $2, $3, $4, $5, $6)`,
			fixture.VideoID,
			sentence.SentenceIndex,
			sentence.Text,
			sentence.StartMs,
			sentence.EndMs,
			nullIfEmpty(sentence.Explanation),
		); err != nil {
			failNow(t, "seed catalog.video_transcript_sentences: %v", err)
		}
	}

	for _, span := range fixture.SemanticSpans {
		if _, err := h.Pool.Exec(
			context.Background(),
			`insert into catalog.video_semantic_spans (video_id, sentence_index, span_index, coarse_unit_id, start_ms, end_ms, text, explanation)
			 values ($1, $2, $3, $4, $5, $6, $7, $8)`,
			fixture.VideoID,
			span.SentenceIndex,
			span.SpanIndex,
			span.CoarseUnitID,
			span.StartMs,
			span.EndMs,
			span.Text,
			nullIfEmpty(span.Explanation),
		); err != nil {
			failNow(t, "seed catalog.video_semantic_spans: %v", err)
		}
	}

	for _, entry := range fixture.UnitIndexes {
		evidenceBytes, err := json.Marshal(entry.EvidenceSpanRefs)
		if err != nil {
			failNow(t, "marshal evidence refs: %v", err)
		}
		if _, err := h.Pool.Exec(
			context.Background(),
			`insert into catalog.video_unit_index (
				video_id,
				coarse_unit_id,
				mention_count,
				sentence_count,
				first_start_ms,
				last_end_ms,
				coverage_ms,
				coverage_ratio,
				sentence_indexes,
				evidence_span_refs,
				sample_surface_forms
			) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			fixture.VideoID,
			entry.CoarseUnitID,
			entry.MentionCount,
			entry.SentenceCount,
			entry.FirstStartMs,
			entry.LastEndMs,
			entry.CoverageMs,
			entry.CoverageRatio,
			entry.SentenceIndexes,
			evidenceBytes,
			entry.SampleSurfaceForms,
		); err != nil {
			failNow(t, "seed catalog.video_unit_index: %v", err)
		}
	}
}

func (h *Harness) SeedVideoUserState(t *testing.T, userID, videoID string, lastWatchedAt *time.Time, watchCount, completedCount int32, lastWatchRatio, maxWatchRatio float64) {
	if t != nil {
		t.Helper()
	}
	if _, err := h.Pool.Exec(
		context.Background(),
		`insert into catalog.video_user_states (
			user_id, video_id, last_watched_at, watch_count, completed_count, last_watch_ratio, max_watch_ratio
		) values ($1, $2, $3, $4, $5, $6, $7)`,
		userID,
		videoID,
		lastWatchedAt,
		watchCount,
		completedCount,
		lastWatchRatio,
		maxWatchRatio,
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
		`select rank, video_id, primary_lane, dominant_bucket, dominant_unit_id
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
		if err := rows.Scan(&item.Rank, &item.VideoID, &item.PrimaryLane, &item.DominantBucket, &item.DominantUnitID); err != nil {
			t.Fatalf("scan recommendation item: %v", err)
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

func (h *Harness) LoadAuditEvidence(t *testing.T, runID string, rank int) (sentenceIndex, spanIndex, startMs, endMs *int32) {
	t.Helper()
	var si, spi, sm, em *int32
	if err := h.Pool.QueryRow(
		context.Background(),
		`select best_evidence_sentence_index, best_evidence_span_index, best_evidence_start_ms, best_evidence_end_ms
		 from recommendation.video_recommendation_items
		 where run_id = $1 and rank = $2`,
		runID,
		rank,
	).Scan(&si, &spi, &sm, &em); err != nil {
		t.Fatalf("load recommendation audit evidence: %v", err)
	}
	return si, spi, sm, em
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
			{SentenceIndex: sentenceIndex, Text: surface + " sentence", StartMs: startMs, EndMs: endMs},
			{SentenceIndex: sentenceIndex + 1, Text: surface + " follow up", StartMs: endMs, EndMs: endMs + 1_500},
		},
		SemanticSpans: []SemanticSpanFixture{
			{SentenceIndex: sentenceIndex, SpanIndex: 0, CoarseUnitID: &unitID, StartMs: startMs, EndMs: endMs, Text: surface},
		},
		UnitIndexes: []VideoUnitIndexFixture{
			{
				CoarseUnitID:       unitID,
				MentionCount:       3,
				SentenceCount:      2,
				FirstStartMs:       startMs,
				LastEndMs:          endMs + 1_500,
				CoverageMs:         endMs + 1_500 - startMs,
				CoverageRatio:      0.08,
				SentenceIndexes:    []int32{sentenceIndex, sentenceIndex + 1},
				EvidenceSpanRefs:   []EvidenceRefFixture{{SentenceIndex: sentenceIndex, SpanIndex: 0}},
				SampleSurfaceForms: []string{surface},
			},
		},
	}
}

func migrationFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migration dir %s: %v", dir, err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".up.sql") {
			files = append(files, filepath.Join(dir, name))
		}
	}
	sort.Strings(files)
	return files
}

func repoRoot(t *testing.T) string {
	t.Helper()
	return repoRootFromRuntime()
}

func repoRootFromRuntime() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", ".."))
}

func freePort(_ any) int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Sprintf("allocate free port: %v", err))
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func migrationFilesForMain(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		panic(fmt.Sprintf("read migration dir %s: %v", dir, err))
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".up.sql") {
			files = append(files, filepath.Join(dir, name))
		}
	}
	sort.Strings(files)
	return files
}

func applySchemaSequence(ctx context.Context, execSQL func(context.Context, string) error, root string, learningMigrations []string, recommendationMigrations []string) error {
	sequence := []string{filepath.Join(root, "internal", "learningengine", "infrastructure", "persistence", "schema", "000000_external_refs.sql")}
	sequence = append(sequence, learningMigrations...)
	sequence = append(sequence, supplementalExternalCatalogSQL())
	sequence = append(sequence, filepath.Join(root, "internal", "recommendation", "infrastructure", "persistence", "schema", "000000_external_refs.sql"))
	sequence = append(sequence, supplementalDropPlaceholderRecommendationViewsSQL())
	sequence = append(sequence, recommendationMigrations...)

	for _, item := range sequence {
		if strings.HasPrefix(item, "-- supplemental-sql --\n") {
			if err := execSQL(ctx, item); err != nil {
				return fmt.Errorf("exec supplemental sql: %w", err)
			}
			continue
		}

		content, err := os.ReadFile(item)
		if err != nil {
			return fmt.Errorf("read sql file %s: %w", item, err)
		}
		if err := execSQL(ctx, string(content)); err != nil {
			return fmt.Errorf("exec sql file %s: %w", item, err)
		}
	}

	return nil
}

func supplementalExternalCatalogSQL() string {
	return `-- supplemental-sql --
alter table if exists catalog.videos add column if not exists duration_ms integer not null default 0;
alter table if exists catalog.videos add column if not exists status text not null default 'active';
alter table if exists catalog.videos add column if not exists visibility_status text not null default 'public';
alter table if exists catalog.videos add column if not exists publish_at timestamptz;
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
		UserID:               userID,
		TargetVideoCount:     targetCount,
		PreferredDurationSec: [2]int{45, 180},
		SessionHint:          "e2e",
		RequestContext:       []byte(`{"source":"e2e"}`),
	})
	if err != nil {
		t.Fatalf("GenerateVideoRecommendations.Execute(): %v", err)
	}
	return response
}
