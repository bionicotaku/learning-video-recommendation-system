package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	appservice "learning-video-recommendation-system/internal/recommendation/application/service"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

func TestDefaultContextAssemblerAssembleAppliesDefaultsAndLoadsDependencies(t *testing.T) {
	learningStates := &stubLearningStateReader{
		states: []model.LearningStateSnapshot{
			{UserID: "user-1", CoarseUnitID: 101},
			{UserID: "user-1", CoarseUnitID: 202},
			{UserID: "user-1", CoarseUnitID: 101},
		},
	}
	inventory := &stubUnitInventoryReader{
		inventory: []model.UnitVideoInventory{
			{CoarseUnitID: 101},
			{CoarseUnitID: 202},
		},
	}
	unitServing := &stubUnitServingStateRepository{
		states: []model.UserUnitServingState{{UserID: "user-1", CoarseUnitID: 101, ServedCount: 2}},
	}

	assembler := appservice.NewDefaultContextAssembler(
		learningStates,
		inventory,
		unitServing,
		&stubVideoServingStateRepository{},
		&stubVideoUserStateReader{},
	)

	contextModel, err := assembler.Assemble(context.Background(), model.RecommendationRequest{
		UserID: "user-1",
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	if contextModel.Request.TargetVideoCount != 8 {
		t.Fatalf("expected default target video count, got %d", contextModel.Request.TargetVideoCount)
	}
	if contextModel.Request.PreferredDurationSec != [2]int{45, 180} {
		t.Fatalf("unexpected preferred duration: %#v", contextModel.Request.PreferredDurationSec)
	}
	if len(contextModel.ActiveUnitStates) != 3 {
		t.Fatalf("expected 3 active unit states, got %d", len(contextModel.ActiveUnitStates))
	}
	if len(contextModel.UnitInventory) != 2 {
		t.Fatalf("expected 2 inventory rows, got %d", len(contextModel.UnitInventory))
	}
	if len(contextModel.UnitServingStates) != 1 {
		t.Fatalf("expected 1 unit serving state, got %d", len(contextModel.UnitServingStates))
	}
	if len(contextModel.VideoServingStates) != 0 || len(contextModel.VideoUserStates) != 0 {
		t.Fatalf("expected lazy-loaded video state inputs to stay empty")
	}
	if inventory.lastUnitIDs[0] != 101 || inventory.lastUnitIDs[1] != 202 {
		t.Fatalf("unexpected unit ids order: %#v", inventory.lastUnitIDs)
	}
}

func TestDefaultContextAssemblerAssembleReturnsErrors(t *testing.T) {
	expectedErr := errors.New("boom")
	assembler := appservice.NewDefaultContextAssembler(
		&stubLearningStateReader{err: expectedErr},
		&stubUnitInventoryReader{},
		&stubUnitServingStateRepository{},
		&stubVideoServingStateRepository{},
		&stubVideoUserStateReader{},
	)

	if _, err := assembler.Assemble(context.Background(), model.RecommendationRequest{UserID: "user-1"}); !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

type stubLearningStateReader struct {
	states []model.LearningStateSnapshot
	err    error
}

func (s *stubLearningStateReader) ListActiveByUser(_ context.Context, _ string) ([]model.LearningStateSnapshot, error) {
	return s.states, s.err
}

type stubUnitInventoryReader struct {
	inventory   []model.UnitVideoInventory
	lastUnitIDs []int64
	err         error
}

func (s *stubUnitInventoryReader) ListByUnitIDs(_ context.Context, unitIDs []int64) ([]model.UnitVideoInventory, error) {
	s.lastUnitIDs = append([]int64(nil), unitIDs...)
	return s.inventory, s.err
}

type stubUnitServingStateRepository struct {
	states []model.UserUnitServingState
	err    error
}

func (s *stubUnitServingStateRepository) ListByUserAndUnitIDs(_ context.Context, _ string, _ []int64) ([]model.UserUnitServingState, error) {
	return s.states, s.err
}

func (s *stubUnitServingStateRepository) Upsert(context.Context, model.UserUnitServingState) error {
	return nil
}

type stubVideoServingStateRepository struct{}

func (s *stubVideoServingStateRepository) ListByUserAndVideoIDs(context.Context, string, []string) ([]model.UserVideoServingState, error) {
	return nil, nil
}

func (s *stubVideoServingStateRepository) Upsert(context.Context, model.UserVideoServingState) error {
	return nil
}

type stubVideoUserStateReader struct{}

func (s *stubVideoUserStateReader) ListByUserAndVideoIDs(context.Context, string, []string) ([]model.VideoUserState, error) {
	return nil, nil
}

var (
	_ apprepo.LearningStateReader         = (*stubLearningStateReader)(nil)
	_ apprepo.UnitInventoryReader         = (*stubUnitInventoryReader)(nil)
	_ apprepo.UnitServingStateRepository  = (*stubUnitServingStateRepository)(nil)
	_ apprepo.VideoServingStateRepository = (*stubVideoServingStateRepository)(nil)
	_ apprepo.VideoUserStateReader        = (*stubVideoUserStateReader)(nil)
)

func TestNormalizeDurationResetsInvalidRange(t *testing.T) {
	assembler := appservice.NewDefaultContextAssembler(
		&stubLearningStateReader{},
		&stubUnitInventoryReader{},
		&stubUnitServingStateRepository{},
		&stubVideoServingStateRepository{},
		&stubVideoUserStateReader{},
	)
	contextModel, err := assembler.Assemble(context.Background(), model.RecommendationRequest{
		UserID:               "user-1",
		PreferredDurationSec: [2]int{300, 30},
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if contextModel.Request.PreferredDurationSec != [2]int{45, 180} {
		t.Fatalf("unexpected duration after normalization: %#v", contextModel.Request.PreferredDurationSec)
	}
	if contextModel.Now.After(time.Now().UTC().Add(5 * time.Second)) {
		t.Fatalf("unexpected now value: %v", contextModel.Now)
	}
}
