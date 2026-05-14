package service

import "errors"

var (
	ErrUserUnitStateNotFound     = errors.New("user unit state not found")
	ErrUserUnitStateNotSuspended = errors.New("user unit state is not suspended")
	ErrLateProgressEvent         = errors.New("late progress event")
)
