package command

import "github.com/google/uuid"

// ReplayUserStatesCommand requests a full user-state rebuild from event history.
type ReplayUserStatesCommand struct {
	UserID uuid.UUID
}
