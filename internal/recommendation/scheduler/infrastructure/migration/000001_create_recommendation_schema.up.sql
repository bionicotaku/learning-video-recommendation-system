-- 文件作用：
--   - 创建 Recommendation 自己的 recommendation schema
-- 输入/输出：
--   - 输入：无，执行当前 migration up
--   - 输出：创建 recommendation schema
-- 谁调用它：
--   - migration 执行器在初始化 Recommendation schema 时调用
-- 它调用谁/传给谁：
--   - 直接传给 PostgreSQL 执行
create schema if not exists recommendation;
