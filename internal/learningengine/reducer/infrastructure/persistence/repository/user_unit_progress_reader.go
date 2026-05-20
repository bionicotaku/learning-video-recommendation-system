package repository

import (
	"context"
	"fmt"

	"learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	apprepo "learning-video-recommendation-system/internal/learningengine/reducer/application/repository"
	"learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/mapper"
	learningenginesqlc "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

type UserUnitProgressReader struct {
	queries *learningenginesqlc.Queries
}

var _ apprepo.UserUnitProgressReader = (*UserUnitProgressReader)(nil)

func NewUserUnitProgressReader(db learningenginesqlc.DBTX) *UserUnitProgressReader {
	return &UserUnitProgressReader{queries: learningenginesqlc.New(db)}
}

func (r *UserUnitProgressReader) ListUserUnitProgress(ctx context.Context, query dto.ListUserUnitProgressQuery) ([]dto.UnitProgressItem, error) {
	userID, err := mapper.StringToUUID(query.UserID)
	if err != nil {
		return nil, err
	}
	limitPlusOne, err := int32LimitPlusOne(query.LimitPlusOne)
	if err != nil {
		return nil, err
	}

	switch query.Bucket {
	case dto.UnitProgressBucketMastered:
		rows, err := r.queries.ListMasteredUnitProgress(ctx, learningenginesqlc.ListMasteredUnitProgressParams{
			UserID:             userID,
			HasCursor:          query.Cursor != nil,
			CursorLabelKey:     cursorLabelKey(query.Cursor),
			CursorLabel:        cursorLabel(query.Cursor),
			CursorCoarseUnitID: cursorCoarseUnitID(query.Cursor),
			LimitPlusOne:       limitPlusOne,
		})
		if err != nil {
			return nil, err
		}
		return mapMasteredUnitProgressRows(rows)
	case dto.UnitProgressBucketUnmastered:
		cursorProgressPercent, err := cursorProgressPercent(query.Cursor)
		if err != nil {
			return nil, err
		}
		rows, err := r.queries.ListUnmasteredUnitProgress(ctx, learningenginesqlc.ListUnmasteredUnitProgressParams{
			UserID:                userID,
			HasCursor:             query.Cursor != nil,
			CursorProgressPercent: cursorProgressPercent,
			CursorLabelKey:        cursorLabelKey(query.Cursor),
			CursorLabel:           cursorLabel(query.Cursor),
			CursorCoarseUnitID:    cursorCoarseUnitID(query.Cursor),
			LimitPlusOne:          limitPlusOne,
		})
		if err != nil {
			return nil, err
		}
		return mapUnmasteredUnitProgressRows(rows)
	default:
		return nil, fmt.Errorf("unsupported unit progress bucket: %s", query.Bucket)
	}
}

func mapMasteredUnitProgressRows(rows []learningenginesqlc.ListMasteredUnitProgressRow) ([]dto.UnitProgressItem, error) {
	result := make([]dto.UnitProgressItem, 0, len(rows))
	for _, row := range rows {
		progressPercent, err := mapper.NumericToFloat64(row.ProgressPercent)
		if err != nil {
			return nil, err
		}
		result = append(result, dto.UnitProgressItem{
			CoarseUnitID:    row.CoarseUnitID,
			Kind:            row.Kind,
			Label:           row.Label,
			LabelKey:        row.LabelKey,
			Pos:             textPointer(row.Pos),
			ChineseLabel:    textPointer(row.ChineseLabel),
			ChineseDef:      textPointer(row.ChineseDef),
			ProgressPercent: progressPercent,
			LastProgressAt:  mapper.TimePointerFromPG(row.LastProgressAt),
		})
	}
	return result, nil
}

func mapUnmasteredUnitProgressRows(rows []learningenginesqlc.ListUnmasteredUnitProgressRow) ([]dto.UnitProgressItem, error) {
	result := make([]dto.UnitProgressItem, 0, len(rows))
	for _, row := range rows {
		progressPercent, err := mapper.NumericToFloat64(row.ProgressPercent)
		if err != nil {
			return nil, err
		}
		result = append(result, dto.UnitProgressItem{
			CoarseUnitID:    row.CoarseUnitID,
			Kind:            row.Kind,
			Label:           row.Label,
			LabelKey:        row.LabelKey,
			Pos:             textPointer(row.Pos),
			ChineseLabel:    textPointer(row.ChineseLabel),
			ChineseDef:      textPointer(row.ChineseDef),
			ProgressPercent: progressPercent,
			LastProgressAt:  mapper.TimePointerFromPG(row.LastProgressAt),
		})
	}
	return result, nil
}

func int32LimitPlusOne(limit int) (int32, error) {
	if limit < 1 {
		return 0, fmt.Errorf("limit_plus_one must be positive")
	}
	return int32(limit), nil
}

func cursorLabelKey(cursor *dto.UnitProgressCursor) string {
	if cursor == nil {
		return ""
	}
	return cursor.LabelKey
}

func cursorLabel(cursor *dto.UnitProgressCursor) string {
	if cursor == nil {
		return ""
	}
	return cursor.Label
}

func cursorCoarseUnitID(cursor *dto.UnitProgressCursor) int64 {
	if cursor == nil {
		return 0
	}
	return cursor.CoarseUnitID
}

func cursorProgressPercent(cursor *dto.UnitProgressCursor) (pgtype.Numeric, error) {
	if cursor == nil {
		return mapper.Float64ToNumeric(0)
	}
	return mapper.Float64ToNumeric(cursor.ProgressPercent)
}

func textPointer(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	text := value.String
	return &text
}
