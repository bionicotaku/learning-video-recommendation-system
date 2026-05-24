package service_test

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/catalog/application/dto"
	apprepo "learning-video-recommendation-system/internal/catalog/application/repository"
	catalogservice "learning-video-recommendation-system/internal/catalog/application/service"
	"learning-video-recommendation-system/internal/catalog/domain/model"
)

func TestWordFavoriteStatusUsesCoarseKeyWhenVideoTranscriptIncludesCoarseUnit(t *testing.T) {
	repository := &fakeWordFavoriteRepository{favoritedByCoarse: true}
	usecase := catalogservice.NewGetWordFavoriteStatusUsecase(repository)

	response, err := usecase.Execute(context.Background(), dto.GetWordFavoriteStatusRequest{
		UserID:          validUserID,
		CoarseUnitID:    int64Ptr(108404),
		Text:            "Making",
		Source:          dto.WordFavoriteSourceVideoTranscript,
		VideoID:         stringPtr(validVideoID),
		SentenceIndex:   int32Ptr(7),
		TokenIndex:      int32Ptr(2),
		IncludeVideoCtx: false,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !response.IsFavorited {
		t.Fatal("expected favorite status from coarse key")
	}
	if len(repository.coarseStatusCalls) != 1 || len(repository.tokenStatusCalls) != 0 || repository.videoContextCalls != 0 {
		t.Fatalf("repository calls = %+v", repository)
	}
}

func TestWordFavoriteStatusReturnsVideoContextWithRequestIdentityFields(t *testing.T) {
	repository := &fakeWordFavoriteRepository{
		videoContext: model.WordFavoriteVideoContext{
			VideoTitle:          "Practice Makes Progress",
			VideoDurationMS:     120000,
			SentenceText:        "Making progress takes practice.",
			SentenceTranslation: stringPtr("取得进步需要练习。"),
			SentenceStartMS:     980,
			SentenceEndMS:       2800,
		},
	}
	usecase := catalogservice.NewGetWordFavoriteStatusUsecase(repository)

	response, err := usecase.Execute(context.Background(), dto.GetWordFavoriteStatusRequest{
		UserID:          validUserID,
		CoarseUnitID:    nil,
		Text:            "Making",
		Source:          dto.WordFavoriteSourceVideoTranscript,
		VideoID:         stringPtr(validVideoID),
		SentenceIndex:   int32Ptr(7),
		TokenIndex:      int32Ptr(2),
		IncludeVideoCtx: true,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.VideoContext == nil {
		t.Fatal("expected video context")
	}
	if response.VideoContext.VideoID != validVideoID || response.VideoContext.SentenceIndex != 7 || response.VideoContext.TokenIndex != 2 {
		t.Fatalf("video context identity fields = %+v", response.VideoContext)
	}
	if response.VideoContext.VideoTitle != "Practice Makes Progress" || response.VideoContext.SentenceText != "Making progress takes practice." {
		t.Fatalf("video context display fields = %+v", response.VideoContext)
	}
}

func TestSetWordFavoritePassesCoarseFavoriteOccurredAt(t *testing.T) {
	repository := &fakeWordFavoriteRepository{}
	usecase := catalogservice.NewSetWordFavoriteUsecase(repository)
	occurredAt := validOccurredAt()

	if err := usecase.Execute(context.Background(), dto.SetWordFavoriteRequest{
		UserID:       validUserID,
		CoarseUnitID: int64Ptr(108404),
		Text:         "Making",
		Source:       dto.WordFavoriteSourceWordList,
		OccurredAt:   occurredAt,
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(repository.setCoarseCalls) != 1 {
		t.Fatalf("set coarse calls = %+v", repository.setCoarseCalls)
	}
	if !repository.setCoarseCalls[0].OccurredAt.Equal(occurredAt) {
		t.Fatalf("occurred_at = %s, want %s", repository.setCoarseCalls[0].OccurredAt, occurredAt)
	}
}

func TestSetWordFavoriteIgnoresVideoFieldsForWordList(t *testing.T) {
	repository := &fakeWordFavoriteRepository{}
	usecase := catalogservice.NewSetWordFavoriteUsecase(repository)
	occurredAt := validOccurredAt()

	if err := usecase.Execute(context.Background(), dto.SetWordFavoriteRequest{
		UserID:        validUserID,
		CoarseUnitID:  int64Ptr(108404),
		Text:          "Making",
		Source:        dto.WordFavoriteSourceWordList,
		VideoID:       stringPtr("not-a-uuid"),
		SentenceIndex: int32Ptr(7),
		TokenIndex:    int32Ptr(2),
		OccurredAt:    occurredAt,
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(repository.setCoarseCalls) != 1 {
		t.Fatalf("set coarse calls = %+v", repository.setCoarseCalls)
	}
	call := repository.setCoarseCalls[0]
	if call.VideoID != nil || call.SentenceIndex != nil || call.TokenIndex != nil {
		t.Fatalf("word_list source fields = video_id:%v sentence:%v token:%v, want nil", call.VideoID, call.SentenceIndex, call.TokenIndex)
	}
	if !call.OccurredAt.Equal(occurredAt) {
		t.Fatalf("occurred_at = %s, want %s", call.OccurredAt, occurredAt)
	}
}

func TestSetWordFavoriteRejectsMissingOccurredAt(t *testing.T) {
	repository := &fakeWordFavoriteRepository{}
	usecase := catalogservice.NewSetWordFavoriteUsecase(repository)

	err := usecase.Execute(context.Background(), dto.SetWordFavoriteRequest{
		UserID:       validUserID,
		CoarseUnitID: int64Ptr(108404),
		Text:         "Making",
		Source:       dto.WordFavoriteSourceWordList,
	})
	if err == nil || !catalogservice.IsValidationError(err) {
		t.Fatalf("Execute() error = %v, want validation", err)
	}
	if len(repository.setCoarseCalls) != 0 {
		t.Fatalf("set calls = %+v, want none", repository.setCoarseCalls)
	}
}

func TestSetWordFavoriteUsesCoarseKeyAndCarriesSourceFieldsWithOccurredAt(t *testing.T) {
	repository := &fakeWordFavoriteRepository{}
	usecase := catalogservice.NewSetWordFavoriteUsecase(repository)
	occurredAt := validOccurredAt()

	if err := usecase.Execute(context.Background(), dto.SetWordFavoriteRequest{
		UserID:        validUserID,
		CoarseUnitID:  int64Ptr(108404),
		Text:          "Making",
		Source:        dto.WordFavoriteSourceVideoTranscript,
		VideoID:       stringPtr(validVideoID),
		SentenceIndex: int32Ptr(7),
		TokenIndex:    int32Ptr(2),
		OccurredAt:    occurredAt,
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(repository.setCoarseCalls) != 1 || len(repository.setTokenCalls) != 0 {
		t.Fatalf("set calls = coarse:%+v token:%+v", repository.setCoarseCalls, repository.setTokenCalls)
	}
	call := repository.setCoarseCalls[0]
	if call.VideoID == nil || *call.VideoID != validVideoID || call.SentenceIndex == nil || *call.SentenceIndex != 7 || call.TokenIndex == nil || *call.TokenIndex != 2 {
		t.Fatalf("source fields = %+v", call)
	}
	if !call.OccurredAt.Equal(occurredAt) {
		t.Fatalf("occurred_at = %s, want %s", call.OccurredAt, occurredAt)
	}
}

func TestSetWordFavoriteMapsRepositoryTargetNotFound(t *testing.T) {
	repository := &fakeWordFavoriteRepository{setCoarseOutcome: apprepo.WordFavoriteWriteTargetNotFound}
	usecase := catalogservice.NewSetWordFavoriteUsecase(repository)

	err := usecase.Execute(context.Background(), dto.SetWordFavoriteRequest{
		UserID:       validUserID,
		CoarseUnitID: int64Ptr(108404),
		Text:         "Making",
		Source:       dto.WordFavoriteSourceWordList,
		OccurredAt:   validOccurredAt(),
	})
	if err == nil || !catalogservice.IsNotFoundError(err) {
		t.Fatalf("Execute() error = %v, want not found", err)
	}
}

func TestUnsetWordFavoriteRejectsMissingOccurredAt(t *testing.T) {
	repository := &fakeWordFavoriteRepository{}
	usecase := catalogservice.NewUnsetWordFavoriteUsecase(repository)

	err := usecase.Execute(context.Background(), dto.UnsetWordFavoriteRequest{
		UserID:       validUserID,
		CoarseUnitID: int64Ptr(108404),
		Text:         "Making",
		Source:       dto.WordFavoriteSourceWordList,
	})
	if err == nil || !catalogservice.IsValidationError(err) {
		t.Fatalf("Execute() error = %v, want validation", err)
	}
	if len(repository.unsetCoarseCalls) != 0 {
		t.Fatalf("unset calls = %+v, want none", repository.unsetCoarseCalls)
	}
}

func TestUnsetWordFavoriteDoesNotValidateTargetExistence(t *testing.T) {
	repository := &fakeWordFavoriteRepository{}
	usecase := catalogservice.NewUnsetWordFavoriteUsecase(repository)
	occurredAt := validOccurredAt()

	if err := usecase.Execute(context.Background(), dto.UnsetWordFavoriteRequest{
		UserID:       validUserID,
		CoarseUnitID: int64Ptr(108404),
		Text:         "Making",
		Source:       dto.WordFavoriteSourceWordList,
		OccurredAt:   occurredAt,
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(repository.unsetCoarseCalls) != 1 {
		t.Fatalf("unset coarse calls = %+v", repository.unsetCoarseCalls)
	}
	if !repository.unsetCoarseCalls[0].OccurredAt.Equal(occurredAt) {
		t.Fatalf("occurred_at = %s, want %s", repository.unsetCoarseCalls[0].OccurredAt, occurredAt)
	}
}

func TestListWordFavoritesDefaultsAndRejectsWrongCursorKind(t *testing.T) {
	repository := &fakeWordFavoriteRepository{}
	usecase := catalogservice.NewListWordFavoritesUsecase(repository)

	response, err := usecase.Execute(context.Background(), dto.ListWordFavoritesRequest{UserID: validUserID})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.Page.Limit != 50 || response.Page.HasMore || response.Page.NextCursor != nil {
		t.Fatalf("page = %+v, want default terminal page", response.Page)
	}
	if len(repository.listQueries) != 1 || repository.listQueries[0].LimitPlusOne != 51 {
		t.Fatalf("list queries = %+v", repository.listQueries)
	}

	wrongCursor := base64.RawURLEncoding.EncodeToString([]byte(`{"kind":"video_favorites","favorited_at":"2026-05-24T10:20:30Z","favorite_id":"00000000-0000-4000-8000-000000000001"}`))
	if _, err := usecase.Execute(context.Background(), dto.ListWordFavoritesRequest{UserID: validUserID, Cursor: wrongCursor}); err == nil || !catalogservice.IsValidationError(err) {
		t.Fatalf("wrong cursor error = %v, want validation", err)
	}
}

type fakeWordFavoriteRepository struct {
	favoritedByCoarse bool
	favoritedByToken  bool
	videoContext      model.WordFavoriteVideoContext
	setCoarseOutcome  apprepo.WordFavoriteWriteOutcome
	setTokenOutcome   apprepo.WordFavoriteWriteOutcome

	coarseStatusCalls []model.WordFavoriteCoarseKey
	tokenStatusCalls  []model.WordFavoriteTokenKey
	videoContextCalls int
	setCoarseCalls    []model.SetCoarseWordFavoriteCommand
	setTokenCalls     []model.SetTokenWordFavoriteCommand
	unsetCoarseCalls  []model.UnsetCoarseWordFavoriteCommand
	unsetTokenCalls   []model.UnsetTokenWordFavoriteCommand
	listQueries       []dto.ListWordFavoritesQuery
}

func (f *fakeWordFavoriteRepository) HasCoarseFavorite(ctx context.Context, key model.WordFavoriteCoarseKey) (bool, error) {
	f.coarseStatusCalls = append(f.coarseStatusCalls, key)
	return f.favoritedByCoarse, nil
}

func (f *fakeWordFavoriteRepository) HasTokenFavorite(ctx context.Context, key model.WordFavoriteTokenKey) (bool, error) {
	f.tokenStatusCalls = append(f.tokenStatusCalls, key)
	return f.favoritedByToken, nil
}

func (f *fakeWordFavoriteRepository) GetVideoContext(ctx context.Context, key model.WordFavoriteVideoContextKey) (model.WordFavoriteVideoContext, error) {
	f.videoContextCalls++
	return f.videoContext, nil
}

func (f *fakeWordFavoriteRepository) SetCoarseFavorite(ctx context.Context, command model.SetCoarseWordFavoriteCommand) (apprepo.WordFavoriteWriteOutcome, error) {
	f.setCoarseCalls = append(f.setCoarseCalls, command)
	if f.setCoarseOutcome != "" {
		return f.setCoarseOutcome, nil
	}
	return apprepo.WordFavoriteWriteApplied, nil
}

func (f *fakeWordFavoriteRepository) SetTokenFavorite(ctx context.Context, command model.SetTokenWordFavoriteCommand) (apprepo.WordFavoriteWriteOutcome, error) {
	f.setTokenCalls = append(f.setTokenCalls, command)
	if f.setTokenOutcome != "" {
		return f.setTokenOutcome, nil
	}
	return apprepo.WordFavoriteWriteApplied, nil
}

func (f *fakeWordFavoriteRepository) UnsetCoarseFavorite(ctx context.Context, command model.UnsetCoarseWordFavoriteCommand) error {
	f.unsetCoarseCalls = append(f.unsetCoarseCalls, command)
	return nil
}

func (f *fakeWordFavoriteRepository) UnsetTokenFavorite(ctx context.Context, command model.UnsetTokenWordFavoriteCommand) error {
	f.unsetTokenCalls = append(f.unsetTokenCalls, command)
	return nil
}

func (f *fakeWordFavoriteRepository) ListWordFavorites(ctx context.Context, query dto.ListWordFavoritesQuery) ([]model.WordFavoriteListItem, error) {
	f.listQueries = append(f.listQueries, query)
	return nil, nil
}

const (
	validUserID  = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	validVideoID = "00000000-0000-4000-8000-000000000001"
)

func int64Ptr(value int64) *int64 {
	return &value
}

func int32Ptr(value int32) *int32 {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func validOccurredAt() time.Time {
	return time.Date(2026, 5, 24, 10, 20, 30, 123000000, time.UTC)
}
