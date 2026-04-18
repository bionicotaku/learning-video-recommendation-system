//go:build integration

package tx_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/learningengine/application/service"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	persisttx "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/tx"
)

func TestManagerRollsBackTransactionOnError(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)

	manager := persisttx.NewManager(db.Pool)
	expectedErr := errors.New("force rollback")
	q4 := int16(4)

	err := manager.WithinTx(context.Background(), func(ctx context.Context, repos service.TransactionalRepositories) error {
		return repos.UnitLearningEvents().Append(ctx, []model.LearningEvent{
			{
				UserID:       userID,
				CoarseUnitID: 101,
				EventType:    "new_learn",
				SourceType:   "quiz_session",
				Quality:      &q4,
				Metadata:     []byte("{}"),
				OccurredAt:   time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
			},
		})
	})
	if err != nil {
		t.Fatalf("WithinTx() error on append = %v", err)
	}

	err = manager.WithinTx(context.Background(), func(ctx context.Context, repos service.TransactionalRepositories) error {
		if err := repos.UnitLearningEvents().Append(ctx, []model.LearningEvent{
			{
				UserID:       userID,
				CoarseUnitID: 101,
				EventType:    "review",
				SourceType:   "quiz_session",
				Quality:      &q4,
				Metadata:     []byte("{}"),
				OccurredAt:   time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC),
			},
		}); err != nil {
			return err
		}
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("WithinTx() error = %v, want expectedErr", err)
	}

	var count int
	if err := db.Pool.QueryRow(context.Background(), `select count(*) from learning.unit_learning_events where user_id = $1`, userID).Scan(&count); err != nil {
		t.Fatalf("count events error = %v", err)
	}
	if count != 1 {
		t.Fatalf("event count = %d, want 1", count)
	}
}

func TestManagerWithinUserTxSerializesSameUser(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	manager := persisttx.NewManager(db.Pool)

	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondFinished := make(chan struct{})
	var secondElapsed int64

	go func() {
		err := manager.WithinUserTx(context.Background(), "same-user", func(ctx context.Context, repos service.TransactionalRepositories) error {
			close(firstStarted)
			<-releaseFirst
			return nil
		})
		if err != nil {
			t.Errorf("first WithinUserTx() error = %v", err)
		}
	}()

	<-firstStarted

	go func() {
		start := time.Now()
		err := manager.WithinUserTx(context.Background(), "same-user", func(ctx context.Context, repos service.TransactionalRepositories) error {
			atomic.StoreInt64(&secondElapsed, int64(time.Since(start)))
			return nil
		})
		if err != nil {
			t.Errorf("second WithinUserTx() error = %v", err)
		}
		close(secondFinished)
	}()

	time.Sleep(150 * time.Millisecond)

	select {
	case <-secondFinished:
		t.Fatal("expected same-user transaction to block until first transaction finishes")
	default:
	}

	close(releaseFirst)

	select {
	case <-secondFinished:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second transaction to finish")
	}

	if got := time.Duration(atomic.LoadInt64(&secondElapsed)); got < 100*time.Millisecond {
		t.Fatalf("second transaction elapsed = %v, want blocking delay", got)
	}
}

func TestManagerWithinUserTxAllowsDifferentUsersConcurrently(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	manager := persisttx.NewManager(db.Pool)

	startBarrier := make(chan struct{})
	releaseBarrier := make(chan struct{})
	var concurrent int32
	var maxConcurrent int32
	var wg sync.WaitGroup

	run := func(userID string) {
		defer wg.Done()
		err := manager.WithinUserTx(context.Background(), userID, func(ctx context.Context, repos service.TransactionalRepositories) error {
			<-startBarrier
			current := atomic.AddInt32(&concurrent, 1)
			for {
				observed := atomic.LoadInt32(&maxConcurrent)
				if current <= observed || atomic.CompareAndSwapInt32(&maxConcurrent, observed, current) {
					break
				}
			}
			<-releaseBarrier
			atomic.AddInt32(&concurrent, -1)
			return nil
		})
		if err != nil {
			t.Errorf("WithinUserTx(%q) error = %v", userID, err)
		}
	}

	wg.Add(2)
	go run("user-a")
	go run("user-b")

	close(startBarrier)
	time.Sleep(150 * time.Millisecond)
	close(releaseBarrier)
	wg.Wait()

	if atomic.LoadInt32(&maxConcurrent) < 2 {
		t.Fatalf("max concurrent transactions = %d, want at least 2 for different users", maxConcurrent)
	}
}
