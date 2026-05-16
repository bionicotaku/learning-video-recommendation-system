//go:build integration

package application_test

import (
	"context"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/analytics/application/dto"
	analyticsservice "learning-video-recommendation-system/internal/analytics/application/service"
	analyticsrepo "learning-video-recommendation-system/internal/analytics/infrastructure/persistence/repository"
)

func TestRecordSelfMarkMasteredWritesSelfMarkRawInteraction(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)

	usecase := analyticsservice.NewRecordSelfMarkMasteredUsecase(analyticsrepo.NewRawEventWriter(db.Pool))
	inputOccurredAt := time.Date(2026, 5, 15, 10, 0, 0, 0, time.FixedZone("PDT", -7*60*60))
	response, err := usecase.Execute(context.Background(), dto.RecordSelfMarkMasteredRequest{
		UserID:        userID,
		ClientContext: []byte(`{"platform":"ios"}`),
		ClientEventID: "self-mark-1",
		CoarseUnitID:  101,
		SourceSurface: "word_detail",
		TokenText:     "trivial",
		OccurredAt:    inputOccurredAt,
		EventPayload:  []byte(`{"entry":"lookup_sheet"}`),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !response.Accepted || !response.Inserted || response.LearningInteractionEventID == "" {
		t.Fatalf("response = %+v, want accepted inserted event id", response)
	}

	var eventType string
	var coarseUnitID int64
	var occurredAt time.Time
	if err := db.Pool.QueryRow(context.Background(), `
		select event_type, coarse_unit_id, occurred_at
		from analytics.learning_interaction_events
		where event_id = $1::uuid
	`, response.LearningInteractionEventID).Scan(&eventType, &coarseUnitID, &occurredAt); err != nil {
		t.Fatalf("read self mark raw event: %v", err)
	}
	if eventType != "self_mark_mastered" || coarseUnitID != 101 {
		t.Fatalf("raw event type/unit = %q/%d, want self_mark_mastered/101", eventType, coarseUnitID)
	}
	if !occurredAt.Equal(inputOccurredAt) {
		t.Fatalf("occurred_at = %v, want same instant as %v", occurredAt, inputOccurredAt)
	}
}

func TestRecordLearningInteractionsBatchRejectsSelfMarkMasteredInRealDBPath(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	unitID := int64(101)
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, unitID)

	usecase := analyticsservice.NewRecordLearningInteractionsBatchUsecase(analyticsrepo.NewRawEventWriter(db.Pool))
	_, err := usecase.Execute(context.Background(), dto.RecordLearningInteractionsBatchRequest{
		UserID:         userID,
		VideoID:        "22222222-2222-2222-2222-222222222222",
		WatchSessionID: "33333333-3333-3333-3333-333333333333",
		Events: []dto.LearningInteractionEventInput{
			{
				ClientEventID: "self-mark-in-batch",
				EventType:     "self_mark_mastered",
				SourceSurface: "word_detail",
				CoarseUnitID:  &unitID,
				OccurredAt:    time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
			},
		},
	})
	if err == nil {
		t.Fatalf("Execute() error = nil, want validation error")
	}

	var count int
	if err := db.Pool.QueryRow(context.Background(), `select count(*) from analytics.learning_interaction_events`).Scan(&count); err != nil {
		t.Fatalf("count interactions: %v", err)
	}
	if count != 0 {
		t.Fatalf("learning interaction rows = %d, want 0", count)
	}
}
