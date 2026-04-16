package model

import "time"

type VideoUserState struct {
	UserID         string
	VideoID        string
	LastWatchedAt  *time.Time
	WatchCount     int32
	CompletedCount int32
	LastWatchRatio float64
	MaxWatchRatio  float64
}
