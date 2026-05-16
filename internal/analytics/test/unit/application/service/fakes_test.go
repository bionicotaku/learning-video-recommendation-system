package service_test

import (
	"context"

	"learning-video-recommendation-system/internal/analytics/domain/model"
)

type fakeRawEventWriter struct {
	interactions       []model.RawLearningInteractionEvent
	quizEvents         []model.RawQuizEvent
	interactionResults []model.RawEventWriteResult
	quizResult         model.RawEventWriteResult
}

func (w *fakeRawEventWriter) UpsertLearningInteractions(_ context.Context, events []model.RawLearningInteractionEvent) ([]model.RawEventWriteResult, error) {
	w.interactions = append(w.interactions, events...)
	if w.interactionResults != nil {
		return w.interactionResults, nil
	}
	results := make([]model.RawEventWriteResult, 0, len(events))
	for _, event := range events {
		results = append(results, model.RawEventWriteResult{ClientEventID: event.ClientEventID, EventID: event.ClientEventID, Inserted: true})
	}
	return results, nil
}

func (w *fakeRawEventWriter) UpsertQuizEvent(_ context.Context, event model.RawQuizEvent) (model.RawEventWriteResult, error) {
	w.quizEvents = append(w.quizEvents, event)
	if w.quizResult.EventID != "" {
		return w.quizResult, nil
	}
	return model.RawEventWriteResult{ClientEventID: event.ClientEventID, EventID: event.ClientEventID, Inserted: true}, nil
}
