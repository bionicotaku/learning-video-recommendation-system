package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"learning-video-recommendation-system/internal/learningengine/application/dto"
	applearningrepo "learning-video-recommendation-system/internal/learningengine/application/repository"
	"learning-video-recommendation-system/internal/learningengine/application/service"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	persistrepo "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/repository"
	persisttx "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/tx"
	"learning-video-recommendation-system/internal/learningengine/testutil"
)

func TestTargetControlUsecasesWithDatabase(t *testing.T) {
	db := testutil.StartPostgres(t)
	db.SeedUser(t, "11111111-1111-1111-1111-111111111111")
	db.SeedCoarseUnit(t, 101)

	txManager := persisttx.NewManager(db.Pool)

	ensureUsecase := service.NewEnsureTargetUnitsUsecase(txManager)
	listUsecase := service.NewListUserUnitStatesUsecase(persistrepo.NewUserUnitStateRepository(db.Pool))
	suspendUsecase := service.NewSuspendTargetUnitUsecase(txManager)
	resumeUsecase := service.NewResumeTargetUnitUsecase(txManager)

	if _, err := ensureUsecase.Execute(context.Background(), dto.EnsureTargetUnitsRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Targets: []dto.TargetUnitSpec{
			{CoarseUnitID: 101, TargetSource: "curriculum", TargetSourceRefID: "lesson_1", TargetPriority: 0.9},
		},
	}); err != nil {
		t.Fatalf("EnsureTargetUnits.Execute() error = %v", err)
	}

	response, err := listUsecase.Execute(context.Background(), dto.ListUserUnitStatesRequest{
		UserID:           "11111111-1111-1111-1111-111111111111",
		OnlyTarget:       true,
		ExcludeSuspended: true,
	})
	if err != nil {
		t.Fatalf("ListUserUnitStates.Execute() error = %v", err)
	}
	if len(response.States) != 1 {
		t.Fatalf("states len = %d, want 1", len(response.States))
	}
	if response.States[0].Status != "new" {
		t.Fatalf("status = %q, want new", response.States[0].Status)
	}

	if _, err := suspendUsecase.Execute(context.Background(), dto.SuspendTargetUnitRequest{
		UserID:          "11111111-1111-1111-1111-111111111111",
		CoarseUnitID:    101,
		SuspendedReason: "manual_pause",
	}); err != nil {
		t.Fatalf("SuspendTargetUnit.Execute() error = %v", err)
	}

	suspended, err := listUsecase.Execute(context.Background(), dto.ListUserUnitStatesRequest{
		UserID:           "11111111-1111-1111-1111-111111111111",
		OnlyTarget:       true,
		ExcludeSuspended: false,
	})
	if err != nil {
		t.Fatalf("ListUserUnitStates.Execute() error = %v", err)
	}
	if suspended.States[0].Status != "suspended" {
		t.Fatalf("status = %q, want suspended", suspended.States[0].Status)
	}

	if _, err := resumeUsecase.Execute(context.Background(), dto.ResumeTargetUnitRequest{
		UserID:       "11111111-1111-1111-1111-111111111111",
		CoarseUnitID: 101,
	}); err != nil {
		t.Fatalf("ResumeTargetUnit.Execute() error = %v", err)
	}

	resumed, err := listUsecase.Execute(context.Background(), dto.ListUserUnitStatesRequest{
		UserID:           "11111111-1111-1111-1111-111111111111",
		OnlyTarget:       true,
		ExcludeSuspended: true,
	})
	if err != nil {
		t.Fatalf("ListUserUnitStates.Execute() error = %v", err)
	}
	if resumed.States[0].Status != "new" {
		t.Fatalf("status = %q, want new", resumed.States[0].Status)
	}
}

func TestRecordLearningEventsWithDatabase(t *testing.T) {
	db := testutil.StartPostgres(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)

	recordUsecase := service.NewRecordLearningEventsUsecase(persisttx.NewManager(db.Pool))
	stateRepo := persistrepo.NewUserUnitStateRepository(db.Pool)

	q4 := int16(4)
	t1 := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(24 * time.Hour)

	response, err := recordUsecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 101, EventType: "review", SourceType: "quiz_session", Quality: &q4, OccurredAt: t2},
			{CoarseUnitID: 101, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: t1},
		},
	})
	if err != nil {
		t.Fatalf("RecordLearningEvents.Execute() error = %v", err)
	}
	if response.RecordedCount != 2 {
		t.Fatalf("RecordedCount = %d, want 2", response.RecordedCount)
	}

	states, err := stateRepo.ListByUser(context.Background(), userID, model.UserUnitStateFilter{})
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("states len = %d, want 1", len(states))
	}
	if states[0].Status != "reviewing" {
		t.Fatalf("status = %q, want reviewing", states[0].Status)
	}
}

func TestRecordLearningEventsRollsBackWhenStateWriteFails(t *testing.T) {
	db := testutil.StartPostgres(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)

	usecase := service.NewRecordLearningEventsUsecase(&failingBatchUpsertTxManager{pool: db.Pool})
	q4 := int16(4)
	t1 := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

	_, err := usecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 101, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: t1},
		},
	})
	if !errors.Is(err, errForcedBatchUpsertFailure) {
		t.Fatalf("Execute() error = %v, want errForcedBatchUpsertFailure", err)
	}

	var count int
	if err := db.Pool.QueryRow(context.Background(), `select count(*) from learning.unit_learning_events where user_id = $1`, userID).Scan(&count); err != nil {
		t.Fatalf("QueryRow() error = %v", err)
	}
	if count != 0 {
		t.Fatalf("event count = %d, want 0 after rollback", count)
	}
}

func TestReplayUserStatesWithDatabase(t *testing.T) {
	db := testutil.StartPostgres(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	db.SeedCoarseUnit(t, 102)

	stateRepo := persistrepo.NewUserUnitStateRepository(db.Pool)
	txManager := persisttx.NewManager(db.Pool)

	ensureUsecase := service.NewEnsureTargetUnitsUsecase(txManager)
	suspendUsecase := service.NewSuspendTargetUnitUsecase(txManager)
	recordUsecase := service.NewRecordLearningEventsUsecase(txManager)
	replayUsecase := service.NewReplayUserStatesUsecase(txManager)

	if _, err := ensureUsecase.Execute(context.Background(), dto.EnsureTargetUnitsRequest{
		UserID: userID,
		Targets: []dto.TargetUnitSpec{
			{CoarseUnitID: 101, TargetSource: "curriculum", TargetSourceRefID: "lesson_1", TargetPriority: 0.9},
			{CoarseUnitID: 102, TargetSource: "curriculum", TargetSourceRefID: "lesson_1", TargetPriority: 0.8},
		},
	}); err != nil {
		t.Fatalf("EnsureTargetUnits.Execute() error = %v", err)
	}

	q4 := int16(4)
	t1 := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(24 * time.Hour)
	if _, err := recordUsecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 101, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: t1},
			{CoarseUnitID: 101, EventType: "review", SourceType: "quiz_session", Quality: &q4, OccurredAt: t2},
		},
	}); err != nil {
		t.Fatalf("RecordLearningEvents.Execute() error = %v", err)
	}

	if _, err := suspendUsecase.Execute(context.Background(), dto.SuspendTargetUnitRequest{
		UserID:          userID,
		CoarseUnitID:    102,
		SuspendedReason: "manual_pause",
	}); err != nil {
		t.Fatalf("SuspendTargetUnit.Execute() error = %v", err)
	}

	beforeReplay, err := stateRepo.ListByUser(context.Background(), userID, model.UserUnitStateFilter{})
	if err != nil {
		t.Fatalf("ListByUser(before replay) error = %v", err)
	}

	response, err := replayUsecase.Execute(context.Background(), dto.ReplayUserStatesRequest{UserID: userID})
	if err != nil {
		t.Fatalf("ReplayUserStates.Execute() error = %v", err)
	}
	if response.ProcessedEventCount != 2 {
		t.Fatalf("ProcessedEventCount = %d, want 2", response.ProcessedEventCount)
	}
	if response.RebuiltUnitCount != 2 {
		t.Fatalf("RebuiltUnitCount = %d, want 2", response.RebuiltUnitCount)
	}

	afterReplay, err := stateRepo.ListByUser(context.Background(), userID, model.UserUnitStateFilter{})
	if err != nil {
		t.Fatalf("ListByUser(after replay) error = %v", err)
	}

	beforeByUnit := indexStatesByUnit(beforeReplay)
	afterByUnit := indexStatesByUnit(afterReplay)

	if afterByUnit[101].Status != "reviewing" {
		t.Fatalf("unit 101 status = %q, want reviewing", afterByUnit[101].Status)
	}
	if afterByUnit[102].Status != "suspended" {
		t.Fatalf("unit 102 status = %q, want suspended", afterByUnit[102].Status)
	}
	if afterByUnit[102].StrongEventCount != 0 {
		t.Fatalf("unit 102 strong_event_count = %d, want 0", afterByUnit[102].StrongEventCount)
	}
	if afterByUnit[101].Repetition != beforeByUnit[101].Repetition || afterByUnit[101].IntervalDays != beforeByUnit[101].IntervalDays {
		t.Fatalf("unit 101 replay progression mismatch: before=%+v after=%+v", beforeByUnit[101], afterByUnit[101])
	}
}

func TestReplayAndRecordSerializeForSameUser(t *testing.T) {
	db := testutil.StartPostgres(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)

	baseManager := persisttx.NewManager(db.Pool)
	replayGate := newBlockingUserTxManager(baseManager, userID)
	recordUsecase := service.NewRecordLearningEventsUsecase(baseManager)
	replayUsecase := service.NewReplayUserStatesUsecase(replayGate)
	stateRepo := persistrepo.NewUserUnitStateRepository(db.Pool)

	firstQuality := int16(4)
	firstOccurredAt := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	if _, err := recordUsecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 101, EventType: "new_learn", SourceType: "quiz_session", Quality: &firstQuality, OccurredAt: firstOccurredAt},
		},
	}); err != nil {
		t.Fatalf("seed RecordLearningEvents.Execute() error = %v", err)
	}

	var replayErr error
	var replayWG sync.WaitGroup
	replayWG.Add(1)
	go func() {
		defer replayWG.Done()
		_, replayErr = replayUsecase.Execute(context.Background(), dto.ReplayUserStatesRequest{UserID: userID})
	}()

	<-replayGate.started

	recordDone := make(chan error, 1)
	go func() {
		secondQuality := int16(4)
		_, err := recordUsecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
			UserID: userID,
			Events: []dto.LearningEventInput{
				{CoarseUnitID: 101, EventType: "review", SourceType: "quiz_session", Quality: &secondQuality, OccurredAt: firstOccurredAt.Add(24 * time.Hour)},
			},
		})
		recordDone <- err
	}()

	select {
	case err := <-recordDone:
		t.Fatalf("record completed before replay released lock: %v", err)
	case <-time.After(150 * time.Millisecond):
	}

	close(replayGate.release)
	replayWG.Wait()
	if replayErr != nil {
		t.Fatalf("ReplayUserStates.Execute() error = %v", replayErr)
	}

	select {
	case err := <-recordDone:
		if err != nil {
			t.Fatalf("RecordLearningEvents.Execute() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for blocked record to finish")
	}

	states, err := stateRepo.ListByUser(context.Background(), userID, model.UserUnitStateFilter{})
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("states len = %d, want 1", len(states))
	}
	if states[0].Status != "reviewing" || states[0].StrongEventCount != 2 || states[0].ReviewCount != 1 {
		t.Fatalf("unexpected final state after replay+record serialization: %+v", states[0])
	}

	replayed, err := service.NewReplayUserStatesUsecase(baseManager).Execute(context.Background(), dto.ReplayUserStatesRequest{UserID: userID})
	if err != nil {
		t.Fatalf("final replay Execute() error = %v", err)
	}
	if replayed.ProcessedEventCount != 2 {
		t.Fatalf("ProcessedEventCount after final replay = %d, want 2", replayed.ProcessedEventCount)
	}
}

var errForcedBatchUpsertFailure = errors.New("forced batch upsert failure")

type failingBatchUpsertTxManager struct {
	pool *pgxpool.Pool
}

func (m *failingBatchUpsertTxManager) WithinTx(ctx context.Context, fn func(ctx context.Context, repos service.TransactionalRepositories) error) error {
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	repos := failingBatchUpsertRepositories{
		userUnitStates: &failingBatchUpsertUserUnitStateRepository{
			inner: persistrepo.NewUserUnitStateRepository(tx),
		},
		unitLearningEvents: persistrepo.NewUnitLearningEventRepository(tx),
	}
	if err := fn(ctx, repos); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (m *failingBatchUpsertTxManager) WithinUserTx(ctx context.Context, _ string, fn func(ctx context.Context, repos service.TransactionalRepositories) error) error {
	return m.WithinTx(ctx, fn)
}

type failingBatchUpsertRepositories struct {
	userUnitStates     applearningrepo.UserUnitStateRepository
	unitLearningEvents applearningrepo.UnitLearningEventRepository
}

func (r failingBatchUpsertRepositories) UserUnitStates() applearningrepo.UserUnitStateRepository {
	return r.userUnitStates
}

func (r failingBatchUpsertRepositories) TargetCommands() applearningrepo.TargetStateCommandRepository {
	return nil
}

func (r failingBatchUpsertRepositories) UnitLearningEvents() applearningrepo.UnitLearningEventRepository {
	return r.unitLearningEvents
}

type failingBatchUpsertUserUnitStateRepository struct {
	inner applearningrepo.UserUnitStateRepository
}

func (r *failingBatchUpsertUserUnitStateRepository) GetByUserAndUnitForUpdate(ctx context.Context, userID string, coarseUnitID int64) (*model.UserUnitState, error) {
	return r.inner.GetByUserAndUnitForUpdate(ctx, userID, coarseUnitID)
}

func (r *failingBatchUpsertUserUnitStateRepository) Upsert(ctx context.Context, state *model.UserUnitState) (*model.UserUnitState, error) {
	return r.inner.Upsert(ctx, state)
}

func (r *failingBatchUpsertUserUnitStateRepository) BatchUpsert(context.Context, []*model.UserUnitState) ([]*model.UserUnitState, error) {
	return nil, errForcedBatchUpsertFailure
}

func (r *failingBatchUpsertUserUnitStateRepository) DeleteByUser(ctx context.Context, userID string) error {
	return r.inner.DeleteByUser(ctx, userID)
}

func (r *failingBatchUpsertUserUnitStateRepository) ListByUser(ctx context.Context, userID string, filter model.UserUnitStateFilter) ([]model.UserUnitState, error) {
	return r.inner.ListByUser(ctx, userID, filter)
}

func indexStatesByUnit(states []model.UserUnitState) map[int64]model.UserUnitState {
	indexed := make(map[int64]model.UserUnitState, len(states))
	for _, state := range states {
		indexed[state.CoarseUnitID] = state
	}
	return indexed
}

type blockingUserTxManager struct {
	inner   service.TxManager
	userID  string
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func newBlockingUserTxManager(inner service.TxManager, userID string) *blockingUserTxManager {
	return &blockingUserTxManager{
		inner:   inner,
		userID:  userID,
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (m *blockingUserTxManager) WithinTx(ctx context.Context, fn func(ctx context.Context, repos service.TransactionalRepositories) error) error {
	return m.inner.WithinTx(ctx, fn)
}

func (m *blockingUserTxManager) WithinUserTx(ctx context.Context, userID string, fn func(ctx context.Context, repos service.TransactionalRepositories) error) error {
	return m.inner.WithinUserTx(ctx, userID, func(ctx context.Context, repos service.TransactionalRepositories) error {
		if userID == m.userID {
			m.once.Do(func() { close(m.started) })
			<-m.release
		}
		return fn(ctx, repos)
	})
}
