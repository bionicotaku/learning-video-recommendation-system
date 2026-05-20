package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	"learning-video-recommendation-system/internal/learningengine/reducer/application/service"
)

func TestListUserUnitProgressExecutePaginatesMasteredAndDecodesCursor(t *testing.T) {
	lastProgressAt := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	reader := &fakeUserUnitProgressReader{
		rows: []dto.UnitProgressItem{
			{CoarseUnitID: 101, Kind: "word", Label: "Abandon", LabelKey: "abandon", ProgressPercent: 100, LastProgressAt: &lastProgressAt},
			{CoarseUnitID: 102, Kind: "word", Label: "absorb", LabelKey: "absorb", ProgressPercent: 100},
		},
	}
	usecase := service.NewListUserUnitProgressUsecase(reader)

	firstPage, err := usecase.Execute(context.Background(), dto.ListUserUnitProgressRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Bucket: dto.UnitProgressBucketMastered,
		Limit:  1,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(firstPage.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(firstPage.Items))
	}
	if firstPage.Page.Limit != 1 || !firstPage.Page.HasMore || firstPage.Page.NextCursor == nil {
		t.Fatalf("page = %+v, want limit=1 has_more=true next_cursor", firstPage.Page)
	}
	if len(reader.queries) != 1 || reader.queries[0].LimitPlusOne != 2 || reader.queries[0].Cursor != nil {
		t.Fatalf("first query = %+v, want limit_plus_one=2 no cursor", reader.queries)
	}

	reader.rows = []dto.UnitProgressItem{}
	_, err = usecase.Execute(context.Background(), dto.ListUserUnitProgressRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Bucket: dto.UnitProgressBucketMastered,
		Limit:  1,
		Cursor: *firstPage.Page.NextCursor,
	})
	if err != nil {
		t.Fatalf("Execute() second page error = %v", err)
	}
	secondQuery := reader.queries[1]
	if secondQuery.Cursor == nil {
		t.Fatalf("second query cursor = nil")
	}
	if secondQuery.Cursor.Bucket != dto.UnitProgressBucketMastered ||
		secondQuery.Cursor.LabelKey != "abandon" ||
		secondQuery.Cursor.Label != "Abandon" ||
		secondQuery.Cursor.CoarseUnitID != 101 {
		t.Fatalf("decoded cursor = %+v, want mastered cursor for unit 101", secondQuery.Cursor)
	}
	if secondQuery.Cursor.HasProgressPercent {
		t.Fatalf("mastered cursor should not carry progress_percent: %+v", secondQuery.Cursor)
	}
}

func TestListUserUnitProgressExecutePaginatesUnmasteredWithProgressCursor(t *testing.T) {
	reader := &fakeUserUnitProgressReader{
		rows: []dto.UnitProgressItem{
			{CoarseUnitID: 201, Kind: "word", Label: "Constrain", LabelKey: "constrain", ProgressPercent: 64.25},
			{CoarseUnitID: 202, Kind: "word", Label: "derive", LabelKey: "derive", ProgressPercent: 20},
		},
	}
	usecase := service.NewListUserUnitProgressUsecase(reader)

	firstPage, err := usecase.Execute(context.Background(), dto.ListUserUnitProgressRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Bucket: dto.UnitProgressBucketUnmastered,
		Limit:  1,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if firstPage.Page.NextCursor == nil {
		t.Fatalf("next_cursor = nil, want cursor")
	}

	reader.rows = []dto.UnitProgressItem{}
	_, err = usecase.Execute(context.Background(), dto.ListUserUnitProgressRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Bucket: dto.UnitProgressBucketUnmastered,
		Limit:  1,
		Cursor: *firstPage.Page.NextCursor,
	})
	if err != nil {
		t.Fatalf("Execute() second page error = %v", err)
	}
	cursor := reader.queries[1].Cursor
	if cursor == nil {
		t.Fatalf("decoded cursor = nil")
	}
	if cursor.Bucket != dto.UnitProgressBucketUnmastered ||
		!cursor.HasProgressPercent ||
		cursor.ProgressPercent != 64.25 ||
		cursor.LabelKey != "constrain" ||
		cursor.Label != "Constrain" ||
		cursor.CoarseUnitID != 201 {
		t.Fatalf("decoded cursor = %+v, want unmastered cursor for unit 201", cursor)
	}
}

func TestListUserUnitProgressExecuteAppliesDefaultsAndNoNextCursorOnLastPage(t *testing.T) {
	reader := &fakeUserUnitProgressReader{
		rows: []dto.UnitProgressItem{
			{CoarseUnitID: 101, Kind: "word", Label: "abandon", LabelKey: "abandon", ProgressPercent: 100},
		},
	}
	usecase := service.NewListUserUnitProgressUsecase(reader)

	response, err := usecase.Execute(context.Background(), dto.ListUserUnitProgressRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Bucket: dto.UnitProgressBucketMastered,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.Page.Limit != 50 || response.Page.HasMore || response.Page.NextCursor != nil {
		t.Fatalf("page = %+v, want default limit and no next cursor", response.Page)
	}
	if len(reader.queries) != 1 || reader.queries[0].LimitPlusOne != 51 {
		t.Fatalf("query = %+v, want limit_plus_one=51", reader.queries)
	}
}

func TestListUserUnitProgressExecuteRejectsInvalidRequest(t *testing.T) {
	reader := &fakeUserUnitProgressReader{}
	usecase := service.NewListUserUnitProgressUsecase(reader)

	cases := []struct {
		name    string
		request dto.ListUserUnitProgressRequest
	}{
		{name: "missing user", request: dto.ListUserUnitProgressRequest{Bucket: dto.UnitProgressBucketMastered}},
		{name: "invalid bucket", request: dto.ListUserUnitProgressRequest{UserID: "11111111-1111-1111-1111-111111111111", Bucket: "all"}},
		{name: "limit below range", request: dto.ListUserUnitProgressRequest{UserID: "11111111-1111-1111-1111-111111111111", Bucket: dto.UnitProgressBucketMastered, Limit: -1}},
		{name: "limit above range", request: dto.ListUserUnitProgressRequest{UserID: "11111111-1111-1111-1111-111111111111", Bucket: dto.UnitProgressBucketMastered, Limit: 101}},
		{name: "malformed cursor", request: dto.ListUserUnitProgressRequest{UserID: "11111111-1111-1111-1111-111111111111", Bucket: dto.UnitProgressBucketMastered, Cursor: "not-base64"}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := usecase.Execute(context.Background(), tt.request)
			if !service.IsValidationError(err) {
				t.Fatalf("Execute() error = %v, want validation error", err)
			}
		})
	}
	if len(reader.queries) != 0 {
		t.Fatalf("reader should not be called for invalid requests")
	}
}

func TestListUserUnitProgressExecuteRejectsCursorBucketMismatch(t *testing.T) {
	reader := &fakeUserUnitProgressReader{
		rows: []dto.UnitProgressItem{
			{CoarseUnitID: 101, Kind: "word", Label: "abandon", LabelKey: "abandon", ProgressPercent: 100},
			{CoarseUnitID: 102, Kind: "word", Label: "absorb", LabelKey: "absorb", ProgressPercent: 100},
		},
	}
	usecase := service.NewListUserUnitProgressUsecase(reader)

	firstPage, err := usecase.Execute(context.Background(), dto.ListUserUnitProgressRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Bucket: dto.UnitProgressBucketMastered,
		Limit:  1,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if firstPage.Page.NextCursor == nil {
		t.Fatalf("next_cursor = nil, want cursor")
	}

	_, err = usecase.Execute(context.Background(), dto.ListUserUnitProgressRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Bucket: dto.UnitProgressBucketUnmastered,
		Limit:  1,
		Cursor: *firstPage.Page.NextCursor,
	})
	if !service.IsValidationError(err) {
		t.Fatalf("Execute() error = %v, want validation error", err)
	}
}

type fakeUserUnitProgressReader struct {
	rows    []dto.UnitProgressItem
	queries []dto.ListUserUnitProgressQuery
	err     error
}

func (f *fakeUserUnitProgressReader) ListUserUnitProgress(_ context.Context, query dto.ListUserUnitProgressQuery) ([]dto.UnitProgressItem, error) {
	f.queries = append(f.queries, query)
	if f.err != nil {
		return nil, f.err
	}
	return f.rows, nil
}

func TestListUserUnitProgressExecuteReturnsReaderError(t *testing.T) {
	readerErr := errors.New("db down")
	reader := &fakeUserUnitProgressReader{err: readerErr}
	usecase := service.NewListUserUnitProgressUsecase(reader)

	_, err := usecase.Execute(context.Background(), dto.ListUserUnitProgressRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Bucket: dto.UnitProgressBucketMastered,
	})
	if !errors.Is(err, readerErr) {
		t.Fatalf("Execute() error = %v, want reader error", err)
	}
}
