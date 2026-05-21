package service

import (
	"context"
	"fmt"

	apivdto "learning-video-recommendation-system/internal/api/application/dto"
	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	learningusecase "learning-video-recommendation-system/internal/learningengine/reducer/application/usecase"
	semanticdto "learning-video-recommendation-system/internal/semantic/application/dto"
	semanticusecase "learning-video-recommendation-system/internal/semantic/application/usecase"
)

type UnitCollectionsService struct {
	listCollections      semanticusecase.ListUnitCollectionsUsecase
	activeCollectionRead learningusecase.GetActiveUnitCollectionUsecase
}

func NewUnitCollectionsService(
	listCollections semanticusecase.ListUnitCollectionsUsecase,
	activeCollectionRead learningusecase.GetActiveUnitCollectionUsecase,
) *UnitCollectionsService {
	return &UnitCollectionsService{
		listCollections:      listCollections,
		activeCollectionRead: activeCollectionRead,
	}
}

func (s *UnitCollectionsService) Execute(ctx context.Context, request apivdto.ListUnitCollectionsRequest) (apivdto.UnitCollectionsResponse, error) {
	if request.UserID == "" {
		return apivdto.UnitCollectionsResponse{}, InvalidRequestError("user_id is required")
	}
	if s.listCollections == nil {
		return apivdto.UnitCollectionsResponse{}, fmt.Errorf("unit collection list usecase is required")
	}
	if s.activeCollectionRead == nil {
		return apivdto.UnitCollectionsResponse{}, fmt.Errorf("active collection read usecase is required")
	}

	collections, err := s.listCollections.Execute(ctx)
	if err != nil {
		return apivdto.UnitCollectionsResponse{}, err
	}
	active, err := s.activeCollectionRead.Execute(ctx, learningdto.GetActiveUnitCollectionRequest{UserID: request.UserID})
	if err != nil {
		return apivdto.UnitCollectionsResponse{}, err
	}

	items := convertUnitCollectionItems(collections.Items)
	return apivdto.UnitCollectionsResponse{
		Items:            items,
		ActiveCollection: activeCollectionSlug(items, active.ActiveCollection),
	}, nil
}

func convertUnitCollectionItems(items []semanticdto.UnitCollectionItem) []apivdto.UnitCollectionItem {
	result := make([]apivdto.UnitCollectionItem, 0, len(items))
	for _, item := range items {
		result = append(result, apivdto.UnitCollectionItem{
			CollectionID:    item.CollectionID,
			Slug:            item.Slug,
			Name:            item.Name,
			Description:     item.Description,
			Category:        item.Category,
			CoarseUnitCount: item.CoarseUnitCount,
			WordUnitCount:   item.WordUnitCount,
		})
	}
	return result
}

func activeCollectionSlug(items []apivdto.UnitCollectionItem, active *learningdto.ActiveUnitCollection) *string {
	if active == nil {
		return nil
	}
	for _, item := range items {
		if item.CollectionID == active.CollectionID && item.Slug == active.CollectionSlug {
			slug := item.Slug
			return &slug
		}
	}
	return nil
}
