package enum

// UnitStatus represents the current scheduler state of a user-unit relation.
type UnitStatus string

const (
	UnitStatusNew       UnitStatus = "new"
	UnitStatusLearning  UnitStatus = "learning"
	UnitStatusReviewing UnitStatus = "reviewing"
	UnitStatusMastered  UnitStatus = "mastered"
	UnitStatusSuspended UnitStatus = "suspended"
)
