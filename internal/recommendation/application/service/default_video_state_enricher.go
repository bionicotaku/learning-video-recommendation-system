package service

import (
	"context"
	"sort"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type DefaultVideoStateEnricher struct {
	videoServing   apprepo.VideoServingStateRepository
	videoUserState apprepo.VideoUserStateReader
}

var _ VideoStateEnricher = (*DefaultVideoStateEnricher)(nil)

func NewDefaultVideoStateEnricher(
	videoServing apprepo.VideoServingStateRepository,
	videoUserState apprepo.VideoUserStateReader,
) *DefaultVideoStateEnricher {
	return &DefaultVideoStateEnricher{
		videoServing:   videoServing,
		videoUserState: videoUserState,
	}
}

func (e *DefaultVideoStateEnricher) Enrich(ctx context.Context, contextModel model.RecommendationContext, videos []model.VideoCandidate) (model.RecommendationContext, error) {
	if len(videos) == 0 {
		return contextModel, nil
	}

	videoIDs := uniqueVideoIDs(videos)
	if len(videoIDs) == 0 {
		return contextModel, nil
	}

	if e.videoServing != nil {
		videoServingStates, err := e.videoServing.ListByUserAndVideoIDs(ctx, contextModel.Request.UserID, videoIDs)
		if err != nil {
			return model.RecommendationContext{}, err
		}
		contextModel.VideoServingStates = videoServingStates
	}
	if e.videoUserState != nil {
		videoUserStates, err := e.videoUserState.ListByUserAndVideoIDs(ctx, contextModel.Request.UserID, videoIDs)
		if err != nil {
			return model.RecommendationContext{}, err
		}
		contextModel.VideoUserStates = videoUserStates
	}

	return contextModel, nil
}

func uniqueVideoIDs(videos []model.VideoCandidate) []string {
	seen := make(map[string]struct{}, len(videos))
	result := make([]string, 0, len(videos))
	for _, video := range videos {
		if _, ok := seen[video.VideoID]; ok {
			continue
		}
		seen[video.VideoID] = struct{}{}
		result = append(result, video.VideoID)
	}
	sort.Strings(result)
	return result
}
