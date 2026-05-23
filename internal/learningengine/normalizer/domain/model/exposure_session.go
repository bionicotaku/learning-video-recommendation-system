package model

import "time"

type ExposureSession3Window struct {
	UserID          string
	CoarseUnitID    int64
	OccurredAt      time.Time
	ThirdVideoID    string
	WatchSessionIDs []string
	VideoIDs        []string
	RawEventCount   int32
}
