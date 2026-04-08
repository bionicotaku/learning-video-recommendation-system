-- 作用：为 Learning engine 的状态表和事件表建立核心查询索引。
-- 输入/输出：输入无；输出是四个索引对象。
-- 谁调用它：仓库级 migrate 流程。
-- 它调用谁/传给谁：直接作用于 PostgreSQL；结果会被 state/event query 和 Recommendation 读取路径使用。
create index if not exists idx_user_unit_states_target_status
on learning.user_unit_states (user_id, is_target, status);

create index if not exists idx_user_unit_states_next_review
on learning.user_unit_states (user_id, next_review_at);

create index if not exists idx_unit_learning_events_user_unit_time
on learning.unit_learning_events (user_id, coarse_unit_id, occurred_at desc);

create index if not exists idx_unit_learning_events_user_video_time
on learning.unit_learning_events (user_id, video_id, occurred_at desc);
