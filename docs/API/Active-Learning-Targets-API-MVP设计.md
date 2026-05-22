# Active Learning Targets API MVP 设计

## 0. 文档信息

文档状态：MVP 设计稿，已实现。
目标读者：前端、后端 API、Learning Engine 维护者。
当前范围：定义当前用户 active learning target coarse unit id 列表读取 API，用于 fullscreen 字幕 exposure 上报过滤。
当前明确不做：不分页、不返回 unit 文本或学习进度、不读取完整词书 members、不做 revision / ETag / 离线同步。

关联文档：

- [API模块总体设计规范.md](API模块总体设计规范.md)
- [Unit-Collections-API-MVP设计.md](Unit-Collections-API-MVP设计.md)
- [学习事件上报API设计.md](学习事件上报API设计.md)

## 1. API 定位

```http
GET /api/learning-targets/active-coarse-unit-ids
```

本 API 返回当前用户仍可用于 exposure 上报过滤的 target coarse unit id 集合。它读取 Learning Engine 当前 target 投影：

```sql
learning.user_unit_states
where user_id = current_user
  and is_target = true
  and status <> 'mastered'
```

`active_collection` 只从 `learning.user_learning_profiles.active_collection_slug` 读取；没有 learning profile 时返回 `null` 和空列表。

本 API 不是词书完整 members API。`target_count` 是当前 exposure target set 数量，不是词书总词数；词书总量继续使用 `GET /api/unit-collections` 返回的 `coarse_unit_count / word_unit_count`。

## 2. Endpoint

```http
GET /api/learning-targets/active-coarse-unit-ids
Authorization: Bearer <token>
Accept: application/json
```

请求规则：

- 不接受 request body。
- 不接受 `user_id`、`collection_slug` 或分页参数。
- `user_id` 必须从 trusted principal 获取。

## 3. Response

```ts
type ActiveLearningTargetCoarseUnitIdsResponse = {
  active_collection: string | null;
  target_count: number;
  coarse_unit_ids: number[];
};
```

字段说明：

| 字段 | 说明 |
|---|---|
| `active_collection` | 当前用户 active collection slug；没有 learning profile 时为 `null`。 |
| `target_count` | `coarse_unit_ids.length`。表示当前仍可用于 exposure 上报的未 mastered target 数量。 |
| `coarse_unit_ids` | 当前用户 `is_target=true AND status!='mastered'` 的 coarse unit ids，升序返回。 |

无 active profile：

```json
{
  "active_collection": null,
  "target_count": 0,
  "coarse_unit_ids": []
}
```

有 active collection 但没有未 mastered target：

```json
{
  "active_collection": "toefl-core",
  "target_count": 0,
  "coarse_unit_ids": []
}
```

正常示例：

```json
{
  "active_collection": "toefl-core",
  "target_count": 3,
  "coarse_unit_ids": [101, 205, 309]
}
```

## 4. Errors

| HTTP | 场景 |
|---|---|
| `200 OK` | 成功返回，包含空列表也是成功。 |
| `401 unauthorized` | trusted principal 缺失。 |
| `500 internal_error` | 数据库或未知服务端错误。 |

## 5. 处理流程

```text
1. API auth middleware 从 trusted principal 解析 user_id。
2. Handler 调用 Learning Engine reducer read usecase。
3. Usecase 读取 active collection slug 和当前 target ids。
4. 没有 learning profile 时返回 active_collection=null 和空数组。
5. 返回 snake_case JSON。
```

数据库读取：

```sql
with profile as (
  select active_collection_slug
  from learning.user_learning_profiles
  where user_id = $1
),
targets as (
  select coalesce(array_agg(coarse_unit_id order by coarse_unit_id), '{}'::bigint[]) as coarse_unit_ids
  from learning.user_unit_states
  where user_id = $1
    and is_target = true
    and status <> 'mastered'
)
select
  coalesce((select active_collection_slug from profile), '')::text as active_collection_slug,
  coalesce((select coarse_unit_ids from targets), '{}'::bigint[])::bigint[] as coarse_unit_ids,
  exists(select 1 from profile) as has_active_profile;
```

`has_active_profile=false` 时，后端必须忽略 target rows 并返回空列表。

## 6. 模块边界

| 模块 | 职责 |
|---|---|
| `internal/api` | HTTP route、principal、错误映射和 JSON response。 |
| `internal/learningengine/reducer` | 拥有 `learning.user_learning_profiles` / `learning.user_unit_states` read model 和 usecase。 |
| `internal/semantic` | 不参与本 API；本 API 不读取完整 collection members。 |
| `internal/recommendation` | 不参与本 API；exposure 过滤不是推荐召回。 |

## 7. 前端调用语义

- Fullscreen 首次需要 target set 时调用本 API。
- API 失败时前端 fail closed，不上报 exposure，不阻断视频播放。
- 词书切换成功后前端应清空该 query cache。
- `lookup` 上报不依赖本 API。

## 8. 测试要求

- 缺 principal 返回 `401`。
- 无 learning profile 返回 null/empty。
- active profile 存在时返回 `is_target=true AND status!='mastered'` ids。
- mastered target 不返回。
- `is_target=false` 不返回。
- ids 按 `coarse_unit_id asc` 返回。
