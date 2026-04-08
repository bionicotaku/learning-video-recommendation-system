-- 作用：回滚删除 Learning engine 的核心索引。
-- 输入/输出：输入无；输出是索引删除的数据库副作用。
-- 谁调用它：仓库级 migrate rollback 流程。
-- 它调用谁/传给谁：直接作用于 PostgreSQL；删除后相关查询性能会下降。
drop index if exists learning.idx_unit_learning_events_user_video_time;
drop index if exists learning.idx_unit_learning_events_user_unit_time;
drop index if exists learning.idx_user_unit_states_next_review;
drop index if exists learning.idx_user_unit_states_target_status;
