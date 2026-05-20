//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	persistrepo "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/learningengine/reducer/test/fixture"
)

func TestUserUnitProgressReaderListMasteredFiltersAndSortsByLabel(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	lastProgressAt := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	seedUnitProgressUnit(t, db, 101, "Banana", "noun", "香蕉", "黄色水果", "active")
	seedUnitProgressUnit(t, db, 102, "apple", "noun", "苹果", "红色水果", "active")
	seedUnitProgressUnit(t, db, 103, "carrot", "noun", "胡萝卜", "蔬菜", "active")
	seedUnitProgressUnit(t, db, 104, "date", "noun", "枣", "水果", "inactive")
	seedUnitProgressState(t, db, userID, 101, false, "mastered", 100, &lastProgressAt)
	seedUnitProgressState(t, db, userID, 102, true, "mastered", 100, nil)
	seedUnitProgressState(t, db, userID, 103, true, "learning", 50, nil)
	seedUnitProgressState(t, db, userID, 104, false, "mastered", 100, nil)

	reader := persistrepo.NewUserUnitProgressReader(db.Pool)
	rows, err := reader.ListUserUnitProgress(ctx, dto.ListUserUnitProgressQuery{
		UserID:       userID,
		Bucket:       dto.UnitProgressBucketMastered,
		LimitPlusOne: 10,
	})
	if err != nil {
		t.Fatalf("ListUserUnitProgress() error = %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows len = %d, want 2: %+v", len(rows), rows)
	}
	if rows[0].CoarseUnitID != 102 || rows[1].CoarseUnitID != 101 {
		t.Fatalf("row order ids = %v, want [102 101]", unitProgressIDs(rows))
	}
	if rows[1].Pos == nil || *rows[1].Pos != "noun" ||
		rows[1].ChineseLabel == nil || *rows[1].ChineseLabel != "香蕉" ||
		rows[1].ChineseDef == nil || *rows[1].ChineseDef != "黄色水果" {
		t.Fatalf("display fields not mapped: %+v", rows[1])
	}
	if rows[1].LastProgressAt == nil || !rows[1].LastProgressAt.Equal(lastProgressAt) {
		t.Fatalf("last_progress_at = %v, want %v", rows[1].LastProgressAt, lastProgressAt)
	}
}

func TestUserUnitProgressReaderListUnmasteredFiltersAndSortsByProgressThenLabel(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	seedUnitProgressUnit(t, db, 201, "derive", "verb", "得出", "从某处取得", "active")
	seedUnitProgressUnit(t, db, 202, "Constrain", "verb", "限制", "限制范围", "active")
	seedUnitProgressUnit(t, db, 203, "abandon", "verb", "放弃", "停止继续", "active")
	seedUnitProgressUnit(t, db, 204, "mastered-target", "noun", "已掌握", "已掌握词", "active")
	seedUnitProgressUnit(t, db, 205, "suspended", "noun", "暂停", "暂停词", "active")
	seedUnitProgressUnit(t, db, 206, "inactive-target", "noun", "非目标", "非目标词", "active")
	seedUnitProgressUnit(t, db, 207, "inactive-unit", "noun", "下线", "下线词", "inactive")
	seedUnitProgressState(t, db, userID, 201, true, "learning", 20, nil)
	seedUnitProgressState(t, db, userID, 202, true, "reviewing", 64.25, nil)
	seedUnitProgressState(t, db, userID, 203, true, "new", 64.25, nil)
	seedUnitProgressState(t, db, userID, 204, true, "mastered", 100, nil)
	seedUnitProgressState(t, db, userID, 205, true, "suspended", 10, nil)
	seedUnitProgressState(t, db, userID, 206, false, "learning", 99, nil)
	seedUnitProgressState(t, db, userID, 207, true, "learning", 98, nil)

	reader := persistrepo.NewUserUnitProgressReader(db.Pool)
	rows, err := reader.ListUserUnitProgress(ctx, dto.ListUserUnitProgressQuery{
		UserID:       userID,
		Bucket:       dto.UnitProgressBucketUnmastered,
		LimitPlusOne: 10,
	})
	if err != nil {
		t.Fatalf("ListUserUnitProgress() error = %v", err)
	}
	if got := unitProgressIDs(rows); len(got) != 3 || got[0] != 203 || got[1] != 202 || got[2] != 201 {
		t.Fatalf("row order ids = %v, want [203 202 201]", got)
	}
}

func TestUserUnitProgressReaderMasteredCursorReturnsNextPage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	seedUnitProgressUnit(t, db, 301, "apple", "noun", "苹果", "红色水果", "active")
	seedUnitProgressUnit(t, db, 302, "Banana", "noun", "香蕉", "黄色水果", "active")
	seedUnitProgressUnit(t, db, 303, "carrot", "noun", "胡萝卜", "蔬菜", "active")
	seedUnitProgressState(t, db, userID, 301, false, "mastered", 100, nil)
	seedUnitProgressState(t, db, userID, 302, false, "mastered", 100, nil)
	seedUnitProgressState(t, db, userID, 303, false, "mastered", 100, nil)

	reader := persistrepo.NewUserUnitProgressReader(db.Pool)
	rows, err := reader.ListUserUnitProgress(ctx, dto.ListUserUnitProgressQuery{
		UserID:       userID,
		Bucket:       dto.UnitProgressBucketMastered,
		LimitPlusOne: 10,
		Cursor: &dto.UnitProgressCursor{
			Bucket:       dto.UnitProgressBucketMastered,
			LabelKey:     "banana",
			Label:        "Banana",
			CoarseUnitID: 302,
		},
	})
	if err != nil {
		t.Fatalf("ListUserUnitProgress() error = %v", err)
	}
	if got := unitProgressIDs(rows); len(got) != 1 || got[0] != 303 {
		t.Fatalf("row ids = %v, want [303]", got)
	}
}

func TestUserUnitProgressReaderUnmasteredCursorReturnsNextPage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	seedUnitProgressUnit(t, db, 401, "abandon", "verb", "放弃", "停止继续", "active")
	seedUnitProgressUnit(t, db, 402, "Constrain", "verb", "限制", "限制范围", "active")
	seedUnitProgressUnit(t, db, 403, "derive", "verb", "得出", "从某处取得", "active")
	seedUnitProgressState(t, db, userID, 401, true, "learning", 64.25, nil)
	seedUnitProgressState(t, db, userID, 402, true, "reviewing", 64.25, nil)
	seedUnitProgressState(t, db, userID, 403, true, "new", 20, nil)

	reader := persistrepo.NewUserUnitProgressReader(db.Pool)
	rows, err := reader.ListUserUnitProgress(ctx, dto.ListUserUnitProgressQuery{
		UserID:       userID,
		Bucket:       dto.UnitProgressBucketUnmastered,
		LimitPlusOne: 10,
		Cursor: &dto.UnitProgressCursor{
			Bucket:             dto.UnitProgressBucketUnmastered,
			ProgressPercent:    64.25,
			HasProgressPercent: true,
			LabelKey:           "constrain",
			Label:              "Constrain",
			CoarseUnitID:       402,
		},
	})
	if err != nil {
		t.Fatalf("ListUserUnitProgress() error = %v", err)
	}
	if got := unitProgressIDs(rows); len(got) != 1 || got[0] != 403 {
		t.Fatalf("row ids = %v, want [403]", got)
	}
}

func seedUnitProgressUnit(t *testing.T, db *fixture.TestDatabase, unitID int64, label string, pos string, chineseLabel string, chineseDef string, status string) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		insert into semantic.coarse_unit (
			id,
			kind,
			label,
			lang,
			pos,
			chinese_label,
			chinese_def,
			english_label,
			status,
			version,
			fine_unit_ids,
			original_defs
		) values (
			$1::bigint,
			'word',
			$2::text,
			'en',
			$3::text,
			$4::text,
			$5::text,
			$2::text,
			$6::text,
			1,
			'{}'::bigint[],
			'{}'::text[]
		)`, unitID, label, pos, chineseLabel, chineseDef, status); err != nil {
		t.Fatalf("seed semantic.coarse_unit: %v", err)
	}
}

func seedUnitProgressState(t *testing.T, db *fixture.TestDatabase, userID string, unitID int64, isTarget bool, status string, progressPercent float64, lastProgressAt *time.Time) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		insert into learning.user_unit_states (
			user_id,
			coarse_unit_id,
			is_target,
			status,
			progress_percent,
			last_progress_at
		) values (
			$1::uuid,
			$2::bigint,
			$3::boolean,
			$4::text,
			$5::numeric,
			$6::timestamptz
		)`, userID, unitID, isTarget, status, progressPercent, lastProgressAt); err != nil {
		t.Fatalf("seed learning.user_unit_states: %v", err)
	}
}

func unitProgressIDs(rows []dto.UnitProgressItem) []int64 {
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.CoarseUnitID)
	}
	return ids
}
