package command

import "github.com/google/uuid"

// ReplayStateCommand requests state rebuild from event history.
type ReplayStateCommand struct {
	UserID uuid.UUID
}
