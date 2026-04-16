package service_test

import (
	"context"
	"errors"
	"testing"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	appservice "learning-video-recommendation-system/internal/recommendation/application/service"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

func TestDefaultVideoStateEnricherSkipsReadsForEmptyVideos(t *testing.T) {
	serving := &spyVideoServingRepository{}
	videoUser := &spyVideoUserStateReader{}
	enricher := appservice.NewDefaultVideoStateEnricher(serving, videoUser)

	contextModel, err := enricher.Enrich(context.Background(), model.RecommendationContext{
		Request: model.RecommendationRequest{UserID: "user-1"},
	}, nil)
	if err != nil {
		t.Fatalf("enrich: %v", err)
	}

	if serving.called || videoUser.called {
		t.Fatalf("expected no repository reads, serving=%v videoUser=%v", serving.called, videoUser.called)
	}
	if len(contextModel.VideoServingStates) != 0 || len(contextModel.VideoUserStates) != 0 {
		t.Fatalf("expected untouched empty states, got %+v", contextModel)
	}
}

func TestDefaultVideoStateEnricherLoadsVideoScopedStateWithCallerContext(t *testing.T) {
	ctxKey := struct{}{}
	serving := &spyVideoServingRepository{
		states: []model.UserVideoServingState{{VideoID: "video-1"}},
	}
	videoUser := &spyVideoUserStateReader{
		states: []model.VideoUserState{{VideoID: "video-1", WatchCount: 2}},
	}
	enricher := appservice.NewDefaultVideoStateEnricher(serving, videoUser)

	ctx := context.WithValue(context.Background(), ctxKey, "trace")
	contextModel, err := enricher.Enrich(ctx, model.RecommendationContext{
		Request: model.RecommendationRequest{UserID: "user-1"},
	}, []model.VideoCandidate{{VideoID: "video-1"}, {VideoID: "video-1"}, {VideoID: "video-2"}})
	if err != nil {
		t.Fatalf("enrich: %v", err)
	}

	if !serving.called || !videoUser.called {
		t.Fatalf("expected both repositories to be called, serving=%v videoUser=%v", serving.called, videoUser.called)
	}
	if got := serving.lastCtx.Value(ctxKey); got != "trace" {
		t.Fatalf("expected serving repo to receive caller ctx, got %#v", got)
	}
	if got := videoUser.lastCtx.Value(ctxKey); got != "trace" {
		t.Fatalf("expected video user repo to receive caller ctx, got %#v", got)
	}
	if len(serving.lastVideoIDs) != 2 || serving.lastVideoIDs[0] != "video-1" || serving.lastVideoIDs[1] != "video-2" {
		t.Fatalf("expected sorted distinct video IDs, got %#v", serving.lastVideoIDs)
	}
	if len(contextModel.VideoServingStates) != 1 || len(contextModel.VideoUserStates) != 1 {
		t.Fatalf("unexpected enriched context: %+v", contextModel)
	}
}

func TestDefaultVideoStateEnricherPropagatesContextCancellation(t *testing.T) {
	serving := &spyVideoServingRepository{err: context.Canceled}
	videoUser := &spyVideoUserStateReader{}
	enricher := appservice.NewDefaultVideoStateEnricher(serving, videoUser)

	_, err := enricher.Enrich(context.Background(), model.RecommendationContext{
		Request: model.RecommendationRequest{UserID: "user-1"},
	}, []model.VideoCandidate{{VideoID: "video-1"}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

type spyVideoServingRepository struct {
	states       []model.UserVideoServingState
	err          error
	called       bool
	lastCtx      context.Context
	lastUserID   string
	lastVideoIDs []string
}

func (s *spyVideoServingRepository) ListByUserAndVideoIDs(ctx context.Context, userID string, videoIDs []string) ([]model.UserVideoServingState, error) {
	s.called = true
	s.lastCtx = ctx
	s.lastUserID = userID
	s.lastVideoIDs = append([]string(nil), videoIDs...)
	if s.err != nil {
		return nil, s.err
	}
	return append([]model.UserVideoServingState(nil), s.states...), nil
}

func (s *spyVideoServingRepository) Upsert(context.Context, model.UserVideoServingState) error {
	return nil
}

var _ apprepo.VideoServingStateRepository = (*spyVideoServingRepository)(nil)

type spyVideoUserStateReader struct {
	states       []model.VideoUserState
	err          error
	called       bool
	lastCtx      context.Context
	lastUserID   string
	lastVideoIDs []string
}

func (s *spyVideoUserStateReader) ListByUserAndVideoIDs(ctx context.Context, userID string, videoIDs []string) ([]model.VideoUserState, error) {
	s.called = true
	s.lastCtx = ctx
	s.lastUserID = userID
	s.lastVideoIDs = append([]string(nil), videoIDs...)
	if s.err != nil {
		return nil, s.err
	}
	return append([]model.VideoUserState(nil), s.states...), nil
}

var _ apprepo.VideoUserStateReader = (*spyVideoUserStateReader)(nil)
