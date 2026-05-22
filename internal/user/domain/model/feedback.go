package model

import (
	"encoding/json"
	"time"
)

type FeedbackSubmission struct {
	UserID           string
	ClientFeedbackID *string
	Payload          json.RawMessage
	Images           []FeedbackImage
}

type FeedbackImage struct {
	SortOrder   int32
	ContentType string
	SizeBytes   int32
	SHA256      string
	Width       int32
	Height      int32
	Data        []byte
}

type FeedbackSubmissionResult struct {
	FeedbackID string
	ImageCount int32
	CreatedAt  time.Time
}
