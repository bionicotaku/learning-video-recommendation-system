package service

import (
	"context"
	"fmt"
	"strings"

	"learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	apprepo "learning-video-recommendation-system/internal/learningengine/reducer/application/repository"
	appusecase "learning-video-recommendation-system/internal/learningengine/reducer/application/usecase"
)

const (
	defaultUnitProgressLimit = 50
	maxUnitProgressLimit     = 100
)

type ListUserUnitProgressUsecase struct {
	reader apprepo.UserUnitProgressReader
}

var _ appusecase.ListUserUnitProgressUsecase = (*ListUserUnitProgressUsecase)(nil)

func NewListUserUnitProgressUsecase(reader apprepo.UserUnitProgressReader) *ListUserUnitProgressUsecase {
	return &ListUserUnitProgressUsecase{reader: reader}
}

func (u *ListUserUnitProgressUsecase) Execute(ctx context.Context, request dto.ListUserUnitProgressRequest) (dto.ListUserUnitProgressResponse, error) {
	userID := strings.TrimSpace(request.UserID)
	if userID == "" {
		return dto.ListUserUnitProgressResponse{}, validationError("user_id is required")
	}
	if !isValidUnitProgressBucket(request.Bucket) {
		return dto.ListUserUnitProgressResponse{}, invalidUnitProgressBucket(request.Bucket)
	}
	if u.reader == nil {
		return dto.ListUserUnitProgressResponse{}, fmt.Errorf("unit progress reader is required")
	}

	limit, err := normalizeUnitProgressLimit(request.Limit)
	if err != nil {
		return dto.ListUserUnitProgressResponse{}, err
	}

	cursor, err := decodeUnitProgressCursor(request.Cursor)
	if err != nil {
		return dto.ListUserUnitProgressResponse{}, err
	}
	if err := validateUnitProgressCursorBucket(cursor, request.Bucket); err != nil {
		return dto.ListUserUnitProgressResponse{}, err
	}

	rows, err := u.reader.ListUserUnitProgress(ctx, dto.ListUserUnitProgressQuery{
		UserID:       userID,
		Bucket:       request.Bucket,
		LimitPlusOne: limit + 1,
		Cursor:       cursor,
	})
	if err != nil {
		return dto.ListUserUnitProgressResponse{}, err
	}

	hasMore := len(rows) > limit
	items := rows
	if hasMore {
		items = rows[:limit]
	}
	if items == nil {
		items = []dto.UnitProgressItem{}
	}

	var nextCursor *string
	if hasMore && len(items) > 0 {
		encoded, err := encodeUnitProgressCursor(request.Bucket, items[len(items)-1])
		if err != nil {
			return dto.ListUserUnitProgressResponse{}, wrapUnitProgressCursorEncodeError(err)
		}
		nextCursor = &encoded
	}

	return dto.ListUserUnitProgressResponse{
		Items: items,
		Page: dto.UnitProgressPage{
			Limit:      limit,
			HasMore:    hasMore,
			NextCursor: nextCursor,
		},
	}, nil
}

func normalizeUnitProgressLimit(limit int) (int, error) {
	if limit == 0 {
		return defaultUnitProgressLimit, nil
	}
	if limit < 1 {
		return 0, validationError("limit must be between 1 and 100")
	}
	if limit > maxUnitProgressLimit {
		return 0, validationError("limit must be between 1 and 100")
	}
	return limit, nil
}

func isValidUnitProgressBucket(bucket string) bool {
	return bucket == dto.UnitProgressBucketMastered || bucket == dto.UnitProgressBucketUnmastered
}
