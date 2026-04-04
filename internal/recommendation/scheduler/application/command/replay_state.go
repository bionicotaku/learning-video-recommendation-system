package command

import (
	"time"

	"github.com/google/uuid"
)

// ReplayStateCommand requests state rebuild from event history.
type ReplayStateCommand struct {
	UserID       uuid.UUID
	CoarseUnitID *int64
	FromTime     *time.Time
}
