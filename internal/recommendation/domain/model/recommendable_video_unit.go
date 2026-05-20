package model

type RecommendableVideoUnit struct {
	VideoID                    string
	CoarseUnitID               int64
	MentionCount               int32
	SentenceCount              int32
	CoverageMs                 int32
	CoverageRatio              float64
	SentenceIndexes            []int32
	BestEvidenceRef            EvidenceRef
	BestEvidenceCandidateScore *float64
	BestEvidenceTargetText     *string
	DurationMs                 int32
	MappedSpanRatio            float64
}
