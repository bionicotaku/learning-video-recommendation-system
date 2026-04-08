-- 作用：给 sqlc 提供 Learning engine 依赖的外部 schema 最小定义，便于本模块独立生成查询代码。
-- 输入/输出：输入无；输出是 auth、catalog、semantic 外部对象的最小 DDL 描述。
-- 谁调用它：sqlc 生成流程。
-- 它调用谁/传给谁：不在运行时执行；生成器据此解析 query 中的外部引用。
create schema if not exists auth;
create table if not exists auth.users (
  id uuid primary key
);

create schema if not exists catalog;
create table if not exists catalog.videos (
  video_id uuid primary key
);

create schema if not exists semantic;
create table if not exists semantic.coarse_unit (
  id bigint primary key,
  kind text not null,
  label text not null,
  pos text,
  english_def text,
  chinese_def text
);
