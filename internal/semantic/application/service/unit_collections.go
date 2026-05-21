package service

import (
	"context"
	"errors"

	"learning-video-recommendation-system/internal/semantic/application/dto"
	"learning-video-recommendation-system/internal/semantic/application/repository"
)

type ListUnitCollectionsUsecase struct {
	reader repository.UnitCollectionReader
}

func NewListUnitCollectionsUsecase(reader repository.UnitCollectionReader) *ListUnitCollectionsUsecase {
	return &ListUnitCollectionsUsecase{reader: reader}
}

func (u *ListUnitCollectionsUsecase) Execute(ctx context.Context) (dto.ListUnitCollectionsResponse, error) {
	if u.reader == nil {
		return dto.ListUnitCollectionsResponse{}, errors.New("unit collection reader is required")
	}
	collections, err := u.reader.ListActiveUnitCollections(ctx)
	if err != nil {
		return dto.ListUnitCollectionsResponse{}, err
	}
	items := make([]dto.UnitCollectionItem, 0, len(collections))
	for _, collection := range collections {
		items = append(items, dto.UnitCollectionItem{
			CollectionID:    collection.CollectionID,
			Slug:            collection.Slug,
			Name:            collection.Name,
			Description:     collection.Description,
			Category:        collection.Category,
			CoarseUnitCount: collection.CoarseUnitCount,
			WordUnitCount:   collection.WordUnitCount,
		})
	}
	return dto.ListUnitCollectionsResponse{Items: items}, nil
}
