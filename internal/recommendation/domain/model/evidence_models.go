package model

type SemanticSpan struct {
	VideoID       string
	SentenceIndex int32
	SpanIndex     int32
	CoarseUnitID  *int64
	StartMs       int32
	EndMs         int32
	SurfaceText   string
	Explanation   *string
	BaseForm      *string
	Translation   *string
	Dictionary    *string
	MappingReason *string
}

type TranscriptSentence struct {
	VideoID       string
	SentenceIndex int32
	StartMs       int32
	EndMs         int32
	Text          string
	Translation   *string
}
