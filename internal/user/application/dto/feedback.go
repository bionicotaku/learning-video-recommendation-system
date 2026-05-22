package dto

import "encoding/json"

type SubmitFeedbackRequest struct {
	UserID           string
	ClientFeedbackID *string
	Payload          json.RawMessage
	Images           []FeedbackImageInput
}

type FeedbackImageInput struct {
	SortOrder   int32
	ContentType string
	SizeBytes   int32
	SHA256      string
	Width       int32
	Height      int32
	Data        []byte
}

type SubmitFeedbackResponse struct {
	FeedbackID string `json:"feedback_id"`
	Accepted   bool   `json:"accepted"`
	ImageCount int32  `json:"image_count"`
	CreatedAt  string `json:"created_at"`
}
