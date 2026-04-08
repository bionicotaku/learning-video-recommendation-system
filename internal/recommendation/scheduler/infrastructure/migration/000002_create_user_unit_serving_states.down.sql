-- 文件作用：
--   - 回滚 recommendation.user_unit_serving_states 表
-- 输入/输出：
--   - 输入：无，执行当前 migration down
--   - 输出：删除 serving state 表
-- 谁调用它：
--   - migration 执行器在回滚 Recommendation serving state 时调用
-- 它调用谁/传给谁：
--   - 直接传给 PostgreSQL 执行
drop table if exists recommendation.user_unit_serving_states;
