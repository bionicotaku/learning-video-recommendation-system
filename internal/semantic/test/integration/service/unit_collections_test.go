//go:build integration

package service_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	semanticservice "learning-video-recommendation-system/internal/semantic/application/service"
	semanticrepo "learning-video-recommendation-system/internal/semantic/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/semantic/test/fixture"
)

var sharedSuite *fixture.Suite

func TestMain(m *testing.M) {
	suite, err := fixture.OpenSuite()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "open semantic integration suite: %v\n", err)
		os.Exit(1)
	}
	sharedSuite = suite

	code := m.Run()
	if err := suite.Close(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "close semantic integration suite: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

func TestListUnitCollectionsReturnsActiveCollectionsInStableOrder(t *testing.T) {
	t.Parallel()

	db := sharedSuite.CreateTestDatabase(t)
	db.SeedUnitCollection(t, fixture.UnitCollectionSeed{
		CollectionID:    "33333333-3333-4333-8333-333333333333",
		Slug:            "z-inactive",
		Name:            "Inactive",
		Category:        "wordbook",
		Status:          "inactive",
		CoarseUnitCount: 99,
		WordUnitCount:   99,
	})
	db.SeedUnitCollection(t, fixture.UnitCollectionSeed{
		CollectionID:    "22222222-2222-4222-8222-222222222222",
		Slug:            "toefl-1000-essential",
		Name:            "TOEFL 1000 Essential",
		Description:     stringPtr("Core TOEFL vocabulary."),
		Category:        "wordbook",
		Status:          "active",
		CoarseUnitCount: 1000,
		WordUnitCount:   1000,
	})
	db.SeedUnitCollection(t, fixture.UnitCollectionSeed{
		CollectionID:    "11111111-1111-4111-8111-111111111111",
		Slug:            "ielts-core",
		Name:            "IELTS Core",
		Category:        "wordbook",
		Status:          "active",
		CoarseUnitCount: 1800,
		WordUnitCount:   1640,
	})

	usecase := semanticservice.NewListUnitCollectionsUsecase(semanticrepo.NewUnitCollectionReader(db.Pool))
	response, err := usecase.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(response.Items) != 2 {
		t.Fatalf("items len = %d, want 2: %+v", len(response.Items), response.Items)
	}
	if response.Items[0].Slug != "ielts-core" || response.Items[1].Slug != "toefl-1000-essential" {
		t.Fatalf("unexpected order/items: %+v", response.Items)
	}
	if response.Items[1].Description == nil || *response.Items[1].Description != "Core TOEFL vocabulary." {
		t.Fatalf("description not mapped: %+v", response.Items[1])
	}
	if response.Items[0].CoarseUnitCount != 1800 || response.Items[0].WordUnitCount != 1640 {
		t.Fatalf("count fields not mapped: %+v", response.Items[0])
	}
}

func stringPtr(value string) *string {
	return &value
}
