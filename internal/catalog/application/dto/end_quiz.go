package dto

type EndQuizQuestionLookupRequest struct {
	VideoID       string
	CoarseUnitIDs []int64
}

type EndQuizQuestionLookupResponse struct {
	VideoID              string        `json:"video_id"`
	Items                []EndQuizItem `json:"items"`
	MissingCoarseUnitIDs []int64       `json:"missing_coarse_unit_ids"`
}

type EndQuizItem struct {
	CoarseUnitID         int64           `json:"coarse_unit_id"`
	QuestionID           string          `json:"question_id"`
	Source               string          `json:"source"`
	QuestionType         string          `json:"question_type"`
	TargetText           string          `json:"target_text"`
	Question             string          `json:"question"`
	ContextText          *string         `json:"context_text"`
	Options              []EndQuizOption `json:"options"`
	Explanation          *string         `json:"explanation"`
	ContextSentenceIndex *int32          `json:"context_sentence_index"`
	ContextSpanIndex     *int32          `json:"context_span_index"`
	ContextStartMS       *int32          `json:"context_start_ms"`
	ContextEndMS         *int32          `json:"context_end_ms"`
}

type EndQuizOption struct {
	OptionID string `json:"option_id"`
	Text     string `json:"text"`
}
