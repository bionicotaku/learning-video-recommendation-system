package model

type EndQuizQuestionCandidate struct {
	QuestionID           string
	ScopeType            string
	QuestionType         string
	CoarseUnitID         int64
	TargetText           string
	ContextSentenceIndex *int32
	ContextSpanIndex     *int32
	ContextStartMS       *int32
	ContextEndMS         *int32
	ContentPayload       []byte
}
