-- 作用：创建 Learning engine 自己拥有的 learning schema，作为后续业务表的逻辑命名空间。
-- 输入/输出：输入无；输出是数据库中的 learning schema。
-- 谁调用它：仓库级 migrate 流程。
-- 它调用谁/传给谁：直接作用于 PostgreSQL；结果供后续 create table migration 使用。
create schema if not exists learning;
