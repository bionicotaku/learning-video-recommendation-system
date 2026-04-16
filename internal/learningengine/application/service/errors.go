package service

import "errors"

var (
	ErrUserUnitStateNotFound     = errors.New("user unit state not found")
	ErrUserUnitStateNotSuspended = errors.New("user unit state is not suspended")
	ErrLateStrongEvent           = errors.New("late strong event")
)
