package dto

// RecordLearningEventsResult reports the accepted events and affected units.
type RecordLearningEventsResult struct {
	AcceptedCount int
	UpdatedUnits  []int64
}
