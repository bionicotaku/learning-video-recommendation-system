-- 作用：定义事件表相关 SQL，包括计数、插入和按用户顺序读取事件。
-- 输入/输出：输入是 sqlc 参数 user_id、event fields；输出是查询结果或执行副作用。
-- 谁调用它：sqlc 生成器；运行时通过 unit_learning_event_repo.go 间接调用。
-- 它调用谁/传给谁：直接作用于 PostgreSQL；生成的方法会传给 event repository 使用。
-- name: CountUnitLearningEvents :one
select count(*)::bigint
from learning.unit_learning_events;

-- name: InsertUnitLearningEvent :exec
insert into learning.unit_learning_events (
  user_id,
  coarse_unit_id,
  video_id,
  event_type,
  source_type,
  source_ref_id,
  is_correct,
  quality,
  response_time_ms,
  metadata,
  occurred_at,
  created_at
) values (
  sqlc.arg(user_id),
  sqlc.arg(coarse_unit_id),
  sqlc.arg(video_id),
  sqlc.arg(event_type),
  sqlc.arg(source_type),
  sqlc.arg(source_ref_id),
  sqlc.arg(is_correct),
  sqlc.arg(quality),
  sqlc.arg(response_time_ms),
  sqlc.arg(metadata),
  sqlc.arg(occurred_at),
  sqlc.arg(created_at)
);

-- name: ListUnitLearningEventsByUserOrdered :many
select *
from learning.unit_learning_events
where user_id = sqlc.arg(user_id)
order by occurred_at asc, event_id asc;
