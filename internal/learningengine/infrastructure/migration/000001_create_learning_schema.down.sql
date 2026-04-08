-- 作用：回滚删除整个 learning schema 及其下属对象。
-- 输入/输出：输入无；输出是删除 learning schema 的数据库副作用。
-- 谁调用它：仓库级 migrate rollback 流程。
-- 它调用谁/传给谁：直接作用于 PostgreSQL；会连带清理该 schema 下的表和索引。
drop schema if exists learning cascade;
