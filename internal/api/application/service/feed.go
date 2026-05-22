package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
	catalogusecase "learning-video-recommendation-system/internal/catalog/application/usecase"
	recommendationdto "learning-video-recommendation-system/internal/recommendation/application/dto"
	recommendationusecase "learning-video-recommendation-system/internal/recommendation/application/usecase"
)

type PublicAssetURLBuilder struct {
	baseURL string
}

func NewPublicAssetURLBuilder(baseURL string) PublicAssetURLBuilder {
	return PublicAssetURLBuilder{baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/")}
}

func (b PublicAssetURLBuilder) Build(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("asset path is required")
	}
	if isAbsoluteHTTPURL(path) {
		return path, nil
	}
	if b.baseURL == "" {
		return "", fmt.Errorf("public asset base url is required for relative asset path")
	}
	return b.baseURL + "/" + strings.TrimLeft(path, "/"), nil
}

type FeedService struct {
	recommender recommendationusecase.GenerateVideoRecommendationsUsecase
	videoLookup catalogusecase.FeedVideoLookupUsecase
	labelLookup catalogusecase.UnitLabelLookupUsecase
	urlBuilder  PublicAssetURLBuilder
	logger      *slog.Logger
}

func NewFeedService(
	recommender recommendationusecase.GenerateVideoRecommendationsUsecase,
	videoLookup catalogusecase.FeedVideoLookupUsecase,
	labelLookup catalogusecase.UnitLabelLookupUsecase,
	urlBuilder PublicAssetURLBuilder,
	logger *slog.Logger,
) *FeedService {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &FeedService{
		recommender: recommender,
		videoLookup: videoLookup,
		labelLookup: labelLookup,
		urlBuilder:  urlBuilder,
		logger:      logger,
	}
}

func (s *FeedService) Execute(ctx context.Context, request apvdto.GetFeedRequest) (apvdto.FeedResponse, error) {
	if request.UserID == "" {
		return apvdto.FeedResponse{}, InvalidRequestError("user_id is required")
	}
	if s.recommender == nil {
		return apvdto.FeedResponse{}, fmt.Errorf("recommendation usecase is required")
	}
	if s.videoLookup == nil {
		return apvdto.FeedResponse{}, fmt.Errorf("feed video lookup usecase is required")
	}
	if s.labelLookup == nil {
		return apvdto.FeedResponse{}, fmt.Errorf("unit label lookup usecase is required")
	}

	plan, err := s.recommender.Execute(ctx, recommendationdto.GenerateVideoRecommendationsRequest{
		UserID:           request.UserID,
		TargetVideoCount: request.TargetVideoCount,
		RequestContext:   request.ClientContext,
	})
	if err != nil {
		return apvdto.FeedResponse{}, err
	}

	videoLookup, err := s.videoLookup.Execute(ctx, catalogdto.FeedVideoLookupRequest{
		UserID:   request.UserID,
		VideoIDs: planVideoIDs(plan.Items),
	})
	if err != nil {
		return apvdto.FeedResponse{}, err
	}
	videosByID := make(map[string]catalogdto.FeedVideoDisplay, len(videoLookup.Videos))
	for _, video := range videoLookup.Videos {
		videosByID[video.VideoID] = video
	}

	unitIDs := planUnitIDs(plan.Items)
	labelsByID := make(map[int64]string, len(unitIDs))
	if len(unitIDs) > 0 {
		labelLookup, err := s.labelLookup.Execute(ctx, catalogdto.UnitLabelLookupRequest{CoarseUnitIDs: unitIDs})
		if err != nil {
			return apvdto.FeedResponse{}, err
		}
		for _, label := range labelLookup.Labels {
			labelsByID[label.CoarseUnitID] = label.Text
		}
	}

	items := make([]apvdto.FeedItem, 0, len(plan.Items))
	for _, planItem := range plan.Items {
		item, err := s.buildFeedItem(plan.RunID, planItem, videosByID, labelsByID)
		if err != nil {
			return apvdto.FeedResponse{}, err
		}
		items = append(items, item)
	}

	return apvdto.FeedResponse{
		RecommendationRunID: plan.RunID,
		Items:               items,
	}, nil
}

func (s *FeedService) buildFeedItem(
	runID string,
	planItem recommendationdto.RecommendationPlanItem,
	videosByID map[string]catalogdto.FeedVideoDisplay,
	labelsByID map[int64]string,
) (apvdto.FeedItem, error) {
	if planItem.DurationMs <= 0 {
		err := fmt.Errorf("invalid duration_ms for recommendation feed item: run_id=%s video_id=%s duration_ms=%d", runID, planItem.VideoID, planItem.DurationMs)
		s.logger.Error("failed to materialize feed item", "run_id", runID, "video_id", planItem.VideoID, "duration_ms", planItem.DurationMs, "error", err)
		return apvdto.FeedItem{}, err
	}
	video, ok := videosByID[planItem.VideoID]
	if !ok {
		err := fmt.Errorf("missing feed video display data: run_id=%s video_id=%s", runID, planItem.VideoID)
		s.logger.Error("failed to materialize feed item", "run_id", runID, "video_id", planItem.VideoID, "error", err)
		return apvdto.FeedItem{}, err
	}
	coverURL, err := s.optionalAssetURL(video.CoverImageURL)
	if err != nil {
		wrapped := fmt.Errorf("build cover_image_url for recommendation feed item: run_id=%s video_id=%s: %w", runID, planItem.VideoID, err)
		s.logger.Error("failed to materialize feed item", "run_id", runID, "video_id", planItem.VideoID, "error", wrapped)
		return apvdto.FeedItem{}, wrapped
	}

	units := make([]apvdto.FeedLearningUnit, 0, len(planItem.LearningUnits))
	for _, unit := range planItem.LearningUnits {
		feedUnit, err := s.buildFeedLearningUnit(runID, planItem.VideoID, unit, labelsByID)
		if err != nil {
			return apvdto.FeedItem{}, err
		}
		units = append(units, feedUnit)
	}

	return apvdto.FeedItem{
		VideoID:         planItem.VideoID,
		Title:           video.Title,
		CoverImageURL:   coverURL,
		DurationSeconds: durationSeconds(planItem.DurationMs),
		ViewCount:       video.ViewCount,
		LearningUnits:   units,
	}, nil
}

func (s *FeedService) buildFeedLearningUnit(runID string, videoID string, unit recommendationdto.ExpectedLearningUnit, labelsByID map[int64]string) (apvdto.FeedLearningUnit, error) {
	evidence := unit.Evidence
	if evidence == nil || evidence.SentenceIndex == nil || evidence.SpanIndex == nil || evidence.StartMs == nil || evidence.EndMs == nil {
		err := fmt.Errorf("incomplete learning unit evidence: run_id=%s video_id=%s coarse_unit_id=%d", runID, videoID, unit.CoarseUnitID)
		s.logger.Error("failed to materialize feed learning unit", "run_id", runID, "video_id", videoID, "coarse_unit_id", unit.CoarseUnitID, "error", err)
		return apvdto.FeedLearningUnit{}, err
	}
	if *evidence.SentenceIndex < 0 || *evidence.SpanIndex < 0 || *evidence.StartMs < 0 || *evidence.EndMs < *evidence.StartMs {
		err := fmt.Errorf("invalid learning unit evidence: run_id=%s video_id=%s coarse_unit_id=%d", runID, videoID, unit.CoarseUnitID)
		s.logger.Error("failed to materialize feed learning unit", "run_id", runID, "video_id", videoID, "coarse_unit_id", unit.CoarseUnitID, "error", err)
		return apvdto.FeedLearningUnit{}, err
	}
	text, ok := labelsByID[unit.CoarseUnitID]
	if !ok || strings.TrimSpace(text) == "" {
		err := fmt.Errorf("missing unit label: run_id=%s video_id=%s coarse_unit_id=%d", runID, videoID, unit.CoarseUnitID)
		s.logger.Error("failed to materialize feed learning unit", "run_id", runID, "video_id", videoID, "coarse_unit_id", unit.CoarseUnitID, "error", err)
		return apvdto.FeedLearningUnit{}, err
	}
	return apvdto.FeedLearningUnit{
		CoarseUnitID:          unit.CoarseUnitID,
		Text:                  text,
		Role:                  unit.Role,
		IsPrimary:             unit.IsPrimary,
		EvidenceSentenceIndex: *evidence.SentenceIndex,
		EvidenceSpanIndex:     *evidence.SpanIndex,
		EvidenceStartMS:       *evidence.StartMs,
		EvidenceEndMS:         *evidence.EndMs,
	}, nil
}

func (s *FeedService) optionalAssetURL(path *string) (*string, error) {
	if path == nil || strings.TrimSpace(*path) == "" {
		return nil, nil
	}
	value, err := s.urlBuilder.Build(*path)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func planVideoIDs(items []recommendationdto.RecommendationPlanItem) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item.VideoID == "" {
			continue
		}
		if _, ok := seen[item.VideoID]; ok {
			continue
		}
		seen[item.VideoID] = struct{}{}
		result = append(result, item.VideoID)
	}
	return result
}

func planUnitIDs(items []recommendationdto.RecommendationPlanItem) []int64 {
	seen := make(map[int64]struct{})
	result := make([]int64, 0)
	for _, item := range items {
		for _, unit := range item.LearningUnits {
			if unit.CoarseUnitID == 0 {
				continue
			}
			if _, ok := seen[unit.CoarseUnitID]; ok {
				continue
			}
			seen[unit.CoarseUnitID] = struct{}{}
			result = append(result, unit.CoarseUnitID)
		}
	}
	return result
}

func durationSeconds(durationMs int32) int {
	return int((durationMs + 999) / 1000)
}

func isAbsoluteHTTPURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.IsAbs() && (parsed.Scheme == "http" || parsed.Scheme == "https")
}
