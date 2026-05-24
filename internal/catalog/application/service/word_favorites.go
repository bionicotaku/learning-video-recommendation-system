package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"learning-video-recommendation-system/internal/catalog/application/dto"
	apprepo "learning-video-recommendation-system/internal/catalog/application/repository"
	"learning-video-recommendation-system/internal/catalog/domain/model"
)

const (
	defaultWordFavoritesLimit = 50
	maxWordFavoritesLimit     = 100
)

type GetWordFavoriteStatusUsecase struct {
	repository apprepo.WordFavoriteRepository
}

func NewGetWordFavoriteStatusUsecase(repository apprepo.WordFavoriteRepository) *GetWordFavoriteStatusUsecase {
	return &GetWordFavoriteStatusUsecase{repository: repository}
}

func (u *GetWordFavoriteStatusUsecase) Execute(ctx context.Context, request dto.GetWordFavoriteStatusRequest) (dto.WordFavoriteStatusResponse, error) {
	if u.repository == nil {
		return dto.WordFavoriteStatusResponse{}, fmt.Errorf("word favorite repository is required")
	}
	identity, err := normalizeWordFavoriteIdentity(wordFavoriteIdentityInput{
		UserID:        request.UserID,
		CoarseUnitID:  request.CoarseUnitID,
		Text:          request.Text,
		Source:        request.Source,
		VideoID:       request.VideoID,
		SentenceIndex: request.SentenceIndex,
		TokenIndex:    request.TokenIndex,
	})
	if err != nil {
		return dto.WordFavoriteStatusResponse{}, err
	}
	if request.IncludeVideoCtx && identity.Source != dto.WordFavoriteSourceVideoTranscript {
		return dto.WordFavoriteStatusResponse{}, validationError("include_video_context requires video_transcript source")
	}

	isFavorited, err := u.isFavorited(ctx, identity)
	if err != nil {
		return dto.WordFavoriteStatusResponse{}, err
	}
	response := dto.WordFavoriteStatusResponse{IsFavorited: isFavorited}
	if request.IncludeVideoCtx {
		context, err := u.repository.GetVideoContext(ctx, model.WordFavoriteVideoContextKey{
			VideoID:       *identity.VideoID,
			SentenceIndex: *identity.SentenceIndex,
		})
		if err != nil {
			return dto.WordFavoriteStatusResponse{}, classifyWordFavoriteRepositoryError(err, "video context not found")
		}
		response.VideoContext = &dto.WordFavoriteVideoContext{
			VideoID:             *identity.VideoID,
			VideoTitle:          context.VideoTitle,
			VideoDurationMS:     int32Pointer(context.VideoDurationMS),
			TokenIndex:          *identity.TokenIndex,
			SentenceIndex:       *identity.SentenceIndex,
			SentenceText:        context.SentenceText,
			SentenceTranslation: context.SentenceTranslation,
			SentenceStartMS:     int32Pointer(context.SentenceStartMS),
			SentenceEndMS:       int32Pointer(context.SentenceEndMS),
		}
	}
	return response, nil
}

func (u *GetWordFavoriteStatusUsecase) isFavorited(ctx context.Context, identity normalizedWordFavoriteIdentity) (bool, error) {
	if identity.CoarseUnitID != nil {
		return u.repository.HasCoarseFavorite(ctx, model.WordFavoriteCoarseKey{
			UserID:       identity.UserID,
			CoarseUnitID: *identity.CoarseUnitID,
		})
	}
	return u.repository.HasTokenFavorite(ctx, identity.tokenKey())
}

type SetWordFavoriteUsecase struct {
	repository apprepo.WordFavoriteRepository
}

func NewSetWordFavoriteUsecase(repository apprepo.WordFavoriteRepository) *SetWordFavoriteUsecase {
	return &SetWordFavoriteUsecase{repository: repository}
}

func (u *SetWordFavoriteUsecase) Execute(ctx context.Context, request dto.SetWordFavoriteRequest) error {
	if u.repository == nil {
		return fmt.Errorf("word favorite repository is required")
	}
	identity, err := normalizeWordFavoriteIdentity(wordFavoriteIdentityInput{
		UserID:        request.UserID,
		CoarseUnitID:  request.CoarseUnitID,
		Text:          request.Text,
		Source:        request.Source,
		VideoID:       request.VideoID,
		SentenceIndex: request.SentenceIndex,
		TokenIndex:    request.TokenIndex,
	})
	if err != nil {
		return err
	}
	if request.OccurredAt.IsZero() {
		return validationError("occurred_at is required")
	}
	occurredAt := request.OccurredAt.UTC()

	if identity.CoarseUnitID != nil {
		outcome, err := u.repository.SetCoarseFavorite(ctx, model.SetCoarseWordFavoriteCommand{
			UserID:        identity.UserID,
			CoarseUnitID:  *identity.CoarseUnitID,
			Source:        identity.Source,
			VideoID:       identity.VideoID,
			SentenceIndex: identity.SentenceIndex,
			TokenIndex:    identity.TokenIndex,
			OccurredAt:    occurredAt,
		})
		return mapWordFavoriteWriteOutcome(outcome, err, "coarse unit not found")
	}

	outcome, err := u.repository.SetTokenFavorite(ctx, model.SetTokenWordFavoriteCommand{
		UserID:        identity.UserID,
		VideoID:       *identity.VideoID,
		SentenceIndex: *identity.SentenceIndex,
		TokenIndex:    *identity.TokenIndex,
		OccurredAt:    occurredAt,
	})
	return mapWordFavoriteWriteOutcome(outcome, err, "video token not found")
}

func mapWordFavoriteWriteOutcome(outcome apprepo.WordFavoriteWriteOutcome, err error, notFoundMessage string) error {
	if err != nil {
		return err
	}
	switch outcome {
	case apprepo.WordFavoriteWriteApplied, apprepo.WordFavoriteWriteStale:
		return nil
	case apprepo.WordFavoriteWriteTargetNotFound:
		return NotFoundError(notFoundMessage)
	default:
		return fmt.Errorf("unknown word favorite write outcome: %s", outcome)
	}
}

type UnsetWordFavoriteUsecase struct {
	repository apprepo.WordFavoriteRepository
}

func NewUnsetWordFavoriteUsecase(repository apprepo.WordFavoriteRepository) *UnsetWordFavoriteUsecase {
	return &UnsetWordFavoriteUsecase{repository: repository}
}

func (u *UnsetWordFavoriteUsecase) Execute(ctx context.Context, request dto.UnsetWordFavoriteRequest) error {
	if u.repository == nil {
		return fmt.Errorf("word favorite repository is required")
	}
	identity, err := normalizeWordFavoriteIdentity(wordFavoriteIdentityInput{
		UserID:        request.UserID,
		CoarseUnitID:  request.CoarseUnitID,
		Text:          request.Text,
		Source:        request.Source,
		VideoID:       request.VideoID,
		SentenceIndex: request.SentenceIndex,
		TokenIndex:    request.TokenIndex,
	})
	if err != nil {
		return err
	}
	if request.OccurredAt.IsZero() {
		return validationError("occurred_at is required")
	}
	occurredAt := request.OccurredAt.UTC()
	if identity.CoarseUnitID != nil {
		return u.repository.UnsetCoarseFavorite(ctx, model.UnsetCoarseWordFavoriteCommand{
			UserID:        identity.UserID,
			CoarseUnitID:  *identity.CoarseUnitID,
			Source:        identity.Source,
			VideoID:       identity.VideoID,
			SentenceIndex: identity.SentenceIndex,
			TokenIndex:    identity.TokenIndex,
			OccurredAt:    occurredAt,
		})
	}
	return u.repository.UnsetTokenFavorite(ctx, model.UnsetTokenWordFavoriteCommand{
		UserID:        identity.UserID,
		VideoID:       *identity.VideoID,
		SentenceIndex: *identity.SentenceIndex,
		TokenIndex:    *identity.TokenIndex,
		OccurredAt:    occurredAt,
	})
}

type ListWordFavoritesUsecase struct {
	repository apprepo.WordFavoriteRepository
}

func NewListWordFavoritesUsecase(repository apprepo.WordFavoriteRepository) *ListWordFavoritesUsecase {
	return &ListWordFavoritesUsecase{repository: repository}
}

func (u *ListWordFavoritesUsecase) Execute(ctx context.Context, request dto.ListWordFavoritesRequest) (dto.WordFavoriteListPage, error) {
	userID := strings.TrimSpace(request.UserID)
	if userID == "" {
		return dto.WordFavoriteListPage{}, validationError("user_id is required")
	}
	if !isUUID(userID) {
		return dto.WordFavoriteListPage{}, validationError("user_id must be a uuid")
	}
	if u.repository == nil {
		return dto.WordFavoriteListPage{}, fmt.Errorf("word favorite repository is required")
	}
	limit, err := normalizeWordFavoritesLimit(request.Limit)
	if err != nil {
		return dto.WordFavoriteListPage{}, err
	}
	cursor, err := decodeWordFavoritesCursor(request.Cursor)
	if err != nil {
		return dto.WordFavoriteListPage{}, err
	}
	rows, err := u.repository.ListWordFavorites(ctx, dto.ListWordFavoritesQuery{
		UserID:       userID,
		LimitPlusOne: limit + 1,
		Cursor:       cursor,
	})
	if err != nil {
		return dto.WordFavoriteListPage{}, err
	}

	hasMore := len(rows) > limit
	items := rows
	if hasMore {
		items = rows[:limit]
	}
	if items == nil {
		items = []model.WordFavoriteListItem{}
	}

	var nextCursor *string
	if hasMore && len(items) > 0 {
		encoded, err := encodeWordFavoritesCursor(items[len(items)-1])
		if err != nil {
			return dto.WordFavoriteListPage{}, fmt.Errorf("encode word favorites cursor: %w", err)
		}
		nextCursor = &encoded
	}

	return dto.WordFavoriteListPage{
		Items: mapWordFavoriteListItems(items),
		Page: dto.WordFavoritePage{
			Limit:      limit,
			HasMore:    hasMore,
			NextCursor: nextCursor,
		},
	}, nil
}

type wordFavoriteIdentityInput struct {
	UserID        string
	CoarseUnitID  *int64
	Text          string
	Source        string
	VideoID       *string
	SentenceIndex *int32
	TokenIndex    *int32
}

type normalizedWordFavoriteIdentity struct {
	UserID        string
	CoarseUnitID  *int64
	Text          string
	Source        string
	VideoID       *string
	SentenceIndex *int32
	TokenIndex    *int32
}

func normalizeWordFavoriteIdentity(input wordFavoriteIdentityInput) (normalizedWordFavoriteIdentity, error) {
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return normalizedWordFavoriteIdentity{}, validationError("user_id is required")
	}
	if !isUUID(userID) {
		return normalizedWordFavoriteIdentity{}, validationError("user_id must be a uuid")
	}
	if strings.TrimSpace(input.Text) == "" {
		return normalizedWordFavoriteIdentity{}, validationError("text is required")
	}
	source := strings.TrimSpace(input.Source)
	switch source {
	case dto.WordFavoriteSourceWordList:
		if input.CoarseUnitID == nil || *input.CoarseUnitID <= 0 {
			return normalizedWordFavoriteIdentity{}, validationError("coarse_unit_id is required")
		}
		return normalizedWordFavoriteIdentity{
			UserID:        userID,
			CoarseUnitID:  input.CoarseUnitID,
			Text:          strings.TrimSpace(input.Text),
			Source:        source,
			VideoID:       nil,
			SentenceIndex: nil,
			TokenIndex:    nil,
		}, nil
	case dto.WordFavoriteSourceVideoTranscript:
		if input.VideoID == nil || !isUUID(strings.TrimSpace(*input.VideoID)) {
			return normalizedWordFavoriteIdentity{}, validationError("video_id must be a uuid")
		}
		videoID := strings.TrimSpace(*input.VideoID)
		if input.SentenceIndex == nil || *input.SentenceIndex < 0 {
			return normalizedWordFavoriteIdentity{}, validationError("sentence_index must be non-negative")
		}
		if input.TokenIndex == nil || *input.TokenIndex < 0 {
			return normalizedWordFavoriteIdentity{}, validationError("token_index must be non-negative")
		}
		if input.CoarseUnitID != nil && *input.CoarseUnitID <= 0 {
			return normalizedWordFavoriteIdentity{}, validationError("coarse_unit_id must be positive")
		}
		return normalizedWordFavoriteIdentity{
			UserID:        userID,
			CoarseUnitID:  input.CoarseUnitID,
			Text:          strings.TrimSpace(input.Text),
			Source:        source,
			VideoID:       &videoID,
			SentenceIndex: input.SentenceIndex,
			TokenIndex:    input.TokenIndex,
		}, nil
	default:
		return normalizedWordFavoriteIdentity{}, validationError("source must be word_list or video_transcript")
	}
}

func (i normalizedWordFavoriteIdentity) tokenKey() model.WordFavoriteTokenKey {
	return model.WordFavoriteTokenKey{
		UserID:        i.UserID,
		VideoID:       *i.VideoID,
		SentenceIndex: *i.SentenceIndex,
		TokenIndex:    *i.TokenIndex,
	}
}

func normalizeWordFavoritesLimit(limit int) (int, error) {
	if limit == 0 {
		return defaultWordFavoritesLimit, nil
	}
	if limit < 1 || limit > maxWordFavoritesLimit {
		return 0, validationError("limit must be between 1 and 100")
	}
	return limit, nil
}

func mapWordFavoriteListItems(items []model.WordFavoriteListItem) []dto.WordFavoriteListItem {
	result := make([]dto.WordFavoriteListItem, 0, len(items))
	for _, item := range items {
		result = append(result, dto.WordFavoriteListItem{
			CoarseUnitID:      item.CoarseUnitID,
			Label:             item.Label,
			Pos:               item.Pos,
			ChineseLabel:      item.ChineseLabel,
			ChineseDef:        item.ChineseDef,
			Source:            item.Source,
			VideoID:           item.VideoID,
			SentenceIndex:     item.SentenceIndex,
			TokenIndex:        item.TokenIndex,
			SourceText:        item.SourceText,
			SourceTranslation: item.SourceTranslation,
			SourceDictionary:  item.SourceDictionary,
			SourceExplanation: item.SourceExplanation,
		})
	}
	return result
}

func classifyWordFavoriteRepositoryError(err error, notFoundMessage string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, apprepo.ErrWordFavoriteProjectionNotFound) {
		return NotFoundError(notFoundMessage)
	}
	return err
}

func int32Pointer(value int32) *int32 {
	return &value
}
