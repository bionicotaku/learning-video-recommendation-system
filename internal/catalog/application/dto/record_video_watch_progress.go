package dto

import "time"

type RecordVideoWatchProgressRequest struct {
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

type RecordVideoWatchProgressResponse struct {
	Accepted bool `json:"accepted"`
}
