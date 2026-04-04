package integration

import (
	appservice "learning-video-recommendation-system/internal/recommendation/scheduler/application/service"
	"learning-video-recommendation-system/internal/recommendation/scheduler/application/usecase"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/aggregate"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/policy"
	repopkg "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"
	txtx "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/tx"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newRecordEventsUseCase(pool *pgxpool.Pool, querier sqlcgen.Querier) usecase.RecordLearningEventsAndUpdateStateUseCase {
	return usecase.NewRecordLearningEventsAndUpdateStateUseCase(
		txtx.NewPGXTxManager(pool),
		repopkg.NewUserUnitStateRepository(querier),
		repopkg.NewUnitLearningEventRepository(querier),
		aggregate.NewUserUnitReducer(),
	)
}

func newReplayUseCase(pool *pgxpool.Pool, querier sqlcgen.Querier) usecase.ReplayUserUnitStatesUseCase {
	return usecase.NewReplayUserUnitStatesUseCase(
		txtx.NewPGXTxManager(pool),
		repopkg.NewUserUnitStateRepository(querier),
		repopkg.NewUnitLearningEventRepository(querier),
		appservice.NewUserStateRebuilder(aggregate.NewUserUnitReducer(), policy.DefaultSchedulerPolicy()),
	)
}

func filterEventsByUnit(events []model.LearningEvent, userID uuid.UUID, coarseUnitID int64) []model.LearningEvent {
	items := make([]model.LearningEvent, 0, len(events))
	for _, event := range events {
		if event.UserID == userID && event.CoarseUnitID == coarseUnitID {
			items = append(items, event)
		}
	}

	return items
}
