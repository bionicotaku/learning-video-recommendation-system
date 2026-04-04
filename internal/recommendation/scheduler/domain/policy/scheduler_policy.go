package policy

// SchedulerPolicy centralizes scheduler constants and tunable defaults.
type SchedulerPolicy struct {
	MasteredIntervalDays float64
	InitialIntervals     []float64
	MinEaseFactor        float64
}

const (
	DefaultMasteredIntervalDays = 21
	DefaultMinEaseFactor        = 1.3
)

var defaultInitialIntervals = []float64{1, 3, 6}

// DefaultSchedulerPolicy returns the MVP scheduler policy defaults from the design doc.
func DefaultSchedulerPolicy() SchedulerPolicy {
	intervals := make([]float64, len(defaultInitialIntervals))
	copy(intervals, defaultInitialIntervals)

	return SchedulerPolicy{
		MasteredIntervalDays: DefaultMasteredIntervalDays,
		InitialIntervals:     intervals,
		MinEaseFactor:        DefaultMinEaseFactor,
	}
}
