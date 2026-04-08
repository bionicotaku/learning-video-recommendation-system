-- 文件作用：
--   - 为 Recommendation 的 sqlc 编译面补充外部依赖 schema
-- 输入/输出：
--   - 输入：无，作为 sqlc schema 解析输入
--   - 输出：让 sqlc 知道 auth.users 和 semantic.coarse_unit 的结构
-- 谁调用它：
--   - sqlc 根据 sqlc.yaml 读取它
-- 它调用谁/传给谁：
--   - 不直接执行到业务链路；主要传给 sqlc 做静态编译
create schema if not exists auth;
create table if not exists auth.users (
  id uuid primary key
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
