-- 文件作用：
--   - 回滚 recommendation schema 的创建
-- 输入/输出：
--   - 输入：无，执行当前 migration down
--   - 输出：删除整个 recommendation schema 及其对象
-- 谁调用它：
--   - migration 执行器在执行 recommendation down 时调用
-- 它调用谁/传给谁：
--   - 直接传给 PostgreSQL 执行
drop schema if exists recommendation cascade;
