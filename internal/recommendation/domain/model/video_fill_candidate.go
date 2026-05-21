package model

import "time"

type VideoFillCandidate struct {
	VideoID           string
	DurationMs        int32
	MatchedUnitCount  int32
	TotalMentionCount int64
	MaxCoverageRatio  float64
	MappedSpanRatio   float64
	ViewCount         int64
	LikeCount         int64
	FavoriteCount     int64
	LastServedAt      *time.Time
	ServedCount       int32
	LastWatchedAt     *time.Time
	WatchCount        int32
	CompletedCount    int32
	MaxPositionMs     int32
}
