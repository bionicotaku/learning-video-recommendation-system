package policy

// LearningPolicy centralizes learning-engine constants and tunable defaults.
type LearningPolicy struct {
	MasteredIntervalDays float64
	InitialIntervals     []float64
	MinEaseFactor        float64
}

const (
	DefaultMasteredIntervalDays = 21
	DefaultMinEaseFactor        = 1.3
)

var defaultInitialIntervals = []float64{1, 3, 6}

// DefaultLearningPolicy returns the MVP learning policy defaults from the design doc.
func DefaultLearningPolicy() LearningPolicy {
	intervals := make([]float64, len(defaultInitialIntervals))
	copy(intervals, defaultInitialIntervals)

	return LearningPolicy{
		MasteredIntervalDays: DefaultMasteredIntervalDays,
		InitialIntervals:     intervals,
		MinEaseFactor:        DefaultMinEaseFactor,
	}
}
