//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	"learning-video-recommendation-system/internal/test/e2e/testutil"
)

func TestE2E_DevModeAuthorizationFallbackAllowsFeedHTTP(t *testing.T) {
	h := harness(t)

	userID := h.NewUserID()
	unitID := h.NewUnitID()
	videoID := h.NewVideoID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, unitID)
	h.SeedCatalogVideo(t, strongSupplyVideo(videoID, unitID, 1_000, 2_000, 0, "feed-devmode", 95_000))
	if _, err := h.Pool.Exec(context.Background(), `
		insert into catalog.video_user_states (user_id, video_id, has_liked, has_bookmarked)
		values ($1, $2, true, true)`, userID, videoID); err != nil {
		t.Fatalf("seed feed interaction state: %v", err)
	}
	testutil.MustEnsureTarget(t, h.LearningSuite(), userID, targetSpec(unitID, 0.9, "feed-devmode"))
	h.RefreshRecommendationViews(t)

	server := h.DevModeAPIServer(t)
	t.Cleanup(server.Close)

	response := postJSONWithBearer(t, server, "/api/feed", devBearerToken(userID), `{"target_video_count":1,"client_context":{"source":"e2e"}}`)
	requireStatus(t, response, http.StatusOK)

	var body struct {
		RecommendationRunID string `json:"recommendation_run_id"`
		Items               []struct {
			VideoID       string `json:"video_id"`
			LearningUnits []struct {
				CoarseUnitID int64  `json:"coarse_unit_id"`
				Text         string `json:"text"`
			} `json:"learning_units"`
		} `json:"items"`
	}
	decodeResponse(t, response, &body)
	if body.RecommendationRunID == "" || len(body.Items) != 1 {
		t.Fatalf("feed body = %+v, want one recommendation item", body)
	}
	if body.Items[0].VideoID != videoID {
		t.Fatalf("feed item = %+v, want seeded video", body.Items[0])
	}
	if len(body.Items[0].LearningUnits) != 1 || body.Items[0].LearningUnits[0].CoarseUnitID != unitID {
		t.Fatalf("learning units = %+v, want seeded unit", body.Items[0].LearningUnits)
	}
	detailResponse := getHTTPWithBearer(t, server, "/api/videos/"+videoID, devBearerToken(userID))
	requireStatus(t, detailResponse, http.StatusOK)
	var detailBody struct {
		VideoID       string  `json:"video_id"`
		VideoURL      string  `json:"video_url"`
		TranscriptURL *string `json:"transcript_url"`
		UserState     struct {
			HasLiked     bool `json:"has_liked"`
			HasFavorited bool `json:"has_favorited"`
		} `json:"user_state"`
	}
	decodeResponse(t, detailResponse, &detailBody)
	if detailBody.VideoID != videoID || detailBody.VideoURL == "" {
		t.Fatalf("video detail = %+v, want seeded video with materialized URL", detailBody)
	}
	if detailBody.TranscriptURL == nil || *detailBody.TranscriptURL == "" {
		t.Fatalf("video detail transcript url = %+v, want materialized URL", detailBody.TranscriptURL)
	}
	if !detailBody.UserState.HasLiked || !detailBody.UserState.HasFavorited {
		t.Fatalf("video detail user_state = %+v, want liked and favorited", detailBody.UserState)
	}
	run := h.LoadRecommendationRun(t, body.RecommendationRunID)
	if run.ResultCount != 1 {
		t.Fatalf("run = %+v, want result_count=1", run)
	}
	if h.LoadVideoServingCount(t, userID, videoID) != 1 {
		t.Fatalf("video serving count not incremented")
	}
}

func TestE2E_UnitProgressHTTPListsMasteredAndUnmastered(t *testing.T) {
	h := harness(t)

	userID := h.NewUserID()
	masteredUnit := h.NewUnitID()
	unmasteredUnit := h.NewUnitID()
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	q4 := int16(4)
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, masteredUnit, unmasteredUnit)
	learning := h.LearningSuite()
	testutil.MustEnsureTarget(t, learning, userID,
		targetSpec(masteredUnit, 0.9, "unit-progress-mastered"),
		targetSpec(unmasteredUnit, 0.8, "unit-progress-unmastered"),
	)
	mustRecordEvents(t, learning, userID,
		learningdto.LearningEventInput{CoarseUnitID: masteredUnit, EventType: "self_mark_mastered", ReducerEffect: "set_mastered", SourceType: "learning_interaction_event", SourceRefID: "unit-progress-mastered", OccurredAt: now},
		learningdto.LearningEventInput{CoarseUnitID: unmasteredUnit, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", SourceRefID: "unit-progress-unmastered", ProgressQuality: &q4, OccurredAt: now.Add(time.Minute)},
	)

	server := h.APIServer(t, userID)
	t.Cleanup(server.Close)

	masteredResponse := getHTTP(t, server, "/api/learning/unit-progress/mastered?limit=10")
	requireStatus(t, masteredResponse, http.StatusOK)
	var masteredBody struct {
		Items []struct {
			CoarseUnitID    int64   `json:"coarse_unit_id"`
			Label           string  `json:"label"`
			ProgressPercent float64 `json:"progress_percent"`
			LastProgressAt  string  `json:"last_progress_at"`
		} `json:"items"`
	}
	decodeResponse(t, masteredResponse, &masteredBody)
	if len(masteredBody.Items) != 1 || masteredBody.Items[0].CoarseUnitID != masteredUnit || masteredBody.Items[0].ProgressPercent != 100 {
		t.Fatalf("mastered body = %+v, want mastered unit only", masteredBody)
	}
	if masteredBody.Items[0].Label == "" || masteredBody.Items[0].LastProgressAt == "" {
		t.Fatalf("mastered item missing display/progress fields: %+v", masteredBody.Items[0])
	}

	unmasteredResponse := getHTTP(t, server, "/api/learning/unit-progress/unmastered")
	requireStatus(t, unmasteredResponse, http.StatusOK)
	var unmasteredBody struct {
		Items []struct {
			CoarseUnitID    int64   `json:"coarse_unit_id"`
			ProgressPercent float64 `json:"progress_percent"`
		} `json:"items"`
	}
	decodeResponse(t, unmasteredResponse, &unmasteredBody)
	if len(unmasteredBody.Items) != 1 || unmasteredBody.Items[0].CoarseUnitID != unmasteredUnit || unmasteredBody.Items[0].ProgressPercent <= 0 {
		t.Fatalf("unmastered body = %+v, want in-progress target unit only", unmasteredBody)
	}
}

func TestE2E_VideoInteractionsHTTPUpdatesCatalogStateAndCounts(t *testing.T) {
	h := harness(t)

	userID := h.NewUserID()
	videoID := h.NewVideoID()
	h.SeedUser(t, userID)
	h.SeedCatalogVideo(t, testutil.CatalogVideoFixture{VideoID: videoID, DurationMs: 90_000, MappedSpanRatio: 0.8})

	server := h.APIServer(t, userID)
	t.Cleanup(server.Close)

	like := requestHTTP(t, server, http.MethodPut, "/api/videos/"+videoID+"/like", "")
	requireStatus(t, like, http.StatusOK)
	var likeBody struct {
		HasLiked bool  `json:"has_liked"`
		Count    int64 `json:"like_count"`
	}
	decodeResponse(t, like, &likeBody)
	if !likeBody.HasLiked || likeBody.Count != 1 {
		t.Fatalf("like body = %+v, want liked count=1", likeBody)
	}

	favorite := requestHTTP(t, server, http.MethodPut, "/api/videos/"+videoID+"/favorite", "")
	requireStatus(t, favorite, http.StatusOK)
	var favoriteBody struct {
		HasFavorited bool  `json:"has_favorited"`
		Count        int64 `json:"favorite_count"`
	}
	decodeResponse(t, favorite, &favoriteBody)
	if !favoriteBody.HasFavorited || favoriteBody.Count != 1 {
		t.Fatalf("favorite body = %+v, want favorited count=1", favoriteBody)
	}

	unlike := requestHTTP(t, server, http.MethodDelete, "/api/videos/"+videoID+"/like", "")
	requireStatus(t, unlike, http.StatusOK)
	decodeResponse(t, unlike, &likeBody)
	if likeBody.HasLiked || likeBody.Count != 0 {
		t.Fatalf("unlike body = %+v, want unliked count=0", likeBody)
	}

	state := loadVideoInteractionState(t, h.Pool, userID, videoID)
	if state.HasLiked || !state.HasBookmarked {
		t.Fatalf("interaction state = %+v, want unliked but favorited", state)
	}
	stats := loadVideoEngagementStats(t, h.Pool, videoID)
	if stats.LikeCount != 0 || stats.FavoriteCount != 1 {
		t.Fatalf("engagement stats = %+v, want like=0 favorite=1", stats)
	}
}

func TestE2E_EndQuizHTTPReturnsVideoContextQuestion(t *testing.T) {
	h := harness(t)

	userID := h.NewUserID()
	unitID := h.NewUnitID()
	videoID := h.NewVideoID()
	questionID := h.NewVideoID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, unitID)
	h.SeedCatalogVideo(t, strongSupplyVideo(videoID, unitID, 1_000, 2_000, 0, "end-quiz", 90_000))
	h.SeedEndQuizQuestion(t, questionID, unitID, videoID)

	server := h.APIServer(t, userID)
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/videos/end-quiz", `{
		"video_id": "`+videoID+`",
		"coarse_unit_ids": [`+itoa64(unitID)+`],
		"client_context": {"source":"e2e"}
	}`)
	requireStatus(t, response, http.StatusOK)

	var body struct {
		VideoID string `json:"video_id"`
		Items   []struct {
			CoarseUnitID int64  `json:"coarse_unit_id"`
			QuestionID   string `json:"question_id"`
			Source       string `json:"source"`
			Question     string `json:"question"`
			Options      []struct {
				OptionID string `json:"option_id"`
				Text     string `json:"text"`
			} `json:"options"`
		} `json:"items"`
	}
	decodeResponse(t, response, &body)
	if body.VideoID != videoID || len(body.Items) != 1 {
		t.Fatalf("end quiz body = %+v, want one item for video", body)
	}
	if body.Items[0].QuestionID != questionID || body.Items[0].Source != "video_context" || len(body.Items[0].Options) < 2 {
		t.Fatalf("end quiz item = %+v, want seeded video context question", body.Items[0])
	}
}

type videoInteractionStateRow struct {
	HasLiked      bool
	HasBookmarked bool
}

func postJSONWithBearer(t *testing.T, server *httptest.Server, path string, bearerToken string, body string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, server.URL+path, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+bearerToken)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post json with bearer: %v", err)
	}
	return response
}

func getHTTP(t *testing.T, server *httptest.Server, path string) *http.Response {
	t.Helper()
	return requestHTTP(t, server, http.MethodGet, path, "")
}

func getHTTPWithBearer(t *testing.T, server *httptest.Server, path string, bearerToken string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+bearerToken)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("get with bearer: %v", err)
	}
	return response
}

func requestHTTP(t *testing.T, server *httptest.Server, method string, path string, body string) *http.Response {
	t.Helper()
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	request, err := http.NewRequest(method, server.URL+path, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request http: %v", err)
	}
	return response
}

func devBearerToken(userID string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"` + userID + `"}`))
	return header + "." + payload + ".sig"
}

func loadVideoInteractionState(t *testing.T, db queryer, userID string, videoID string) videoInteractionStateRow {
	t.Helper()
	var row videoInteractionStateRow
	if err := db.QueryRow(context.Background(), `
		select has_liked, has_bookmarked
		from catalog.video_user_states
		where user_id = $1 and video_id = $2
	`, userID, videoID).Scan(&row.HasLiked, &row.HasBookmarked); err != nil {
		t.Fatalf("load video interaction state: %v", err)
	}
	return row
}
