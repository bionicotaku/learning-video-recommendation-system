package dto

import "time"

const (
	UnitProgressBucketMastered   = "mastered"
	UnitProgressBucketUnmastered = "unmastered"
)

type ListUserUnitProgressRequest struct {
	UserID string
	Bucket string
	Limit  int
	Cursor string
}

type ListUserUnitProgressQuery struct {
	UserID       string
	Bucket       string
	LimitPlusOne int
	Cursor       *UnitProgressCursor
}

type ListUserUnitProgressResponse struct {
	Items []UnitProgressItem `json:"items"`
	Page  UnitProgressPage   `json:"page"`
}

type UnitProgressItem struct {
	CoarseUnitID    int64      `json:"coarse_unit_id"`
	Kind            string     `json:"kind"`
	Label           string     `json:"label"`
	LabelKey        string     `json:"-"`
	Pos             *string    `json:"pos"`
	ChineseLabel    *string    `json:"chinese_label"`
	ChineseDef      *string    `json:"chinese_def"`
	ProgressPercent float64    `json:"progress_percent"`
	LastProgressAt  *time.Time `json:"last_progress_at"`
}

type UnitProgressPage struct {
	Limit      int     `json:"limit"`
	HasMore    bool    `json:"has_more"`
	NextCursor *string `json:"next_cursor"`
}

type UnitProgressCursor struct {
	Bucket             string
	LabelKey           string
	Label              string
	CoarseUnitID       int64
	ProgressPercent    float64
	HasProgressPercent bool
}
