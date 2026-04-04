package rule

import (
	"testing"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"

	"github.com/google/uuid"
)

func TestWeakEventHandlerExposureAndLookupDoNotAdvanceScheduling(t *testing.T) {
	handler := NewWeakEventHandler()
	userID := uuid.New()
	occurredAt := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	nextReviewAt := occurredAt.Add(24 * time.Hour)

	base := &model.UserUnitState{
		UserID:       userID,
		CoarseUnitID: 42,
		Status:       enum.UnitStatusNew,
		SeenCount:    2,
		Repetition:   3,
		IntervalDays: 6,
		EaseFactor:   2.5,
		NextReviewAt: &nextReviewAt,
	}

	tests := []struct {
		name      string
		eventType enum.EventType
	}{
		{name: "exposure", eventType: enum.EventTypeExposure},
		{name: "lookup", eventType: enum.EventTypeLookup},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := model.LearningEvent{
				UserID:       userID,
				CoarseUnitID: 42,
				EventType:    tt.eventType,
				OccurredAt:   occurredAt,
			}

			got, err := handler.Apply(base, event)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}

			if got.SeenCount != base.SeenCount+1 {
				t.Fatalf("SeenCount = %d, want %d", got.SeenCount, base.SeenCount+1)
			}
			if got.LastSeenAt == nil || !got.LastSeenAt.Equal(occurredAt) {
				t.Fatalf("LastSeenAt = %v, want %v", got.LastSeenAt, occurredAt)
			}
			if got.Status != base.Status {
				t.Fatalf("Status = %q, want %q", got.Status, base.Status)
			}
			if got.Repetition != base.Repetition {
				t.Fatalf("Repetition = %d, want %d", got.Repetition, base.Repetition)
			}
			if got.IntervalDays != base.IntervalDays {
				t.Fatalf("IntervalDays = %v, want %v", got.IntervalDays, base.IntervalDays)
			}
			if got.EaseFactor != base.EaseFactor {
				t.Fatalf("EaseFactor = %v, want %v", got.EaseFactor, base.EaseFactor)
			}
			if got.NextReviewAt == nil || !got.NextReviewAt.Equal(nextReviewAt) {
				t.Fatalf("NextReviewAt = %v, want %v", got.NextReviewAt, nextReviewAt)
			}
		})
	}
}
