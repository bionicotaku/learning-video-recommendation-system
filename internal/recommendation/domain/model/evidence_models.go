package model

type SemanticSpan struct {
	VideoID       string
	SentenceIndex int32
	SpanIndex     int32
	CoarseUnitID  *int64
	StartMs       int32
	EndMs         int32
}

type TranscriptSentence struct {
	VideoID       string
	SentenceIndex int32
	StartMs       int32
	EndMs         int32
}
