-- 文件作用：
--   - 定义 scheduler 读取 review/new 候选的 SQL
-- 输入/输出：
--   - 输入：user_id，以及 review 查询使用的 now
--   - 输出：learning.user_unit_states + semantic.coarse_unit + recommendation.user_unit_serving_states 的联表结果
-- 谁调用它：
--   - sqlc 读取它生成 FindDueReviewCandidates / FindNewCandidates
--   - repository/learning_state_snapshot_read_repo.go 间接调用
-- 它调用谁/传给谁：
--   - 直接传给 PostgreSQL 执行
-- name: FindDueReviewCandidates :many
select
  s.user_id,
  s.coarse_unit_id,
  s.is_target,
  s.target_source,
  s.target_source_ref_id,
  s.target_priority,
  s.status,
  s.progress_percent,
  s.mastery_score,
  s.first_seen_at,
  s.last_seen_at,
  s.last_reviewed_at,
  s.seen_count,
  s.strong_event_count,
  s.review_count,
  s.correct_count,
  s.wrong_count,
  s.consecutive_correct,
  s.consecutive_wrong,
  s.last_quality,
  s.recent_quality_window,
  s.recent_correctness_window,
  s.repetition,
  s.interval_days,
  s.ease_factor,
  s.next_review_at,
  s.suspended_reason,
  s.created_at as state_created_at,
  s.updated_at as state_updated_at,
  us.last_recommended_at,
  us.last_recommendation_run_id,
  us.created_at as serving_created_at,
  us.updated_at as serving_updated_at,
  u.kind as unit_kind,
  u.label as unit_label,
  u.pos as unit_pos,
  u.english_def as unit_english_def,
  u.chinese_def as unit_chinese_def
from learning.user_unit_states s
join semantic.coarse_unit u on u.id = s.coarse_unit_id
left join recommendation.user_unit_serving_states us
  on us.user_id = s.user_id
 and us.coarse_unit_id = s.coarse_unit_id
where s.user_id = sqlc.arg(user_id)
  and s.is_target = true
  and s.status in ('learning', 'reviewing', 'mastered')
  and s.next_review_at <= sqlc.arg(now)
order by s.next_review_at asc, s.coarse_unit_id asc;

-- name: FindNewCandidates :many
select
  s.user_id,
  s.coarse_unit_id,
  s.is_target,
  s.target_source,
  s.target_source_ref_id,
  s.target_priority,
  s.status,
  s.progress_percent,
  s.mastery_score,
  s.first_seen_at,
  s.last_seen_at,
  s.last_reviewed_at,
  s.seen_count,
  s.strong_event_count,
  s.review_count,
  s.correct_count,
  s.wrong_count,
  s.consecutive_correct,
  s.consecutive_wrong,
  s.last_quality,
  s.recent_quality_window,
  s.recent_correctness_window,
  s.repetition,
  s.interval_days,
  s.ease_factor,
  s.next_review_at,
  s.suspended_reason,
  s.created_at as state_created_at,
  s.updated_at as state_updated_at,
  us.last_recommended_at,
  us.last_recommendation_run_id,
  us.created_at as serving_created_at,
  us.updated_at as serving_updated_at,
  u.kind as unit_kind,
  u.label as unit_label,
  u.pos as unit_pos,
  u.english_def as unit_english_def,
  u.chinese_def as unit_chinese_def
from learning.user_unit_states s
join semantic.coarse_unit u on u.id = s.coarse_unit_id
left join recommendation.user_unit_serving_states us
  on us.user_id = s.user_id
 and us.coarse_unit_id = s.coarse_unit_id
where s.user_id = sqlc.arg(user_id)
  and s.is_target = true
  and s.status = 'new'
order by s.target_priority desc, s.coarse_unit_id asc;
