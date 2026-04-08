-- 文件作用：
--   - 为 Recommendation 当前读写热点补充索引
-- 输入/输出：
--   - 输入：无，执行当前 migration up
--   - 输出：创建 user_id + last_recommended_at 索引
-- 谁调用它：
--   - migration 执行器
-- 它调用谁/传给谁：
--   - 直接传给 PostgreSQL 执行
create index if not exists idx_user_unit_serving_states_user_last_recommended
on recommendation.user_unit_serving_states (user_id, last_recommended_at);
