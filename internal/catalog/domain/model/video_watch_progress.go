package model

import "time"

type VideoWatchProgress struct {
	UserID         string
	VideoID        string
	WatchSessionID string
	PositionMS     int32
	ActiveWatchMS  int64
	OccurredAt     time.Time
	SourceSurface  string
	ClientContext  []byte
	Metadata       []byte
}

type VideoWatchProgressResult struct {
	Accepted           bool
	CreatedSession     bool
	CompletedSession   bool
	DeltaActiveWatchMS int64
}
