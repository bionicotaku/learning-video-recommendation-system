# Me API MVP 设计

## 0. 文档信息

文档状态：MVP 已实现，作为当前 API 契约维护。
目标读者：前端、后端 API、User、Supabase Auth 维护者。
当前范围：定义 `GET /api/me` 与 `GET /api/me/activity-calendar` 的请求、响应、profile lazy repair、timezone 顺手更新、活动统计读取、错误语义和模块边界。
当前明确不做：不做用户资料编辑 API、不做邮箱修改、不做头像上传、不做登录注册 API、不做 streak 规则。

关联文档：

- [../User模块设计.md](../User模块设计.md)
- [API模块总体设计规范.md](API模块总体设计规范.md)
- [Unit-Collections-API-MVP设计.md](Unit-Collections-API-MVP设计.md)

## 1. API 定位

`GET /api/me` 返回当前登录用户的基础 profile 和累计活动统计，用于 App 启动、首页初始化、onboarding 判断和本地时区同步。

`GET /api/me/activity-calendar` 返回今天和过去 6 天的活动日历统计，用于个人页、学习日历或连续活跃 UI。

本接口从 trusted principal 获取 `user_id`，不接受 body/query/path 中的用户 ID。

MVP 中 `GET /api/me` 不是纯读接口。它可能做两个轻量写入：

```text
1. 如果 app_user.user_profiles 缺失，补建 profile。
2. 如果 header 携带合法 timezone，且和库里不同，更新 timezone。
```

两个接口都只读取 User 模块的 projection，不直接扫描 `analytics.*`、`catalog.*` 或 `learning.*` 原始表聚合。

## 2. Endpoint

```http
GET /api/me
```

```http
GET /api/me/activity-calendar
```

可选 header：

```http
X-Client-Timezone: Asia/Shanghai
```

MVP 不需要 request body。

## 3. Request Header

| Header | 必填 | 说明 |
|---|---|---|
| `X-Client-Timezone` | 否 | 前端设备当前 IANA timezone name，例如 `Asia/Shanghai`、`America/Los_Angeles`。 |

前端可通过浏览器或原生运行时获取 timezone。Web 示例：

```ts
Intl.DateTimeFormat().resolvedOptions().timeZone
```

后端只接受 IANA timezone name，不接受 `+08:00` 这类 offset。

非法 timezone 的处理：

```text
忽略 header
不更新 profile
仍返回 200 OK
```

原因是 `/api/me` 是启动路径，设备时区异常不应导致用户无法进入 App。

## 4. Response

```ts
type MeResponse = {
  user_id: string;
  email: string | null;
  email_confirmed: boolean;
  display_name: string | null;
  avatar_url: string | null;
  locale: string;
  timezone: string | null;
  onboarding_status: "new" | "collection_selected" | "completed";
  stats: MeStats;
};

type MeStats = {
  total_watch_seconds: number;
  quiz_attempt_count: number;
  started_unit_count: number;
};
```

示例：

```json
{
  "user_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  "email": "alice@example.com",
  "email_confirmed": true,
  "display_name": "alice",
  "avatar_url": null,
  "locale": "zh-CN",
  "timezone": "Asia/Shanghai",
  "onboarding_status": "new",
  "stats": {
    "total_watch_seconds": 3600,
    "quiz_attempt_count": 12,
    "started_unit_count": 48
  }
}
```

字段说明：

| 字段 | 来源 | 说明 |
|---|---|---|
| `user_id` | trusted principal / `app_user.user_profiles.user_id` | 当前用户 ID。 |
| `email` | `app_user.user_profiles.email` | `auth.users.email` 的缓存。 |
| `email_confirmed` | `email_confirmed_at is not null` | 邮箱是否已确认。 |
| `display_name` | `app_user.user_profiles.display_name` | 用户展示昵称。 |
| `avatar_url` | `app_user.user_profiles.avatar_url` | 头像地址，MVP 可为空。 |
| `locale` | `app_user.user_profiles.locale` | 展示语言/地区偏好，MVP 默认 `zh-CN`。 |
| `timezone` | `app_user.user_profiles.timezone` | 用户 IANA timezone。 |
| `onboarding_status` | `app_user.user_profiles.onboarding_status` | 新用户初始化状态。 |
| `stats.total_watch_seconds` | `app_user.user_activity_stats.total_watch_ms` | 用户累计有效观看时长，向下取整为秒。 |
| `stats.quiz_attempt_count` | `app_user.user_activity_stats.quiz_attempt_count` | 用户累计完成 quiz 次数。 |
| `stats.started_unit_count` | `app_user.user_activity_stats.started_unit_count` | 用户历史上第一次让 learning unit 产生正进度的数量。 |

`started_unit_count` 不是 learned word 数，也不是每日学习次数。它表示“有过学习进度的 learning unit 数量”，只增不减。

## 5. 处理流程

```text
1. API auth middleware 从 trusted principal 解析 user_id。
2. Handler 读取可选 X-Client-Timezone。
3. Handler 调用 User.GetMe(user_id, client_timezone)。
4. User usecase 读取 app_user.user_profiles。
5. 如果 profile 不存在，从 auth.users 读取 email / email_confirmed_at，并补建 profile。
6. 如果 client_timezone 合法且与 profile.timezone 不同，更新 timezone。
7. 读取 `app_user.user_activity_stats`；缺失时按 0 返回或补建默认行。
8. 返回 MeResponse。
```

### 5.1 Lazy Repair

Trigger 正常工作时，新用户注册会自动创建 `app_user.user_profiles`。但 `/api/me` 仍必须支持 lazy repair，防止历史用户、导入用户或 trigger 异常导致 profile 缺失。

补建规则：

```text
email = auth.users.email
email_confirmed_at = auth.users.email_confirmed_at
display_name = email @ 前缀
avatar_url = null
locale = 'zh-CN'
timezone = null
onboarding_status = 'new'
```

如果 `auth.users` 中也找不到该 `user_id`，返回 `401 unauthorized` 或 `404 not_found` 均可实现。MVP 推荐返回 `401 unauthorized`，因为 principal 指向了不存在的身份。

### 5.2 Timezone 更新

后端用 Go 校验 timezone：

```go
_, err := time.LoadLocation(clientTimezone)
```

更新 SQL 语义：

```sql
update app_user.user_profiles
set timezone = $2,
    updated_at = now()
where user_id = $1
  and timezone is distinct from $2;
```

MVP 删除 `timezone_source` 字段，所以 `/api/me` 可以自动覆盖之前保存的 timezone。后续如果支持用户手动设置时区，再补 `timezone_source` 或拆出设置 API。

## 6. GET /api/me/activity-calendar

### 6.1 Endpoint

```http
GET /api/me/activity-calendar
```

可选 header：

```http
X-Client-Timezone: Asia/Shanghai
```

本接口固定返回“今天 + 过去 6 天”，共 7 天。MVP 不接受 `from/to` query，避免前端传入过大范围，也降低分页和补洞复杂度。

### 6.2 Response

```ts
type ActivityCalendarResponse = {
  timezone: string;
  today: string;
  days: ActivityDay[];
};

type ActivityDay = {
  local_date: string;
  watch_seconds: number;
  quiz_attempt_count: number;
  learning_interaction_count: number;
  is_active: boolean;
};
```

示例：

```json
{
  "timezone": "Asia/Shanghai",
  "today": "2026-05-22",
  "days": [
    {
      "local_date": "2026-05-16",
      "watch_seconds": 420,
      "quiz_attempt_count": 3,
      "learning_interaction_count": 8,
      "is_active": true
    },
    {
      "local_date": "2026-05-17",
      "watch_seconds": 0,
      "quiz_attempt_count": 0,
      "learning_interaction_count": 0,
      "is_active": false
    }
  ]
}
```

字段说明：

| 字段 | 来源 | 说明 |
|---|---|---|
| `timezone` | request header / profile / default | 本次计算 today 和日期窗口使用的 timezone。 |
| `today` | 服务端按 timezone 计算 | 当前本地日期，格式 `YYYY-MM-DD`。 |
| `days[].local_date` | 日期窗口补齐 | 本地日期，格式 `YYYY-MM-DD`。 |
| `days[].watch_seconds` | `app_user.user_daily_activity_stats.watch_ms` | 当日有效观看时长，向下取整为秒。 |
| `days[].quiz_attempt_count` | `app_user.user_daily_activity_stats.quiz_attempt_count` | 当日完成 quiz 次数。 |
| `days[].learning_interaction_count` | `app_user.user_daily_activity_stats.learning_interaction_count` | 当日 exposure / lookup 学习互动次数。 |
| `days[].is_active` | API 计算 | 三个统计任一大于 0 即为 true。 |

`days` 固定 7 个元素，按 `local_date` 升序返回。没有 activity stats 行的日期必须补 0。

### 6.3 Timezone 选择

timezone 选择顺序：

```text
1. 合法 X-Client-Timezone
2. app_user.user_profiles.timezone
3. UTC
```

和 `/api/me` 不同，本接口只使用 timezone，不更新 `app_user.user_profiles.timezone`。原因是 activity calendar 是只读查询，避免隐式副作用扩散到多个 GET endpoint。

非法 `X-Client-Timezone` 直接忽略，继续使用 profile timezone 或 UTC。

### 6.4 查询逻辑

后端按 timezone 计算：

```text
today = now().In(location).Date()
from = today - 6 days
to = today
```

查询：

```sql
select
  local_date,
  watch_ms,
  quiz_attempt_count,
  learning_interaction_count
from app_user.user_daily_activity_stats
where user_id = $1
  and local_date between $2 and $3
order by local_date asc;
```

后端补齐缺失日期，保证 response shape 稳定。

## 7. 错误

| HTTP | code | 场景 |
|---|---|---|
| `200 OK` | 无 | 成功返回当前用户 profile 或 activity calendar。 |
| `401 Unauthorized` | `unauthorized` | trusted principal 缺失、无法解析，或 principal 指向不存在的 Auth user。 |
| `500 Internal Server Error` | `internal_error` | 数据库错误或未知服务端错误。 |

非法 `X-Client-Timezone` 不返回 `400`，直接忽略。

## 8. 模块边界

| 层 | 职责 |
|---|---|
| `internal/api` | 解析 principal、读取 header、调用 User usecase、返回 JSON。 |
| `internal/user` | 读取和修复 profile、校验 timezone、更新 timezone、读取累计和每日活动 stats、返回应用 DTO。 |
| Supabase Auth | 身份源和 email 权威来源。 |
| `internal/catalog` | 在观看进度写入路径通过 User stats port 写 watch 增量。 |
| `internal/analytics` | 在 quiz 和 learning interaction 写入/归一化路径通过 User stats port 写统计增量。 |
| `internal/learningengine/reducer` | 在 learning unit progress 首次从 0 到正数时通过 User stats port 写 `started_unit_count`。 |

`GET /api/me` 不返回 active collection。当前词书状态属于 Learning Engine target control。前端如需词书列表和选择状态，使用词书 API 或后续专门的 learning target summary API。

## 9. 数据库访问

`GET /api/me` 正常路径：

```sql
select
  user_id,
  email,
  email_confirmed_at,
  display_name,
  avatar_url,
  locale,
  timezone,
  onboarding_status
from app_user.user_profiles
where user_id = $1;
```

累计 stats：

```sql
select
  total_watch_ms,
  quiz_attempt_count,
  started_unit_count
from app_user.user_activity_stats
where user_id = $1;
```

如果 stats 缺失，可以按 0 返回，或执行：

```sql
insert into app_user.user_activity_stats (user_id)
values ($1)
on conflict (user_id) do nothing;
```

profile 缺失时：

```sql
select email, email_confirmed_at
from auth.users
where id = $1;
```

然后：

```sql
insert into app_user.user_profiles (
  user_id,
  email,
  email_confirmed_at,
  display_name,
  locale,
  onboarding_status
)
values (
  $1,
  $2,
  $3,
  nullif(split_part(coalesce($2, ''), '@', 1), ''),
  'zh-CN',
  'new'
)
on conflict (user_id) do nothing;
```

如果发生并发 lazy repair，`on conflict do nothing` 后重新读取 profile 即可。

`GET /api/me/activity-calendar` 只读：

```sql
select
  local_date,
  watch_ms,
  quiz_attempt_count,
  learning_interaction_count
from app_user.user_daily_activity_stats
where user_id = $1
  and local_date between $2 and $3
order by local_date asc;
```

## 10. 活动统计写入语义

活动统计由 `internal/user` 拥有，其他模块通过 tx-aware User stats port 写入。API 不直接写 stats。

| 统计 | 来源模块 | 写入语义 |
|---|---|---|
| `total_watch_ms` / daily `watch_ms` | Catalog watch progress | 本次新增有效观看时长 `delta_watch_ms > 0` 时累加。 |
| `quiz_attempt_count` | Analytics quiz writer | `analytics.quiz_events` 幂等插入成功且不是 duplicate 时累加。 |
| `started_unit_count` | Learning Engine reducer | learning unit progress 从 `0` 到 `>0` 时累加，只增不减。 |
| daily `learning_interaction_count` | Learning interaction normalizer / reducer | exposure / lookup 事件归一化成功时累加。 |

每日统计的 `local_date` 由事件时间点按用户 timezone 派生。MVP 对跨午夜观看不做精确拆分，watch delta 归到本次 watch progress 的 activity time 所在本地日期。

## 11. 前端调用建议

App 启动后先调用：

```http
GET /api/me
X-Client-Timezone: Asia/Shanghai
```

根据返回的 `onboarding_status` 决定页面：

| `onboarding_status` | 前端行为 |
|---|---|
| `new` | 引导用户选择词书。 |
| `collection_selected` | 可以进入主流程，或继续完成其他初始化步骤。 |
| `completed` | 直接进入主流程。 |

如果用户选择词书，调用：

```http
PUT /api/learning-targets/active-collection
```

后端可以在词书激活成功后更新 `onboarding_status`。

个人页或日历组件调用：

```http
GET /api/me/activity-calendar
X-Client-Timezone: Asia/Shanghai
```

前端不需要传日期范围。后端固定返回 7 天并补齐空日期。

## 12. 不做事项

MVP 不做：

```text
PATCH /api/me
上传头像
修改邮箱
修改密码
返回 active collection
返回 streak
返回推荐摘要
activity calendar 自定义日期范围
根据 raw_user_meta_data 同步昵称
```

这些能力后续单独设计，避免 `/api/me` 变成过重的首页聚合接口。

## 13. 测试计划

目标测试：

```bash
go test ./internal/api/test/integration/me
go test ./internal/user/...
make quick-check
```

覆盖场景：

- 缺 principal 返回 `401 unauthorized`。
- 已存在 profile 时返回完整 `MeResponse`。
- profile 缺失时 lazy repair 并返回默认字段。
- `display_name` 默认使用 email 的 `@` 前缀。
- `locale` 默认 `zh-CN`。
- 合法 `X-Client-Timezone` 更新 `timezone`。
- 重复传同一 timezone 不产生额外业务变化。
- 非法 `X-Client-Timezone` 被忽略，仍返回 `200 OK`。
- `/api/me` 返回 `stats.total_watch_seconds`、`quiz_attempt_count`、`started_unit_count`。
- stats 缺失时返回 0 或补建默认 stats 行。
- `/api/me/activity-calendar` 缺 principal 返回 `401 unauthorized`。
- `/api/me/activity-calendar` 固定返回 7 天，日期升序。
- `/api/me/activity-calendar` 对缺失日期补 0。
- `/api/me/activity-calendar` 使用 header timezone 计算 today，但不更新 profile timezone。
- `is_active` 在三个活动统计任一大于 0 时为 true。
- response 不包含 password、raw metadata、provider identity、Auth token。
