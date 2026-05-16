package service_test

import (
	"context"
	"testing"

	recommendationservice "learning-video-recommendation-system/internal/recommendation/application/service"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type fakeRecommendationAuditRepository struct {
	insertRunCalls   int
	insertItemsCalls int
	insertedRun      model.RecommendationRun
	insertedItems    []model.RecommendationItem
}

func (r *fakeRecommendationAuditRepository) InsertRun(_ context.Context, run model.RecommendationRun) error {
	r.insertRunCalls++
	r.insertedRun = run
	return nil
}

func (r *fakeRecommendationAuditRepository) InsertItems(_ context.Context, items []model.RecommendationItem) error {
	r.insertItemsCalls++
	r.insertedItems = append([]model.RecommendationItem(nil), items...)
	return nil
}

func TestDefaultAuditWriterWritesItemsInOneRepositoryCall(t *testing.T) {
	repo := &fakeRecommendationAuditRepository{}
	writer := recommendationservice.NewDefaultAuditWriter(repo)

	items := []model.RecommendationItem{
		{RunID: "run-1", Rank: 1, VideoID: "video-1"},
		{RunID: "run-1", Rank: 2, VideoID: "video-2"},
	}
	if err := writer.Write(context.Background(), model.RecommendationRun{RunID: "run-1"}, items); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if repo.insertRunCalls != 1 {
		t.Fatalf("InsertRun calls = %d, want 1", repo.insertRunCalls)
	}
	if repo.insertItemsCalls != 1 {
		t.Fatalf("InsertItems calls = %d, want 1", repo.insertItemsCalls)
	}
	if len(repo.insertedItems) != 2 {
		t.Fatalf("inserted items = %d, want 2", len(repo.insertedItems))
	}
}

func TestDefaultAuditWriterSkipsItemInsertForEmptyItems(t *testing.T) {
	repo := &fakeRecommendationAuditRepository{}
	writer := recommendationservice.NewDefaultAuditWriter(repo)

	if err := writer.Write(context.Background(), model.RecommendationRun{RunID: "run-1"}, nil); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if repo.insertRunCalls != 1 {
		t.Fatalf("InsertRun calls = %d, want 1", repo.insertRunCalls)
	}
	if repo.insertItemsCalls != 0 {
		t.Fatalf("InsertItems calls = %d, want 0", repo.insertItemsCalls)
	}
}
