-- 文件作用：
--   - 回滚 recommendation.scheduler_run_items 表
-- 输入/输出：
--   - 输入：无，执行当前 migration down
--   - 输出：删除 run item 明细表
-- 谁调用它：
--   - migration 执行器
-- 它调用谁/传给谁：
--   - 直接传给 PostgreSQL 执行
drop table if exists recommendation.scheduler_run_items;
