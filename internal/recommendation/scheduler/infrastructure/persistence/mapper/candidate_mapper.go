package mapper

import (
	"learning-video-recommendation-system/internal/recommendation/scheduler/application/query"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"
)

// ReviewCandidatesFromRows maps sqlc candidate rows to domain query objects.
func ReviewCandidatesFromRows(rows []sqlcgen.FindDueReviewCandidatesRow) ([]query.ReviewCandidate, error) {
	items := make([]query.ReviewCandidate, 0, len(rows))
	for _, row := range rows {
		item, err := ReviewCandidateFromRow(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

// NewCandidatesFromRows maps sqlc candidate rows to domain query objects.
func NewCandidatesFromRows(rows []sqlcgen.FindNewCandidatesRow) ([]query.NewCandidate, error) {
	items := make([]query.NewCandidate, 0, len(rows))
	for _, row := range rows {
		item, err := NewCandidateFromRow(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}
