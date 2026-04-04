package enum

// EventType represents normalized learning events consumed by the scheduler.
type EventType string

const (
	EventTypeExposure EventType = "exposure"
	EventTypeLookup   EventType = "lookup"
	EventTypeNewLearn EventType = "new_learn"
	EventTypeReview   EventType = "review"
	EventTypeQuiz     EventType = "quiz"
)
