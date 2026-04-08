-- 作用：回滚删除 learning.user_unit_states 状态表。
-- 输入/输出：输入无；输出是删除状态表的数据库副作用。
-- 谁调用它：仓库级 migrate rollback 流程。
-- 它调用谁/传给谁：直接作用于 PostgreSQL；删除后上游 state repository 将无法读取或写入状态。
drop table if exists learning.user_unit_states;
