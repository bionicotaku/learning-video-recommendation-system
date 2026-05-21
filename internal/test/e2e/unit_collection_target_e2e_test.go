//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestE2E_ActivateUnitCollectionFeedsRecommendation(t *testing.T) {
	h := harness(t)

	userID := h.NewUserID()
	activeUnit := h.NewUnitID()
	masteredUnit := h.NewUnitID()
	activeVideo := h.NewVideoID()
	masteredVideo := h.NewVideoID()
	collectionID := h.NewVideoID()
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(), `delete from auth.users where id = $1`, userID)
		_, _ = h.Pool.Exec(context.Background(), `delete from catalog.videos where video_id in ($1::uuid, $2::uuid)`, activeVideo, masteredVideo)
		_, _ = h.Pool.Exec(context.Background(), `delete from semantic.unit_collections where collection_id = $1`, collectionID)
		h.RefreshRecommendationViews(t)
	})

	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, activeUnit, masteredUnit)
	h.SeedUnitCollection(t, collectionID, "toefl-core", "TOEFL Core", "active", activeUnit, masteredUnit)
	h.SeedCatalogVideo(t, strongSupplyVideo(activeVideo, activeUnit, 1_000, 2_000, 0, "active", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(masteredVideo, masteredUnit, 4_000, 5_000, 2, "mastered", 90_000))
	h.RefreshRecommendationViews(t)

	server := h.APIServer(t, userID)
	t.Cleanup(server.Close)

	list := get(t, server, "/api/unit-collections")
	requireStatus(t, list, http.StatusOK)

	activate := putJSON(t, server, "/api/learning-targets/active-collection", `{"collection_slug":"toefl-core"}`)
	requireStatus(t, activate, http.StatusOK)

	if _, err := h.Pool.Exec(context.Background(), `
		update learning.user_unit_states
		set status = 'mastered', progress_percent = 100
		where user_id = $1 and coarse_unit_id = $2`, userID, masteredUnit); err != nil {
		t.Fatalf("mark activated unit mastered: %v", err)
	}

	var masteredTarget bool
	if err := h.Pool.QueryRow(context.Background(), `
		select is_target
		from learning.user_unit_states
		where user_id = $1 and coarse_unit_id = $2`, userID, masteredUnit).Scan(&masteredTarget); err != nil {
		t.Fatalf("read mastered target state: %v", err)
	}
	if !masteredTarget {
		t.Fatalf("mastered collection unit should remain is_target=true")
	}

	feed := postJSON(t, server, "/api/feed", `{"target_video_count":1,"client_context":{"source":"unit_collection_e2e"}}`)
	requireStatus(t, feed, http.StatusOK)
	var body struct {
		Items []struct {
			VideoID       string `json:"video_id"`
			LearningUnits []struct {
				CoarseUnitID int64 `json:"coarse_unit_id"`
			} `json:"learning_units"`
		} `json:"items"`
	}
	decodeResponse(t, feed, &body)
	if len(body.Items) != 1 || body.Items[0].VideoID != activeVideo {
		t.Fatalf("feed items = %+v, want active unit video %s", body.Items, activeVideo)
	}
	if len(body.Items[0].LearningUnits) != 1 || body.Items[0].LearningUnits[0].CoarseUnitID != activeUnit {
		t.Fatalf("learning units = %+v, want active unit %d", body.Items[0].LearningUnits, activeUnit)
	}
}

func putJSON(t *testing.T, server *httptest.Server, path string, body string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPut, server.URL+path, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("put json: %v", err)
	}
	return response
}

func get(t *testing.T, server *httptest.Server, path string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	return response
}
