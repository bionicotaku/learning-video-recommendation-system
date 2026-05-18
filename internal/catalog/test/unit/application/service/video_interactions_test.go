package service_test

import (
	"context"
	"errors"
	"testing"

	"learning-video-recommendation-system/internal/catalog/application/dto"
	apprepo "learning-video-recommendation-system/internal/catalog/application/repository"
	"learning-video-recommendation-system/internal/catalog/application/service"
	"learning-video-recommendation-system/internal/catalog/domain/model"
)

func TestVideoInteractionUsecasesValidateRequiredInput(t *testing.T) {
	t.Run("like requires user id", func(t *testing.T) {
		writer := &fakeVideoInteractionWriter{}
		usecase := service.NewSetVideoLikeUsecase(writer)

		_, err := usecase.Execute(context.Background(), dto.SetVideoLikeRequest{VideoID: videoID, Enabled: true})
		if err == nil || !service.IsValidationError(err) {
			t.Fatalf("expected validation error, got %T %v", err, err)
		}
		if writer.likeCalled {
			t.Fatal("writer should not be called")
		}
	})

	t.Run("favorite requires video id", func(t *testing.T) {
		writer := &fakeVideoInteractionWriter{}
		usecase := service.NewSetVideoFavoriteUsecase(writer)

		_, err := usecase.Execute(context.Background(), dto.SetVideoFavoriteRequest{UserID: userID, Enabled: true})
		if err == nil || !service.IsValidationError(err) {
			t.Fatalf("expected validation error, got %T %v", err, err)
		}
		if writer.favoriteCalled {
			t.Fatal("writer should not be called")
		}
	})
}

func TestSetVideoLikeUsecaseMapsCommandAndResponse(t *testing.T) {
	writer := &fakeVideoInteractionWriter{
		likeResult: model.VideoLikeResult{
			VideoID:   videoID,
			HasLiked:  true,
			LikeCount: 86,
		},
	}
	usecase := service.NewSetVideoLikeUsecase(writer)

	response, err := usecase.Execute(context.Background(), dto.SetVideoLikeRequest{
		UserID:  userID,
		VideoID: videoID,
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !writer.likeCalled || writer.likeCommand.UserID != userID || writer.likeCommand.VideoID != videoID || !writer.likeCommand.Enabled {
		t.Fatalf("unexpected writer command: %+v", writer.likeCommand)
	}
	if response.VideoID != videoID || !response.HasLiked || response.LikeCount != 86 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestSetVideoFavoriteUsecaseMapsCommandAndResponse(t *testing.T) {
	writer := &fakeVideoInteractionWriter{
		favoriteResult: model.VideoFavoriteResult{
			VideoID:       videoID,
			HasFavorited:  false,
			FavoriteCount: 17,
		},
	}
	usecase := service.NewSetVideoFavoriteUsecase(writer)

	response, err := usecase.Execute(context.Background(), dto.SetVideoFavoriteRequest{
		UserID:  userID,
		VideoID: videoID,
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !writer.favoriteCalled || writer.favoriteCommand.UserID != userID || writer.favoriteCommand.VideoID != videoID || writer.favoriteCommand.Enabled {
		t.Fatalf("unexpected writer command: %+v", writer.favoriteCommand)
	}
	if response.VideoID != videoID || response.HasFavorited || response.FavoriteCount != 17 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestVideoInteractionUsecasesMapRepositoryNotFound(t *testing.T) {
	t.Run("like", func(t *testing.T) {
		usecase := service.NewSetVideoLikeUsecase(&fakeVideoInteractionWriter{err: apprepo.ErrVideoNotFound})

		_, err := usecase.Execute(context.Background(), dto.SetVideoLikeRequest{UserID: userID, VideoID: videoID, Enabled: true})
		if err == nil || !service.IsNotFoundError(err) {
			t.Fatalf("expected not found error, got %T %v", err, err)
		}
	})

	t.Run("favorite", func(t *testing.T) {
		usecase := service.NewSetVideoFavoriteUsecase(&fakeVideoInteractionWriter{err: apprepo.ErrVideoNotFound})

		_, err := usecase.Execute(context.Background(), dto.SetVideoFavoriteRequest{UserID: userID, VideoID: videoID, Enabled: true})
		if err == nil || !service.IsNotFoundError(err) {
			t.Fatalf("expected not found error, got %T %v", err, err)
		}
	})
}

func TestVideoInteractionUsecasesPropagateUnexpectedErrors(t *testing.T) {
	usecase := service.NewSetVideoLikeUsecase(&fakeVideoInteractionWriter{err: errors.New("db down")})

	_, err := usecase.Execute(context.Background(), dto.SetVideoLikeRequest{UserID: userID, VideoID: videoID, Enabled: true})
	if err == nil || service.IsNotFoundError(err) || service.IsValidationError(err) {
		t.Fatalf("expected raw unexpected error, got %T %v", err, err)
	}
}

type fakeVideoInteractionWriter struct {
	likeCalled      bool
	favoriteCalled  bool
	likeCommand     model.VideoLikeCommand
	favoriteCommand model.VideoFavoriteCommand
	likeResult      model.VideoLikeResult
	favoriteResult  model.VideoFavoriteResult
	err             error
}

func (f *fakeVideoInteractionWriter) SetVideoLike(ctx context.Context, command model.VideoLikeCommand) (model.VideoLikeResult, error) {
	f.likeCalled = true
	f.likeCommand = command
	return f.likeResult, f.err
}

func (f *fakeVideoInteractionWriter) SetVideoFavorite(ctx context.Context, command model.VideoFavoriteCommand) (model.VideoFavoriteResult, error) {
	f.favoriteCalled = true
	f.favoriteCommand = command
	return f.favoriteResult, f.err
}

const (
	userID  = "11111111-1111-1111-1111-111111111111"
	videoID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
)
