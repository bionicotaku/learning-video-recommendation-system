package learningevents

import (
	"encoding/json"
	"net/http"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
)

type quizAttemptBody struct {
	ClientContext json.RawMessage `json:"client_context"`

	ClientEventID       string   `json:"client_event_id"`
	QuestionID          string   `json:"question_id"`
	CoarseUnitID        int64    `json:"coarse_unit_id"`
	VideoID             string   `json:"video_id"`
	RecommendationRunID string   `json:"recommendation_run_id"`
	TriggerType         string   `json:"trigger_type"`
	SelectedOptionIDs   []string `json:"selected_option_ids"`
	SelectionIntervalMS []int32  `json:"selection_interval_ms"`
	IsFirstTryCorrect   *bool    `json:"is_first_try_correct"`
	TotalElapsedMS      int32    `json:"total_elapsed_ms"`
	ShownAt             string   `json:"shown_at"`
	CompletedAt         string   `json:"completed_at"`
}

func (h *Handler) recordQuizAttempt(w http.ResponseWriter, r *http.Request) {
	principal, err := requiredPrincipal(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	if err := validateContentType(r); err != nil {
		writeHandlerError(w, r, err)
		return
	}

	var body quizAttemptBody
	if err := request.DecodeJSONObject(r.Body, &body); err != nil {
		writeHandlerError(w, r, invalidRequest(err))
		return
	}

	command, err := mapQuizAttemptBody(principal.UserID, body)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}

	result, err := h.quizAttempts.Execute(r.Context(), command)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}

func mapQuizAttemptBody(userID string, body quizAttemptBody) (apvdto.RecordQuizAttemptRequest, error) {
	if len(body.ClientContext) == 0 {
		body.ClientContext = json.RawMessage(`{}`)
	}
	if err := request.ValidateJSONObject("client_context", body.ClientContext); err != nil {
		return apvdto.RecordQuizAttemptRequest{}, invalidRequest(err)
	}
	if body.ClientEventID == "" {
		return apvdto.RecordQuizAttemptRequest{}, apiservice.InvalidRequestError("client_event_id is required")
	}
	if err := request.ValidateRequiredUUID("question_id", body.QuestionID); err != nil {
		return apvdto.RecordQuizAttemptRequest{}, invalidRequest(err)
	}
	if body.CoarseUnitID <= 0 {
		return apvdto.RecordQuizAttemptRequest{}, apiservice.InvalidRequestError("coarse_unit_id is required")
	}
	if err := validateOptionalUUIDs(map[string]string{
		"video_id":              body.VideoID,
		"recommendation_run_id": body.RecommendationRunID,
	}); err != nil {
		return apvdto.RecordQuizAttemptRequest{}, invalidRequest(err)
	}
	if body.TriggerType == "" {
		return apvdto.RecordQuizAttemptRequest{}, apiservice.InvalidRequestError("trigger_type is required")
	}
	if !isValidQuizTriggerType(body.TriggerType) {
		return apvdto.RecordQuizAttemptRequest{}, apiservice.InvalidRequestError("trigger_type is unsupported")
	}
	if len(body.SelectedOptionIDs) == 0 {
		return apvdto.RecordQuizAttemptRequest{}, apiservice.InvalidRequestError("selected_option_ids is required")
	}
	if len(body.SelectedOptionIDs) != len(body.SelectionIntervalMS) {
		return apvdto.RecordQuizAttemptRequest{}, apiservice.InvalidRequestError("selection_interval_ms must match selected_option_ids")
	}
	if body.IsFirstTryCorrect == nil {
		return apvdto.RecordQuizAttemptRequest{}, apiservice.InvalidRequestError("is_first_try_correct is required")
	}
	if body.TotalElapsedMS < 0 {
		return apvdto.RecordQuizAttemptRequest{}, apiservice.InvalidRequestError("total_elapsed_ms must be non-negative")
	}
	shownAt, err := request.ParseRequiredTime("shown_at", body.ShownAt)
	if err != nil {
		return apvdto.RecordQuizAttemptRequest{}, invalidRequest(err)
	}
	completedAt, err := request.ParseRequiredTime("completed_at", body.CompletedAt)
	if err != nil {
		return apvdto.RecordQuizAttemptRequest{}, invalidRequest(err)
	}
	if completedAt.Before(shownAt) {
		return apvdto.RecordQuizAttemptRequest{}, apiservice.InvalidRequestError("completed_at must be >= shown_at")
	}
	if body.SelectedOptionIDs[len(body.SelectedOptionIDs)-1] != "correct" {
		return apvdto.RecordQuizAttemptRequest{}, apiservice.InvalidRequestError("selected_option_ids must end with correct")
	}
	if *body.IsFirstTryCorrect != (body.SelectedOptionIDs[0] == "correct") {
		return apvdto.RecordQuizAttemptRequest{}, apiservice.InvalidRequestError("is_first_try_correct does not match selected_option_ids")
	}
	for index, interval := range body.SelectionIntervalMS {
		if interval < 0 {
			return apvdto.RecordQuizAttemptRequest{}, apiservice.InvalidRequestError("selection_interval_ms[" + itoa(index) + "] must be non-negative")
		}
	}

	return apvdto.RecordQuizAttemptRequest{
		UserID:              userID,
		ClientContext:       body.ClientContext,
		ClientEventID:       body.ClientEventID,
		QuestionID:          body.QuestionID,
		CoarseUnitID:        body.CoarseUnitID,
		VideoID:             body.VideoID,
		RecommendationRunID: body.RecommendationRunID,
		TriggerType:         body.TriggerType,
		SelectedOptionIDs:   body.SelectedOptionIDs,
		SelectionIntervalMS: body.SelectionIntervalMS,
		IsFirstTryCorrect:   *body.IsFirstTryCorrect,
		TotalElapsedMS:      body.TotalElapsedMS,
		ShownAt:             shownAt,
		CompletedAt:         completedAt,
	}, nil
}

func isValidQuizTriggerType(value string) bool {
	switch value {
	case "video_end", "lookup_practice", "feed_review", "mid_video", "manual":
		return true
	default:
		return false
	}
}
