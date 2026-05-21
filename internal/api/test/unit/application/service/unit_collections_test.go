package service_test

import (
	"context"
	"testing"

	apivdto "learning-video-recommendation-system/internal/api/application/dto"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	semanticdto "learning-video-recommendation-system/internal/semantic/application/dto"
)

func TestUnitCollectionsServiceReturnsActiveCollectionSlugWhenItIsInActiveItems(t *testing.T) {
	active := "toefl-core"
	service := apiservice.NewUnitCollectionsService(
		&fakeSemanticCollections{response: semanticdto.ListUnitCollectionsResponse{
			Items: []semanticdto.UnitCollectionItem{{
				CollectionID: "11111111-1111-4111-8111-111111111111",
				Slug:         active,
				Name:         "TOEFL Core",
				Category:     "wordbook",
			}},
		}},
		&fakeActiveCollection{response: learningdto.GetActiveUnitCollectionResponse{
			ActiveCollection: &learningdto.ActiveUnitCollection{
				CollectionID:   "11111111-1111-4111-8111-111111111111",
				CollectionSlug: active,
			},
		}},
	)

	response, err := service.Execute(context.Background(), apivdto.ListUnitCollectionsRequest{
		UserID: "22222222-2222-4222-8222-222222222222",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.ActiveCollection == nil || *response.ActiveCollection != active {
		t.Fatalf("ActiveCollection = %v, want %q", response.ActiveCollection, active)
	}
	if len(response.Items) != 1 || response.Items[0].Slug != active {
		t.Fatalf("items not mapped: %+v", response.Items)
	}
}

func TestUnitCollectionsServiceReturnsNullActiveCollectionWhenProfileMissingOrNotActive(t *testing.T) {
	cases := []struct {
		name   string
		active *learningdto.ActiveUnitCollection
	}{
		{name: "missing profile"},
		{name: "inactive or missing collection", active: &learningdto.ActiveUnitCollection{
			CollectionID:   "33333333-3333-4333-8333-333333333333",
			CollectionSlug: "inactive-book",
		}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			service := apiservice.NewUnitCollectionsService(
				&fakeSemanticCollections{response: semanticdto.ListUnitCollectionsResponse{
					Items: []semanticdto.UnitCollectionItem{{
						CollectionID: "11111111-1111-4111-8111-111111111111",
						Slug:         "toefl-core",
						Name:         "TOEFL Core",
						Category:     "wordbook",
					}},
				}},
				&fakeActiveCollection{response: learningdto.GetActiveUnitCollectionResponse{ActiveCollection: tt.active}},
			)

			response, err := service.Execute(context.Background(), apivdto.ListUnitCollectionsRequest{
				UserID: "22222222-2222-4222-8222-222222222222",
			})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if response.ActiveCollection != nil {
				t.Fatalf("ActiveCollection = %v, want nil", *response.ActiveCollection)
			}
		})
	}
}

type fakeSemanticCollections struct {
	response semanticdto.ListUnitCollectionsResponse
	err      error
}

func (f *fakeSemanticCollections) Execute(context.Context) (semanticdto.ListUnitCollectionsResponse, error) {
	return f.response, f.err
}

type fakeActiveCollection struct {
	response learningdto.GetActiveUnitCollectionResponse
	err      error
}

func (f *fakeActiveCollection) Execute(context.Context, learningdto.GetActiveUnitCollectionRequest) (learningdto.GetActiveUnitCollectionResponse, error) {
	return f.response, f.err
}
