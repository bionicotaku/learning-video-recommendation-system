package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appservice "learning-video-recommendation-system/internal/recommendation/application/service"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

func TestDefaultServingStateManagerApplySelectionDoesNotPreReadServingCounts(t *testing.T) {
	unitRepo := &preReadFailingUnitServingRepository{err: errors.New("unit pre-read should not happen")}
	videoRepo := &preReadFailingVideoServingRepository{err: errors.New("video pre-read should not happen")}
	manager := appservice.NewDefaultServingStateManager(unitRepo, videoRepo)

	err := manager.ApplySelection(context.Background(), "00000000-0000-0000-0000-000000000401", "00000000-0000-0000-0000-000000000101", []model.FinalRecommendationItem{
		{VideoID: "00000000-0000-0000-0000-000000000201", LearningUnits: []model.ExpectedLearningUnit{{CoarseUnitID: 301}, {CoarseUnitID: 301}}},
		{VideoID: "00000000-0000-0000-0000-000000000201", LearningUnits: []model.ExpectedLearningUnit{{CoarseUnitID: 302}}},
	})
	if err != nil {
		t.Fatalf("ApplySelection() error = %v", err)
	}

	if unitRepo.readCalled {
		t.Fatal("expected unit serving state writes to avoid pre-reading counts")
	}
	if videoRepo.readCalled {
		t.Fatal("expected video serving state writes to avoid pre-reading counts")
	}
	if len(unitRepo.incrementedUnitIDs) != 2 {
		t.Fatalf("incremented unit count = %d, want 2 distinct units", len(unitRepo.incrementedUnitIDs))
	}
	if len(videoRepo.incrementedVideoIDs) != 1 {
		t.Fatalf("incremented video count = %d, want 1 distinct video", len(videoRepo.incrementedVideoIDs))
	}
}

func TestDefaultServingStateManagerApplySelectionSkipsUnitServingForFillItems(t *testing.T) {
	unitRepo := &preReadFailingUnitServingRepository{}
	videoRepo := &preReadFailingVideoServingRepository{}
	manager := appservice.NewDefaultServingStateManager(unitRepo, videoRepo)

	err := manager.ApplySelection(context.Background(), "00000000-0000-0000-0000-000000000401", "00000000-0000-0000-0000-000000000101", []model.FinalRecommendationItem{
		{VideoID: "00000000-0000-0000-0000-000000000201", LearningUnits: []model.ExpectedLearningUnit{}},
	})
	if err != nil {
		t.Fatalf("ApplySelection() error = %v", err)
	}

	if len(unitRepo.incrementedUnitIDs) != 0 {
		t.Fatalf("incremented unit ids = %#v, want none", unitRepo.incrementedUnitIDs)
	}
	if len(videoRepo.incrementedVideoIDs) != 1 {
		t.Fatalf("incremented video ids = %#v, want one video", videoRepo.incrementedVideoIDs)
	}
}

type preReadFailingUnitServingRepository struct {
	err                error
	readCalled         bool
	incrementedUnitIDs []int64
}

func (r *preReadFailingUnitServingRepository) ListByUserAndUnitIDs(context.Context, string, []int64) ([]model.UserUnitServingState, error) {
	r.readCalled = true
	return nil, r.err
}

func (r *preReadFailingUnitServingRepository) IncrementServedCounts(_ context.Context, _ string, _ string, _ time.Time, coarseUnitIDs []int64) error {
	r.incrementedUnitIDs = append([]int64(nil), coarseUnitIDs...)
	return nil
}

type preReadFailingVideoServingRepository struct {
	err                 error
	readCalled          bool
	incrementedVideoIDs []string
}

func (r *preReadFailingVideoServingRepository) ListByUserAndVideoIDs(context.Context, string, []string) ([]model.UserVideoServingState, error) {
	r.readCalled = true
	return nil, r.err
}

func (r *preReadFailingVideoServingRepository) IncrementServedCounts(_ context.Context, _ string, _ string, _ time.Time, videoIDs []string) error {
	r.incrementedVideoIDs = append([]string(nil), videoIDs...)
	return nil
}
