//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	learningdto "learning-video-recommendation-system/internal/learningengine/application/dto"
	"learning-video-recommendation-system/internal/test/e2e/testutil"
)

func TestE2E_EnsureTargetWithoutEventsFeedsRecommendation(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecase()

	userID := h.NewUserID()
	unitID := h.NewUnitID()
	videoID := h.NewVideoID()

	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, unitID)
	h.SeedCatalogVideo(t, singleUnitVideo(videoID, unitID, 1_000, 3_000, 0, "new target", 90_000, 3, 0.08, 0.80))
	h.RefreshRecommendationViews(t)

	testutil.MustEnsureTarget(t, learning, userID, targetSpec(unitID, 0.95, "lesson_zero_event"))

	states, err := learning.ListUserUnitState.Execute(context.Background(), learningdto.ListUserUnitStatesRequest{
		UserID:           userID,
		OnlyTarget:       true,
		ExcludeSuspended: true,
	})
	if err != nil {
		t.Fatalf("ListUserUnitStates.Execute(): %v", err)
	}
	if len(states.States) != 1 {
		t.Fatalf("expected 1 state, got %d", len(states.States))
	}
	if !states.States[0].IsTarget || states.States[0].Status != "new" {
		t.Fatalf("unexpected state: %+v", states.States[0])
	}

	response := testutil.MustRecommend(t, recommendation, userID, 1)
	if response.SelectorMode != "normal" {
		t.Fatalf("selector_mode = %q, want normal", response.SelectorMode)
	}
	if response.Underfilled {
		t.Fatalf("underfilled = true, want false")
	}
	if len(response.Videos) != 1 || response.Videos[0].VideoID != videoID {
		t.Fatalf("unexpected videos: %#v", videoIDs(response.Videos))
	}
	if !containsReason(response.Videos[0].ReasonCodes, "new_unit_introduced") {
		t.Fatalf("reason codes = %#v, want new_unit_introduced", response.Videos[0].ReasonCodes)
	}
	if !containsUnit(response.Videos[0].CoveredNewNowUnits, unitID) {
		t.Fatalf("covered_new_now_units = %#v, want %d", response.Videos[0].CoveredNewNowUnits, unitID)
	}
}

func TestE2E_TargetControlsAreVisibleToRecommendation(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecase()

	userID := h.NewUserID()
	unitA := h.NewUnitID()
	unitB := h.NewUnitID()
	videoA := h.NewVideoID()
	videoB := h.NewVideoID()

	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, unitA, unitB)
	h.SeedCatalogVideo(t, singleUnitVideo(videoA, unitA, 1_000, 2_000, 0, "alpha", 90_000, 3, 0.07, 0.82))
	h.SeedCatalogVideo(t, singleUnitVideo(videoB, unitB, 5_000, 6_200, 2, "beta", 92_000, 3, 0.07, 0.82))
	h.RefreshRecommendationViews(t)

	testutil.MustEnsureTarget(t, learning, userID,
		targetSpec(unitA, 0.90, "lesson_a"),
		targetSpec(unitB, 0.80, "lesson_b"),
	)

	if _, err := learning.SuspendTargetUnit.Execute(context.Background(), learningdto.SuspendTargetUnitRequest{
		UserID:          userID,
		CoarseUnitID:    unitB,
		SuspendedReason: "manual_pause",
	}); err != nil {
		t.Fatalf("SuspendTargetUnit.Execute(): %v", err)
	}

	suspended := testutil.MustRecommend(t, recommendation, userID, 2)
	if videoIndex(suspended.Videos, videoB) != -1 {
		t.Fatalf("suspended video %s should be excluded, got %v", videoB, videoIDs(suspended.Videos))
	}

	if _, err := learning.ResumeTargetUnit.Execute(context.Background(), learningdto.ResumeTargetUnitRequest{
		UserID:       userID,
		CoarseUnitID: unitB,
	}); err != nil {
		t.Fatalf("ResumeTargetUnit.Execute(): %v", err)
	}

	resumed := testutil.MustRecommend(t, recommendation, userID, 2)
	if videoIndex(resumed.Videos, videoB) == -1 {
		t.Fatalf("resumed video %s should be visible, got %v", videoB, videoIDs(resumed.Videos))
	}

	if _, err := learning.SetTargetInactive.Execute(context.Background(), learningdto.SetTargetInactiveRequest{
		UserID:       userID,
		CoarseUnitID: unitA,
	}); err != nil {
		t.Fatalf("SetTargetInactive.Execute(): %v", err)
	}

	inactive := testutil.MustRecommend(t, recommendation, userID, 2)
	if videoIndex(inactive.Videos, videoA) != -1 {
		t.Fatalf("inactive target video %s should be excluded, got %v", videoA, videoIDs(inactive.Videos))
	}
	if videoIndex(inactive.Videos, videoB) == -1 {
		t.Fatalf("remaining active target video %s should be visible, got %v", videoB, videoIDs(inactive.Videos))
	}
}

func TestE2E_ReplayPreservesObservableRecommendationSemantics(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecase()

	userID := h.NewUserID()
	hardUnit := h.NewUnitID()
	newUnit := h.NewUnitID()
	softUnit := h.NewUnitID()
	futureUnit := h.NewUnitID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, hardUnit, newUnit, softUnit, futureUnit)

	hardVideo := h.NewVideoID()
	newVideo := h.NewVideoID()
	softVideo := h.NewVideoID()
	futureVideo := h.NewVideoID()
	h.SeedCatalogVideo(t, singleUnitVideo(hardVideo, hardUnit, 1_000, 2_200, 0, "hard", 90_000, 3, 0.08, 0.85))
	h.SeedCatalogVideo(t, singleUnitVideo(newVideo, newUnit, 2_500, 3_800, 2, "new", 95_000, 3, 0.08, 0.82))
	h.SeedCatalogVideo(t, singleUnitVideo(softVideo, softUnit, 4_000, 5_300, 4, "soft", 90_000, 3, 0.08, 0.82))
	h.SeedCatalogVideo(t, singleUnitVideo(futureVideo, futureUnit, 6_000, 7_400, 6, "future", 90_000, 3, 0.08, 0.82))
	h.RefreshRecommendationViews(t)

	testutil.MustEnsureTarget(t, learning, userID,
		targetSpec(hardUnit, 0.95, "hard"),
		targetSpec(newUnit, 0.80, "new"),
		targetSpec(softUnit, 0.70, "soft"),
		targetSpec(futureUnit, 0.60, "future"),
	)

	now := time.Now().UTC()
	q4 := int16(4)
	q2 := int16(2)
	if _, err := learning.RecordEvents.Execute(context.Background(), learningdto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []learningdto.LearningEventInput{
			{CoarseUnitID: hardUnit, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: mustTimeAdd(now, -48*time.Hour)},
			{CoarseUnitID: hardUnit, EventType: "review", SourceType: "quiz_session", Quality: &q2, OccurredAt: mustTimeAdd(now, -24*time.Hour)},
			{CoarseUnitID: softUnit, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: mustTimeAdd(now, -1*time.Hour)},
			{CoarseUnitID: futureUnit, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: mustTimeAdd(now, -72*time.Hour)},
			{CoarseUnitID: futureUnit, EventType: "review", SourceType: "quiz_session", Quality: &q4, OccurredAt: mustTimeAdd(now, -48*time.Hour)},
			{CoarseUnitID: futureUnit, EventType: "review", SourceType: "quiz_session", Quality: &q4, OccurredAt: mustTimeAdd(now, -12*time.Hour)},
		},
	}); err != nil {
		t.Fatalf("RecordLearningEvents.Execute(): %v", err)
	}

	beforeReplay := testutil.MustRecommend(t, recommendation, userID, 4)
	beforeStates, err := learning.ListUserUnitState.Execute(context.Background(), learningdto.ListUserUnitStatesRequest{
		UserID:           userID,
		OnlyTarget:       true,
		ExcludeSuspended: true,
	})
	if err != nil {
		t.Fatalf("ListUserUnitStates(before replay): %v", err)
	}

	if len(beforeReplay.Videos) == 0 || beforeReplay.Videos[0].VideoID != hardVideo {
		t.Fatalf("expected hard-review video first before replay, got %v", videoIDs(beforeReplay.Videos))
	}
	if videoIndex(beforeReplay.Videos, newVideo) == -1 {
		t.Fatalf("expected new video to remain in result set, got %v", videoIDs(beforeReplay.Videos))
	}

	if _, err := learning.ReplayUserStates.Execute(context.Background(), learningdto.ReplayUserStatesRequest{UserID: userID}); err != nil {
		t.Fatalf("ReplayUserStates.Execute(): %v", err)
	}

	afterReplay := testutil.MustRecommend(t, recommendation, userID, 4)
	afterStates, err := learning.ListUserUnitState.Execute(context.Background(), learningdto.ListUserUnitStatesRequest{
		UserID:           userID,
		OnlyTarget:       true,
		ExcludeSuspended: true,
	})
	if err != nil {
		t.Fatalf("ListUserUnitStates(after replay): %v", err)
	}

	if got, want := videoIDs(afterReplay.Videos), videoIDs(beforeReplay.Videos); len(got) != len(want) {
		t.Fatalf("video count changed after replay: before=%v after=%v", want, got)
	} else {
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("video order changed after replay: before=%v after=%v", want, got)
			}
		}
	}

	if len(beforeStates.States) != len(afterStates.States) {
		t.Fatalf("state count changed after replay: before=%d after=%d", len(beforeStates.States), len(afterStates.States))
	}
	for i := range beforeStates.States {
		before := beforeStates.States[i]
		after := afterStates.States[i]
		if before.CoarseUnitID != after.CoarseUnitID || before.Status != after.Status || before.IsTarget != after.IsTarget || before.TargetPriority != after.TargetPriority || before.ReviewCount != after.ReviewCount || before.StrongEventCount != after.StrongEventCount {
			t.Fatalf("observable state drift after replay: before=%+v after=%+v", before, after)
		}
	}
}
