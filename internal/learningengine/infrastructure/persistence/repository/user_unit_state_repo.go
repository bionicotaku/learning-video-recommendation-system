// 作用：实现状态仓储接口，负责状态表的单条读取、upsert、批量 upsert 和按用户删除。
// 输入/输出：输入是 userID/coarseUnitID 或 UserUnitState；输出是 UserUnitState、error 或写入结果。
// 谁调用它：record/replay use case，通过 application/repository/UserUnitStateRepository 接口调用；fixture 负责装配。
// 它调用谁/传给谁：调用 querier_resolver.go、user_unit_state_mapper.go 和 sqlcgen/unit_states.sql.go；结果会传回 use case。
package repository

import (
	"context"
	"errors"

	apprepo "learning-video-recommendation-system/internal/learningengine/application/repository"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/mapper"
	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/sqlcgen"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type userUnitStateRepository struct {
	querier sqlcgen.Querier
}

func NewUserUnitStateRepository(querier sqlcgen.Querier) apprepo.UserUnitStateRepository {
	return userUnitStateRepository{querier: querier}
}

func (r userUnitStateRepository) GetByUserAndUnit(ctx context.Context, userID uuid.UUID, coarseUnitID int64) (*model.UserUnitState, error) {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return nil, err
	}

	row, err := q.GetUserUnitStateByUserAndUnit(ctx, sqlcgen.GetUserUnitStateByUserAndUnitParams{
		UserID:       mapper.UUIDToPG(userID),
		CoarseUnitID: coarseUnitID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	state, err := mapper.UserUnitStateFromRow(row)
	if err != nil {
		return nil, err
	}

	return &state, nil
}

func (r userUnitStateRepository) Upsert(ctx context.Context, state *model.UserUnitState) error {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return err
	}

	params, err := mapper.UserUnitStateToUpsertParams(state)
	if err != nil {
		return err
	}

	return q.UpsertUserUnitState(ctx, params)
}

func (repo userUnitStateRepository) BatchUpsert(ctx context.Context, states []*model.UserUnitState) error {
	for _, state := range states {
		if err := repo.Upsert(ctx, state); err != nil {
			return err
		}
	}

	return nil
}

func (r userUnitStateRepository) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return err
	}

	return q.DeleteUserUnitStatesByUser(ctx, mapper.UUIDToPG(userID))
}
