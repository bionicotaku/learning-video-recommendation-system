-- 文件作用：
--   - 回滚 Recommendation 当前定义的索引
-- 输入/输出：
--   - 输入：无，执行当前 migration down
--   - 输出：删除 user_unit_serving_states 上的用户-最近推荐时间索引
-- 谁调用它：
--   - migration 执行器
-- 它调用谁/传给谁：
--   - 直接传给 PostgreSQL 执行
drop index if exists recommendation.idx_user_unit_serving_states_user_last_recommended;
