# Unit Collections API MVP 设计

## 0. 文档信息

文档状态：MVP 设计稿，已实现。
目标读者：前端、后端 API、Semantic、Learning Engine、Recommendation 维护者。
当前范围：定义词书列表读取和当前学习目标集合激活两个 API 的契约、字段语义、模块边界、事务要求和数据库访问策略。
当前明确不做：不做词书详情分页、不做词书进度 summary、不做 target required feed guard、不做用户自定义词书、不做多 active collection。

关联文档：

- [API模块总体设计规范.md](API模块总体设计规范.md)
- [../词书系统设计.md](../词书系统设计.md)
- [../学习引擎设计.md](../学习引擎设计.md)
- [../推荐模块设计.md](../推荐模块设计.md)

## 1. API 定位

词书 API 分成两个能力：

```text
GET /api/unit-collections
  读取当前可选学习目标合集，以及当前用户已激活的词书 slug。

PUT /api/learning-targets/active-collection
  把某个词书激活为当前用户的学习目标集合。

GET /api/learning-targets/active-coarse-unit-ids
  读取当前用户仍可用于 exposure 上报过滤的 active target coarse unit ids。
```

`GET /api/unit-collections` 是用户态纯读接口。列表来自 `semantic.unit_collections`，当前选择来自
`learning.user_learning_profiles`，由 API facade 组合后返回。

`PUT /api/learning-targets/active-collection` 是 API facade 编排的 target activation 写接口。它会在一个用户级事务内：

1. 校验目标 collection 存在且 active。
2. 更新 `learning.user_learning_profiles` 当前激活集合。
3. 把不属于新集合的旧词书 target 设为 `is_target = false`。
4. 把新集合 members 批量 upsert 到 `learning.user_unit_states`，并设置 `is_target = true`。
5. 把 `app_user.user_profiles.onboarding_status` 更新为 `collection_selected`。

新注册用户不需要预先存在 `learning.user_learning_profiles` 或 `learning.user_unit_states` 行。首次激活 collection 时，本接口会在同一事务内创建 Learning profile，并为 collection members 批量创建缺失的 unit state。`app_user.user_profiles` 缺失时，API facade 会在同一事务内通过 User repository lazy repair 后再更新 onboarding 状态。

Recommendation 不直接读取词书表，也不接收 `collection_slug`。Recommendation 仍只读取：

```sql
learning.user_unit_states
where is_target = true
  and status in ('new', 'learning', 'reviewing')
```

没有 active collection 时，feed 不返回 `target_required`。Recommendation 可以按无 target 逻辑返回视频。

`GET /api/learning-targets/active-coarse-unit-ids` 不读取完整 `semantic.unit_collection_members`。它读取 Learning Engine 当前 target projection：

```sql
learning.user_unit_states
where is_target = true
  and status in ('new', 'learning', 'reviewing')
```

该接口服务 fullscreen exposure 过滤，`target_count` 是当前 exposure target set 数量，不是词书总词数。

## 2. 模块边界

| 模块 | 职责 | 不做什么 |
|---|---|---|
| `internal/api` | HTTP handler、principal 解析、request validation、错误映射；在 active collection 写接口中编排 Learning Engine target projection 和 User onboarding 的同事务提交。 | 不拥有词书表、learning target 表或 user profile 表；不把 collection members 拉到内存逐条处理。 |
| `internal/semantic` | 拥有词书定义和 membership 读取能力。 | 不写用户学习状态，不理解 Recommendation 排序。 |
| `internal/learningengine/reducer` | 拥有用户当前 active collection 和 `user_unit_states` target 投影。 | 不拥有词书定义，不生成推荐列表。 |
| `internal/user` | 拥有 user profile 和 onboarding 状态。 | 不切换词书、不写 learning target 投影。 |
| `internal/recommendation` | 只读 `learning.user_unit_states` 生成推荐计划。 | 不接触 `unit_collections`，不处理 active collection。 |

`PUT /api/learning-targets/active-collection` 的事务由 API facade 打开，因为它需要同时更新 Learning Engine 和 User 两个模块资产。事务内仍调用各模块 repository port；API 层不能先把几千个 members 拉到内存后逐条调用 `EnsureTargetUnits`，也不能直接绕过模块 SQL owner 写表。

## 3. GET /api/unit-collections

### 3.1 Endpoint

```http
GET /api/unit-collections
```

`user_id` 必须从 trusted principal 获取，不允许 body/query/path 传入。MVP 返回所有 active collections，
并额外返回当前用户有效 active collection 的 `slug`；如果用户没有 learning profile，或 profile 指向的 collection
已经 inactive / missing，则返回 `active_collection: null`。

### 3.2 Response

```ts
type UnitCollectionsResponse = {
  items: UnitCollectionItem[];
  active_collection: string | null;
};

type UnitCollectionItem = {
  collection_id: string;
  slug: string;
  name: string;
  description: string | null;
  category: string;
  coarse_unit_count: number;
  word_unit_count: number;
};
```

示例：

```json
{
  "items": [
    {
      "collection_id": "11111111-1111-4111-8111-111111111111",
      "slug": "toefl-1000-essential",
      "name": "TOEFL 1000 Essential",
      "description": "Core TOEFL vocabulary for short-video learning.",
      "category": "wordbook",
      "coarse_unit_count": 1000,
      "word_unit_count": 1000
    },
    {
      "collection_id": "22222222-2222-4222-8222-222222222222",
      "slug": "ielts-core",
      "name": "IELTS Core",
      "description": null,
      "category": "wordbook",
      "coarse_unit_count": 1800,
      "word_unit_count": 1640
    }
  ],
  "active_collection": "toefl-1000-essential"
}
```

### 3.3 字段说明

| 字段 | 来源 | 说明 |
|---|---|---|
| `collection_id` | `semantic.unit_collections.collection_id` | 词书稳定 ID。前端一般不解析，但可以作为 key。 |
| `slug` | `semantic.unit_collections.slug` | API 使用的可读唯一标识，例如 `toefl`、`ielts-core`。 |
| `name` | `semantic.unit_collections.name` | 展示名。 |
| `description` | `semantic.unit_collections.description` | 词书说明，可为空。 |
| `category` | `semantic.unit_collections.category` | 集合类型。MVP 通常为 `wordbook`。 |
| `coarse_unit_count` | `semantic.unit_collections.coarse_unit_count` | 该 collection 在 `semantic.unit_collection_members` 中的实际 member 条目数。 |
| `word_unit_count` | `semantic.unit_collections.word_unit_count` | 来源词书中去重后、且至少匹配到一个 `semantic.coarse_unit` 的 `headWord` 数量。 |
| `active_collection` | `learning.user_learning_profiles.active_collection_slug` | 当前用户已激活词书的 slug。只有该 slug 对应的 collection 仍在本次 active `items` 中时返回；否则返回 `null`。 |

`word_unit_count` 只统计有实际 coarse unit 匹配的去重 `headWord`，未匹配词条不计入。
`coarse_unit_count` 与成员表实际行数保持一致；由于 `semantic.unit_collection_members`
的主键是 `(collection_id, coarse_unit_id)`，同一个 coarse unit 通过多个词条命中时只算一个 member。

### 3.4 查询策略

```sql
select
  collection_id,
  slug,
  name,
  description,
  category,
  coarse_unit_count,
  word_unit_count
from semantic.unit_collections
where status = 'active'
order by category asc, name asc, slug asc;
```

MVP 不分页。词书数量通常很小，直接返回全部 active collections。

API facade 另外按 principal `user_id` 从 Learning Engine 读取当前 active collection：

```sql
select
  active_collection_id,
  active_collection_slug
from learning.user_learning_profiles
where user_id = $1;
```

没有 profile row 不是错误，映射为 `active_collection: null`。如果读取到的 `active_collection_id` /
`active_collection_slug` 不在 active collection 列表中，也映射为 `null`，避免历史脏状态阻断词书列表读取。

## 4. PUT /api/learning-targets/active-collection

### 4.1 Endpoint

```http
PUT /api/learning-targets/active-collection
Content-Type: application/json
```

`user_id` 必须从 trusted principal 获取，不允许 body/query/path 传入。

### 4.2 Request

```ts
type ActivateUnitCollectionRequest = {
  collection_slug: string;
};
```

示例：

```json
{
  "collection_slug": "toefl-1000-essential"
}
```

规则：

- `collection_slug` 必填。
- 只能包含小写字母、数字、连字符，建议正则为 `^[a-z0-9][a-z0-9-]{0,80}$`。
- 不能由前端传入 `collection_id`、`coarse_unit_ids`、`target_priority`。

### 4.3 Response

```ts
type ActivateUnitCollectionResponse = {
  collection_id: string;
  collection_slug: string;
  target_count: number;
};
```

示例：

```json
{
  "collection_id": "11111111-1111-4111-8111-111111111111",
  "collection_slug": "toefl-1000-essential",
  "target_count": 1000
}
```

`target_count` 表示本次激活集合包含的 member 数量。MVP 可以直接使用 `semantic.unit_collections.coarse_unit_count`，也可以从本次 upserted members count 返回；两者应在数据维护正确时一致。

### 4.4 业务语义

本接口是幂等 set，不是 toggle。

本接口是同步提交接口，不是异步 job：

- 成功响应使用 `200 OK`，表示 active collection、`learning.user_unit_states` target projection、以及 `app_user.user_profiles.onboarding_status = collection_selected` 已经在同一个事务内提交完成。
- 前端收到 `200 OK` 后，可以立即刷新 `/api/me`、`/api/feed` 或 unit progress；这些接口不应再读到旧 collection 的最终 target projection。
- 请求处理中前端应保持提交态或 loading 态，不要提前假设切换完成。
- MVP 不返回 `202 Accepted`、`activation_id`、`processing` 状态，也不提供后台轮询接口。

如果未来大规模词书导致同步接口持续超时，再单独设计 activation job 表、worker、状态查询 API 和 feed 在 switching 状态下的行为。MVP 阶段优先优化 set-based SQL、索引和 no-op 更新，保持成功语义简单明确。

重复激活同一本词书：

- 返回 `200 OK`。
- `learning.user_learning_profiles` 仍指向同一 collection。
- `learning.user_unit_states` target projection 最终一致。

首次激活词书：

- 如果 `learning.user_learning_profiles` 中没有当前用户行，则创建。
- 如果某个 collection member 在 `learning.user_unit_states` 中没有当前用户状态行，则创建默认 `status = new` 的状态行。
- 已有状态行只更新 target control 字段，不重置学习进度。

从雅思切换到托福：

- 不属于托福的新旧词书 target 被设为 `is_target = false`。
- 托福 members 被批量 upsert 为 `is_target = true`。
- 雅思和托福重合的 unit 不先 false 再 true，只保持或更新为新 collection target。
- 所有已有学习状态、进度、掌握度、schedule、历史计数都保留。

本接口只更新 control 字段：

```text
is_target
target_source
target_source_ref_id
target_priority
updated_at
```

本接口不更新：

```text
status
progress_percent
mastery_score
next_review_at
schedule_*
observation_count
progress_event_count
recent_progress_*
```

MVP 中 `target_priority` 固定写 `0`。后续如果启用 collection-level priority，可以从 `semantic.unit_collection_members.target_priority` 复制。

## 5. 数据库访问与事务

### 5.1 原则

`PUT /api/learning-targets/active-collection` 必须使用 set-based SQL：

- 不按 member 逐条查询。
- 不按 member 逐条 upsert。
- 不把全部 `coarse_unit_ids` 从 Semantic 拉到 API 层再传给 Learning Engine。
- 不循环调用现有 `EnsureTargetUnits`。

API facade 应使用现有 user-scoped transaction 机制，保证同一用户并发切换串行化，并把 Learning Engine target projection 与 User onboarding 状态作为一个提交单元。

### 5.2 推荐执行计划

API facade 提供组合 usecase：

```text
ActivateUnitCollectionTarget(user_id, collection_slug)
```

事务内执行：

1. `selected_collection`：按 slug 找 active collection。
2. `new_members`：读取该 collection 下所有 members。
3. `profile_upsert`：更新 `learning.user_learning_profiles`。
4. `deactivated`：只关闭不属于新 collection 的旧 `unit_collection` targets。
5. `upserted`：批量 upsert 新 collection members 到 `learning.user_unit_states`。
6. `user_profile_repair`：如果 `app_user.user_profiles` 缺失，从 `auth.users` 补建 profile。
7. `onboarding_update`：把 `app_user.user_profiles.onboarding_status` 设为 `collection_selected`。
8. 返回 collection id、slug、target count。

### 5.3 推荐 SQL 形态

以下 SQL 是设计形态，最终可以按 sqlc 约束拆分或调整：

```sql
with selected_collection as (
  select
    collection_id,
    slug,
    coarse_unit_count
  from semantic.unit_collections
  where slug = sqlc.arg(collection_slug)
    and status = 'active'
),
new_members as (
  select
    m.collection_id,
    m.coarse_unit_id,
    0::numeric as target_priority
  from semantic.unit_collection_members m
  join selected_collection c
    on c.collection_id = m.collection_id
),
profile_upsert as (
  insert into learning.user_learning_profiles (
    user_id,
    active_collection_id,
    active_collection_slug,
    active_collection_activated_at,
    updated_at
  )
  select
    sqlc.arg(user_id),
    collection_id,
    slug,
    now(),
    now()
  from selected_collection
  on conflict (user_id) do update
  set
    active_collection_id = excluded.active_collection_id,
    active_collection_slug = excluded.active_collection_slug,
    active_collection_activated_at = now(),
    updated_at = now()
  returning user_id
),
deactivated as (
  update learning.user_unit_states s
  set
    is_target = false,
    updated_at = now()
  where s.user_id = sqlc.arg(user_id)
    and s.target_source = 'unit_collection'
    and s.is_target = true
    and not exists (
      select 1
      from new_members nm
      where nm.coarse_unit_id = s.coarse_unit_id
    )
  returning s.coarse_unit_id
),
upserted as (
  insert into learning.user_unit_states (
    user_id,
    coarse_unit_id,
    is_target,
    target_source,
    target_source_ref_id,
    target_priority
  )
  select
    sqlc.arg(user_id),
    nm.coarse_unit_id,
    true,
    'unit_collection',
    nm.collection_id::text,
    nm.target_priority
  from new_members nm
  on conflict (user_id, coarse_unit_id) do update
  set
    is_target = true,
    target_source = 'unit_collection',
    target_source_ref_id = excluded.target_source_ref_id,
    target_priority = excluded.target_priority,
    updated_at = now()
  returning coarse_unit_id
)
select
  c.collection_id,
  c.slug,
  c.coarse_unit_count::integer as target_count
from selected_collection c;
```

如果 `selected_collection` 为空，query 返回 0 rows，repository 映射为 `ErrUnitCollectionNotFound`，API 返回 `404 not_found`。

### 5.4 重合 unit 优化

deactivate 条件使用 `not exists (select 1 from new_members ...)`，避免重合 unit 被先关闭再开启。

upsert 新 members 仍会触达重合 unit，用于更新：

```text
target_source = 'unit_collection'
target_source_ref_id = new collection_id
target_priority = 0
```

这比 API 层 diff 两个大集合更简单，也避免多次网络 roundtrip。

## 6. 错误处理

### 6.1 `GET /api/unit-collections`

| HTTP | code | 场景 |
|---|---|---|
| `200 OK` | - | 列表读取成功；没有 active profile 时也是成功。 |
| `401 Unauthorized` | `unauthorized` | trusted principal 缺失。 |
| `500 Internal Server Error` | `internal_error` | 数据库或未知服务端错误。 |

### 6.2 `PUT /api/learning-targets/active-collection`

| HTTP | code | 场景 |
|---|---|---|
| `200 OK` | - | active collection 设置成功。 |
| `400 Bad Request` | `invalid_request` | JSON 非法、未知字段、`collection_slug` 缺失或格式非法。 |
| `401 Unauthorized` | `unauthorized` | trusted principal 缺失。 |
| `404 Not Found` | `not_found` | collection 不存在或 inactive。 |
| `500 Internal Server Error` | `internal_error` | 数据库或未知服务端错误。 |

`GET /api/unit-collections` 有 principal 但没有 `learning.user_learning_profiles` row 时仍返回 `200 OK`，
并返回 `active_collection: null`。

## 7. 与 Feed / Recommendation 的关系

本 API 不控制 feed 是否可用。

无 active collection 时：

```text
POST /api/feed
  -> Recommendation 无 target / fill 逻辑
  -> 仍可返回视频
```

有 active collection 时：

```text
PUT /api/learning-targets/active-collection
  -> learning.user_unit_states target projection 更新
POST /api/feed
  -> Recommendation 读取 is_target=true 且未掌握 units
```

Feed API 不需要 `target_required` guard。

## 8. 测试计划

API integration：

- `GET /api/unit-collections` 需要 principal。
- `GET /api/unit-collections` 返回 active collections 和 `active_collection` 字段。
- 没有 `learning.user_learning_profiles` row 时返回 `active_collection: null`。
- 已有有效 active collection 时返回对应 slug。
- profile 指向 inactive / missing collection 时返回 `active_collection: null`。
- inactive collection 不返回。
- `PUT /api/learning-targets/active-collection` 首次激活成功。
- 重复激活同一 collection 幂等。
- 切换 collection 后旧 collection 独有 unit `is_target=false`。
- 切换 collection 后新 collection unit `is_target=true`。
- 重合 unit 保留学习状态，并最终指向新 collection。
- `status / progress_percent / mastery_score / schedule_*` 不被重置。
- unknown collection 返回 `404 not_found`。
- invalid slug 返回 `400 invalid_request`。
- missing principal 返回 `401 unauthorized`。
- 激活成功时 `app_user.user_profiles.onboarding_status` 同事务更新为 `collection_selected`。
- onboarding 更新失败时，`learning.user_learning_profiles` 和 `learning.user_unit_states` 写入一起回滚。

Learning Engine integration：

- Learning Engine repository 单次调用完成 profile upsert、old target deactivate、new target upsert。
- 并发切换同一用户时最终状态一致。
- members 数量较大时仍只使用批量 SQL，不出现 per-unit roundtrip。

Repository / SQL：

- `selected_collection` 为空时返回 no rows。
- 空 collection 可被激活，`target_count=0`，同时关闭旧 collection targets。
- `target_source != 'unit_collection'` 的非词书 target 不被本接口关闭。

## 9. 非目标

MVP 不做：

- 用户自定义词书。
- 多个 active collections。
- 词书详情分页接口。
- active collection progress summary。
- collection member 管理后台 API。
- collection import audit。
- 根据 collection 强制 feed target required。
