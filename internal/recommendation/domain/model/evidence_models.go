package model

type SemanticSpan struct {
	VideoID       string
	SentenceIndex int32
	SpanIndex     int32
	CoarseUnitID  *int64
	StartMs       int32
	EndMs         int32
	Text          string
	Explanation   string
}

type TranscriptSentence struct {
	VideoID       string
	SentenceIndex int32
	Text          string
	StartMs       int32
	EndMs         int32
	Explanation   string
}
