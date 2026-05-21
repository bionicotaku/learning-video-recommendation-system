# User 模块设计

## 0. 文档信息

文档状态：MVP 已实现，作为当前模块边界维护。
目标读者：后端 API、User、Learning Engine、Supabase Auth 维护者。
当前范围：定义 `internal/user` 的模块边界、`app_user.user_profiles` 表、Supabase Auth 注册 trigger、邮箱缓存同步、用户活动统计投影、`/api/me` 与 activity calendar 支撑能力。
当前明确不做：不做完整账号系统、不做用户关系、不做权限系统、不做 streak 规则、不做多租户组织模型。

关联文档：

- [API/Me-API-MVP设计.md](API/Me-API-MVP设计.md)
- [API/API模块总体设计规范.md](API/API模块总体设计规范.md)
- [词书系统设计.md](词书系统设计.md)
- [学习引擎设计.md](学习引擎设计.md)

## 1. 一句话结论

Supabase Auth 继续作为身份认证源，业务侧新增轻量 `internal/user` 保存产品资料缓存和用户活动统计投影。

```text
auth.users
  认证身份 source of truth

app_user.user_profiles
  产品资料、onboarding 状态、locale/timezone、email cache

app_user.user_activity_stats
  用户累计活动统计

app_user.user_daily_activity_stats
  用户本地日期维度活动日历统计
```

`auth.users.email` 是邮箱权威来源，`app_user.user_profiles.email` 只是只读缓存。业务 API 不接受 email 修改，邮箱修改必须走 Supabase Auth。

`/api/me` 只读用户级 projection，不直接扫描 `analytics.*`、`catalog.*` 或 `learning.*` 原始表做实时聚合。

## 2. 模块边界

| 模块 | 职责 | 不做什么 |
|---|---|---|
| Supabase Auth / `auth.users` | 登录、注册、JWT subject、邮箱权威来源。 | 不承载业务 profile、学习状态、推荐状态。 |
| `internal/user` | 用户 profile 读取、lazy repair、locale/timezone 更新、onboarding 状态更新、累计和每日活动统计投影。 | 不切换词书、不归约学习状态、不写推荐状态、不解释视频或 quiz 业务规则。 |
| `internal/api` | HTTP handler、principal 解析、header validation、调用 User usecase、错误映射。 | 不直接写 SQL，不拥有 `app_user.*` 表。 |
| `internal/learningengine/reducer` | 当前 active collection 和 `user_unit_states` target projection。 | 不保存 display name、email、timezone。 |
| `internal/catalog` | 观看进度事实和视频用户状态。 | 不拥有 `app_user.*` SQL；只通过 User stats port 写观看统计增量。 |
| `internal/analytics` | quiz / learning interaction raw fact。 | 不拥有 `app_user.*` SQL；只通过 User stats port 写 quiz 和学习互动统计增量。 |

学习目标选择仍归 Learning Engine。`internal/user` 只可以记录用户是否完成 onboarding，例如把 `onboarding_status` 从 `new` 更新为 `collection_selected` 或 `completed`。

活动统计表归 `internal/user` 拥有。Catalog、Analytics、Learning Engine 不直接写 `app_user.*` 表，而是通过 User 模块提供的 tx-aware port 传入已确认的业务增量。

## 3. 数据模型

### 3.1 Schema

新增 schema：

```sql
create schema if not exists app_user;
```

### 3.2 `app_user.user_profiles`

```sql
create table app_user.user_profiles (
  user_id uuid primary key references auth.users(id) on delete cascade,

  email text,
  email_confirmed_at timestamptz,

  display_name text,
  avatar_url text,

  locale text not null default 'zh-CN',
  timezone text,

  onboarding_status text not null default 'new'
    check (onboarding_status in ('new', 'collection_selected', 'completed')),

  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index idx_user_profiles_email
on app_user.user_profiles (email)
where email is not null;
```

字段说明：

| 字段 | 含义 |
|---|---|
| `user_id` | 用户业务主键，对应 `auth.users.id`。 |
| `email` | `auth.users.email` 的缓存，不是权威来源。 |
| `email_confirmed_at` | `auth.users.email_confirmed_at` 的缓存。 |
| `display_name` | App 内展示昵称。注册时默认取 email 的 `@` 前缀，之后由 User 模块维护。 |
| `avatar_url` | App 内头像地址。MVP trigger 默认写 `null`。 |
| `locale` | 用户界面和展示偏好，MVP 默认 `zh-CN`。 |
| `timezone` | IANA timezone name，例如 `Asia/Shanghai`。用于后续本地日期、streak、每日统计。 |
| `onboarding_status` | 新用户初始化状态。 |
| `created_at` / `updated_at` | profile 创建和更新时间。 |

不加 `email unique` 约束。`email` 是 Auth 派生缓存，可能为空，也可能受 Supabase 身份合并、邮箱确认流程、provider 策略影响。缓存字段加唯一约束会增加注册和邮箱同步失败风险。

### 3.3 `app_user.user_activity_stats`

一行表示某个用户的累计活动统计。该表用于 `/api/me`，避免启动路径实时聚合事件表。

```sql
create table app_user.user_activity_stats (
  user_id uuid primary key references auth.users(id) on delete cascade,

  total_watch_ms bigint not null default 0,
  quiz_attempt_count bigint not null default 0,
  started_unit_count bigint not null default 0,

  updated_at timestamptz not null default now(),

  check (total_watch_ms >= 0),
  check (quiz_attempt_count >= 0),
  check (started_unit_count >= 0)
);
```

字段说明：

| 字段 | 含义 |
|---|---|
| `user_id` | 用户 ID。 |
| `total_watch_ms` | 用户累计有效观看时长，单位毫秒。 |
| `quiz_attempt_count` | 用户完成 quiz 的累计次数，幂等重复 quiz event 不重复增加。 |
| `started_unit_count` | 用户历史上第一次让某个 learning unit 的 progress 从 `0` 进入正数的数量，只增不减。 |
| `updated_at` | 统计投影更新时间。 |

`started_unit_count` 不叫 `learned_word_count`。它不是 distinct word 词汇量，也不限定 `semantic.coarse_unit.kind = 'word'`，而是“开始产生学习进度的 learning unit 数量”。

### 3.4 `app_user.user_daily_activity_stats`

一行表示某个用户某个本地日期的活动统计。该表用于 activity calendar。

```sql
create table app_user.user_daily_activity_stats (
  user_id uuid not null references auth.users(id) on delete cascade,

  local_date date not null,
  timezone text not null,

  watch_ms bigint not null default 0,
  quiz_attempt_count bigint not null default 0,
  learning_interaction_count bigint not null default 0,

  first_activity_at timestamptz,
  last_activity_at timestamptz,
  updated_at timestamptz not null default now(),

  primary key (user_id, local_date),

  check (watch_ms >= 0),
  check (quiz_attempt_count >= 0),
  check (learning_interaction_count >= 0)
);

create index idx_user_daily_activity_stats_user_date_desc
on app_user.user_daily_activity_stats (user_id, local_date desc);
```

字段说明：

| 字段 | 含义 |
|---|---|
| `local_date` | 按用户 timezone 派生出的本地日期。 |
| `timezone` | 本行统计写入时使用的 timezone 快照。 |
| `watch_ms` | 当日本地日期内累计有效观看时长，单位毫秒。 |
| `quiz_attempt_count` | 当日本地日期内完成 quiz 次数。 |
| `learning_interaction_count` | 当日本地日期内 exposure / lookup 学习互动次数。 |
| `first_activity_at` | 当日第一条活动对应的 UTC 时间点。 |
| `last_activity_at` | 当日最后一条活动对应的 UTC 时间点。 |

`learning_interaction_count` 是次数统计，不是 distinct word 数，也不是 `started_unit_count` 的每日版本。

## 4. 时间和 Timezone

数据库业务时间点继续统一使用 `timestamptz`：

```text
created_at
updated_at
occurred_at
shown_at
completed_at
started_at
last_seen_at
publish_at
```

`timestamptz` 保存 absolute instant，不保存用户原始时区名称。`app_user.user_profiles.timezone` 只用于把 UTC 时间点派生为用户本地日期。

示例：

```text
occurred_at = 2026-05-22T02:30:00Z
timezone = America/Los_Angeles
local_date = 2026-05-21
```

普通事件上报中的 `*_at` 字段必须自己携带 `Z` 或 offset，不能靠 `timezone` 解释。

每日统计写入时使用当时解析到的 timezone 计算 `local_date`，并把 timezone 快照写入 `app_user.user_daily_activity_stats.timezone`。MVP 不在用户后来修改 timezone 时重算历史日历。

如果 profile timezone 为空，统计写入和 calendar 查询默认使用 `UTC`。

## 5. Onboarding 状态

MVP 状态：

| 状态 | 含义 |
|---|---|
| `new` | 刚注册或 profile 刚补建，还没有完成初始化。 |
| `collection_selected` | 用户已经选择学习目标合集。 |
| `completed` | 用户已完成完整初始化流程。 |

当前如果只要求用户选择一本词书，可以在 `PUT /api/learning-targets/active-collection` 成功后把状态更新为 `collection_selected` 或 `completed`。具体选择由产品流程决定。

User 模块只维护 onboarding 状态本身。词书激活的事务仍在 Learning Engine usecase 内完成。

## 6. Supabase Trigger

### 6.1 注册 Trigger

`after insert on auth.users` 创建 profile。Trigger 必须保持短小、稳定，只做一条 `insert ... on conflict do nothing`。

行为：

```text
user_id = new.id
email = new.email
email_confirmed_at = new.email_confirmed_at
display_name = email @ 前缀
avatar_url = null
locale = 'zh-CN'
timezone = null
onboarding_status = 'new'
```

SQL 草案：

```sql
create or replace function app_user.handle_auth_user_created()
returns trigger
language plpgsql
security definer
set search_path = app_user, auth, public
as $$
begin
  insert into app_user.user_profiles (
    user_id,
    email,
    email_confirmed_at,
    display_name,
    locale,
    onboarding_status
  )
  values (
    new.id,
    new.email,
    new.email_confirmed_at,
    nullif(split_part(coalesce(new.email, ''), '@', 1), ''),
    'zh-CN',
    'new'
  )
  on conflict (user_id) do nothing;

  return new;
end;
$$;

drop trigger if exists on_auth_user_created on auth.users;

create trigger on_auth_user_created
after insert on auth.users
for each row execute function app_user.handle_auth_user_created();
```

### 6.2 邮箱同步 Trigger

`after update of email, email_confirmed_at on auth.users` 只同步邮箱相关缓存，不覆盖 `display_name`。

行为：

```text
email = new.email
email_confirmed_at = new.email_confirmed_at
updated_at = now()
```

SQL 草案：

```sql
create or replace function app_user.handle_auth_user_email_updated()
returns trigger
language plpgsql
security definer
set search_path = app_user, auth, public
as $$
begin
  update app_user.user_profiles
  set
    email = new.email,
    email_confirmed_at = new.email_confirmed_at,
    updated_at = now()
  where user_id = new.id;

  return new;
end;
$$;

drop trigger if exists on_auth_user_email_updated on auth.users;

create trigger on_auth_user_email_updated
after update of email, email_confirmed_at on auth.users
for each row
when (
  old.email is distinct from new.email
  or old.email_confirmed_at is distinct from new.email_confirmed_at
)
execute function app_user.handle_auth_user_email_updated();
```

### 6.3 Trigger 不做事项

Trigger 不做：

```text
不创建 learning.user_unit_states
不自动选择默认词书
不写 analytics
不写 recommendation
不调用外部服务
不根据 raw_user_meta_data 覆盖 profile
不在邮箱更新时覆盖 display_name
```

原因是 Auth trigger 失败会影响注册或邮箱更新，必须把失败面压到最小。

## 7. User Usecase

### 7.1 `GetMe`

```text
GetMe(ctx, user_id, client_timezone)
  -> MeProfile
```

职责：

1. 按 `user_id` 读取 `app_user.user_profiles`。
2. 如果 profile 不存在，从 `auth.users` 读取 email 并补建 profile。
3. 如果 `client_timezone` 合法且和库里不同，更新 `timezone`。
4. 读取或默认补齐 `app_user.user_activity_stats`。
5. 返回 profile 和累计统计。

`GetMe` 是 read + light write usecase。它可能创建缺失 profile，也可能更新 timezone。

### 7.2 `GetActivityCalendar`

```text
GetActivityCalendar(ctx, user_id, client_timezone)
  -> timezone
  -> today
  -> 7 days
```

职责：

1. 解析 timezone：优先合法 `client_timezone`，其次 profile timezone，最后 `UTC`。
2. 按 timezone 计算今天和过去 6 天。
3. 读取 `app_user.user_daily_activity_stats`。
4. 后端补齐没有活动的日期，固定返回 7 天。

`GetActivityCalendar` 只读 daily stats，不更新 profile timezone。

### 7.3 `UpdateProfile`

后续可新增：

```text
UpdateProfile(ctx, user_id, display_name, avatar_url, locale)
```

MVP 可以先不实现。实现时不允许修改 email。

### 7.4 `UpdateOnboardingStatus`

```text
UpdateOnboardingStatus(ctx, user_id, status)
```

可供 API facade 在词书激活成功后调用。该 usecase 只更新 `app_user.user_profiles.onboarding_status`，不参与 Learning Engine 事务。

### 7.5 `ActivityStatsRecorder`

User 模块提供 tx-aware stats recorder。调用方可以把当前事务传给 User recorder，实现业务写入和统计投影同事务提交。

```go
type ActivityStatsRecorder interface {
    AddWatchDuration(ctx context.Context, userID string, activityAt time.Time, deltaWatchMs int64) error
    IncrementQuizAttempt(ctx context.Context, userID string, completedAt time.Time) error
    IncrementStartedUnit(ctx context.Context, userID string) error
    IncrementLearningInteraction(ctx context.Context, userID string, occurredAt time.Time) error
}
```

写入规则：

| 方法 | 调用方 | 规则 |
|---|---|---|
| `AddWatchDuration` | Catalog watch progress | 本次新增有效观看时长 `deltaWatchMs > 0` 时调用；累加 `total_watch_ms` 和 daily `watch_ms`。 |
| `IncrementQuizAttempt` | Analytics quiz writer | `analytics.quiz_events` 幂等插入成功且不是 duplicate 时调用；累加全局和 daily quiz count。 |
| `IncrementStartedUnit` | Learning Engine reducer | `progress_percent` 从 `0` 变成 `> 0` 时调用；只累加全局 `started_unit_count`。 |
| `IncrementLearningInteraction` | Learning interaction normalizer / reducer | exposure / lookup 事件归一化成功时调用；只累加 daily `learning_interaction_count`。 |

User recorder 不判断 Catalog / Analytics / Learning 的业务事实是否成立，只接收调用方已经确认的增量。

### 7.6 事务边界

User stats recorder 必须能复用调用方当前事务。User 模块不在 recorder 内自行 commit。

```text
调用方开启事务
  -> 调用方业务表写入
  -> User stats recorder 使用同一个 tx 更新 app_user projection
  -> 调用方统一 commit / rollback
```

如果当前实现使用通用 `DBTX` 抽象，User repository 可接受 `pgx.Tx` 或 pool：

```go
type DBTX interface {
    Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
```

优先级：

| 指标 | 事务建议 |
|---|---|
| `started_unit_count` | 与 Learning Engine reducer 状态更新同事务。 |
| `quiz_attempt_count` | 与 Analytics quiz insert 同事务。 |
| `total_watch_ms` | 与 Catalog watch progress 同事务；如果现有链路成本较高，可先 best-effort 并保留 rebuild。 |
| `learning_interaction_count` | 与 learning interaction normalization 同事务。 |

## 8. `/api/me` 与 Activity Calendar 支撑策略

`/api/me` 的详细 HTTP 契约见 [API/Me-API-MVP设计.md](API/Me-API-MVP设计.md)。

模块层需要保证：

```text
profile 缺失时可以自愈
timezone 非法时不更新
email cache 只从 auth.users 或 trigger 单向同步
累计 stats 从 app_user.user_activity_stats 读取，缺失时按 0 返回或补建
activity calendar 从 app_user.user_daily_activity_stats 读取，并补齐 7 天空洞
response 不要求 API 层再读 auth.users 或聚合业务表
```

## 9. 实施建议

建议落地顺序：

1. 新增 `internal/user` 目录和 README。
2. 新增 `app_user` migration：schema、`user_profiles`、`user_activity_stats`、`user_daily_activity_stats`、index、trigger function、trigger。
3. 新增 User repository：读取 profile、读取 auth user email、insert repair profile、update timezone、update onboarding status、读取/更新 activity stats。
4. 新增 `GetMe` usecase。
5. 新增 `GetActivityCalendar` usecase。
6. 新增 tx-aware `ActivityStatsRecorder`。
7. 新增 API handler `GET /api/me` 和 `GET /api/me/activity-calendar`。
8. 在 `PUT /api/learning-targets/active-collection` 成功后，可选调用 User usecase 更新 onboarding 状态。
9. 在 Catalog / Analytics / Learning Engine 的对应写入路径接入 User stats recorder。
10. 补单元测试、repository integration、API integration。

## 10. 测试要求

核心测试：

```bash
go test ./internal/user/...
go test ./internal/api/test/integration/me
make quick-check
```

覆盖场景：

- 新注册 Auth user 触发创建 profile。
- profile 的 `display_name` 默认是 email 的 `@` 前缀。
- 邮箱更新只同步 `email` 和 `email_confirmed_at`，不覆盖 `display_name`。
- `GetMe` 在 profile 缺失时可以 lazy repair。
- `GET /api/me` 缺 principal 返回 `401 unauthorized`。
- 合法 `X-Client-Timezone` 会更新 profile。
- 非法 `X-Client-Timezone` 被忽略，不影响 `GET /api/me` 成功。
- `timezone` 不参与任何 `*_at` 字段解析。
- `GET /api/me` 返回累计 `total_watch_seconds`、`quiz_attempt_count`、`started_unit_count`。
- activity stats 缺失时按 0 返回或补建。
- `GET /api/me/activity-calendar` 固定返回今天和过去 6 天，共 7 个日期。
- activity calendar 对没有数据的日期补 0。
- `AddWatchDuration` 同时累加全局和 daily watch。
- `IncrementQuizAttempt` 只在 quiz inserted 时调用，duplicate 不重复增加。
- `IncrementStartedUnit` 只在 progress 从 `0` 到 `>0` 时调用。
- `IncrementLearningInteraction` 按 exposure / lookup 有效事件次数累加 daily count。
