package model

import "time"

type VideoUserState struct {
	UserID         string
	VideoID        string
	LastWatchedAt  *time.Time
	WatchCount     int32
	CompletedCount int32
	LastPositionMs int32
	MaxPositionMs  int32
	TotalWatchMs   int64
}
