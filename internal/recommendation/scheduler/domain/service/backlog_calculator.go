package service

type BacklogCalculator interface {
	Compute(reviewBacklog int) int
}

type backlogCalculator struct{}

func NewBacklogCalculator() BacklogCalculator {
	return backlogCalculator{}
}

func (backlogCalculator) Compute(reviewBacklog int) int {
	if reviewBacklog < 0 {
		return 0
	}

	return reviewBacklog
}
