package model

import "time"

type UserUnitServingState struct {
	UserID       string
	CoarseUnitID int64
	LastServedAt *time.Time
	LastRunID    string
	ServedCount  int32
}

type UserVideoServingState struct {
	UserID       string
	VideoID      string
	LastServedAt *time.Time
	LastRunID    string
	ServedCount  int32
}
