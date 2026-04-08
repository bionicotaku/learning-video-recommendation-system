-- 作用：回滚删除 learning.unit_learning_events 事件表。
-- 输入/输出：输入无；输出是删除事件表的数据库副作用。
-- 谁调用它：仓库级 migrate rollback 流程。
-- 它调用谁/传给谁：直接作用于 PostgreSQL；删除后 record/replay 链路将失去事件真相层。
drop table if exists learning.unit_learning_events;
