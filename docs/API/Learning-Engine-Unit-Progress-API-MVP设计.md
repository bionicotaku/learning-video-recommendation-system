# Learning Engine Unit Progress API MVP 设计

## 0. 文档信息

文档名称：Learning Engine Unit Progress API MVP 设计
适用阶段：MVP
目标读者：前端、后端、Learning Engine 维护者
文档目标：定义前端分页读取用户单词学习进度的最小 API 契约，包括 mastered / unmastered 两类列表、字段语义、排序规则、分页方式和数据库读取边界。

---

## 1. 背景

前端需要展示用户对学习单元的学习进度，当前需求可以拆成两类：

1. 已掌握词列表：读取所有已经 mastered 的目标词，并按 `label` 的字典序分页。
2. 未掌握词列表：读取所有尚未 mastered 的目标词，先按 `progress_percent` 从大到小分页，相同进度下再按 `label` 的字典序分页。

Learning Engine 当前已经维护 `learning.user_unit_states`。该表保存用户对每个 `semantic.coarse_unit` 的学习状态、进度百分比、掌握分和最近 progress 时间。

`semantic.coarse_unit` 保存学习单元本身的内容元数据，包括英文 label、词性、中文短标签和中文释义。前端展示学习进度时，需要把 Learning Engine 状态与 `semantic.coarse_unit` 展示字段合并返回。

---

## 2. 设计目标

MVP 目标：

- 给前端提供稳定、可分页的用户 unit progress 列表。
- API 返回前端直接展示所需字段，不要求前端再用 `coarse_unit_id` 二次补详情。
- mastered / unmastered 的分组语义由 Learning Engine 状态字段决定。
- 分页顺序稳定，不因相同排序值导致重复或漏项。
- 不引入 `display_text`、`base_form` 等数据库不存在或语义不清的展示字段。

MVP 不解决：

- 不提供按中文、释义、词性搜索。
- 不提供所有历史 exposure / lookup 过的非目标词列表。
- 不提供 suspended / inactive target 的混合列表。
- 不提供 Recommendation 维度的推荐理由、视频覆盖信息或 serving state。
- 不定义 HTTP 鉴权细节；本文只定义 Learning Engine 对外读契约。

---

## 3. 数据来源

### 3.1 主状态表

主状态来自：

```text
learning.user_unit_states
```

关键字段：

- `user_id`
- `coarse_unit_id`
- `is_target`
- `status`
- `progress_percent`
- `last_progress_at`

### 3.2 展示元数据表

展示字段来自：

```text
semantic.coarse_unit
```

关键字段：

- `id`
- `kind`
- `label`
- `pos`
- `chinese_label`
- `chinese_def`

### 3.3 Join 关系

```sql
learning.user_unit_states.coarse_unit_id = semantic.coarse_unit.id
```

MVP 查询只返回能成功 join 到 `semantic.coarse_unit` 的状态行。若状态表中存在孤儿 `coarse_unit_id`，说明数据完整性已经异常，不应由 API 临时伪造展示文本。

---

## 4. 状态分组语义

### 4.1 Mastered

已掌握列表使用状态字段判断：

```sql
s.status = 'mastered'
```

不要使用：

```sql
s.progress_percent = 100
```

原因：

- `status` 是状态机结论。
- `progress_percent` 是展示和排序用的百分比信号。
- 当前 reducer 在 mastered 时通常会把 `progress_percent` 置为 `100`，但这不应反过来成为 mastered 的权威判断。
- 迁移、手工修复或未来公式调整都可能让两者暂时不完全等价。

### 4.2 Unmastered

未掌握列表使用：

```sql
s.status in ('new', 'learning', 'reviewing')
```

不要使用：

```sql
s.status <> 'mastered'
```

原因是 `suspended` 是控制态，不应混入正常未掌握学习列表。

### 4.3 Target 范围

MVP 默认只展示当前目标学习单元：

```sql
s.is_target = true
```

`is_target = false` 表示该 user-unit state 已不在当前目标学习范围内。若未来需要“历史学过词库”，应新增独立 API 或增加显式 filter，不应让 MVP 列表默认混入 inactive target。

---

## 5. API 形态

### 5.1 前端端点

建议对前端暴露两个语义明确的端点：

```text
GET /learning/unit-progress/mastered
GET /learning/unit-progress/unmastered
```

两个端点共享同一个内部 read usecase：

```text
ListUserUnitProgress(user_id, bucket, limit, cursor)
```

其中：

```text
bucket = mastered | unmastered
```

这样可以避免 mastered / unmastered 两套实现重复，也方便后续统一增加搜索、过滤和字段。

### 5.2 请求参数

通用 query 参数：

| 参数 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `limit` | integer | 否 | `50` | 每页数量。MVP 建议范围 `1..100`。 |
| `cursor` | string | 否 | 无 | 上一页返回的 opaque cursor。第一页不传。 |

`user_id` 不建议由前端 query 传入。实际 HTTP 层应从认证上下文解析当前用户，再传给 Learning Engine usecase。

---

## 6. 返回结构

### 6.1 Response

```json
{
  "items": [
    {
      "coarse_unit_id": 101,
      "kind": "word",
      "label": "abandon",
      "pos": "verb",
      "chinese_label": "放弃；抛弃",
      "chinese_def": "表示放弃某事物、抛弃某人或中止某计划。",
      "progress_percent": 64.25,
      "last_progress_at": "2026-05-08T09:20:00Z"
    }
  ],
  "page": {
    "limit": 50,
    "has_more": true,
    "next_cursor": "opaque-token"
  }
}
```

### 6.2 Item 字段

| 字段 | 类型 | 来源 | 说明 |
| --- | --- | --- | --- |
| `coarse_unit_id` | integer | `learning.user_unit_states.coarse_unit_id` | 学习单元 ID。 |
| `kind` | string | `semantic.coarse_unit.kind` | `word` / `phrase` / `grammar` 等。 |
| `label` | string | `semantic.coarse_unit.label` | 英文主标签，作为字典序排序字段。 |
| `pos` | string \| null | `semantic.coarse_unit.pos` | 原始词性值，例如 `noun`、`verb`、`adjective`、`adverb`。后端不映射为 `n.` / `v.` / `adj.`，展示映射交给前端。 |
| `chinese_label` | string \| null | `semantic.coarse_unit.chinese_label` | 中文短标签，用于列表展示。 |
| `chinese_def` | string \| null | `semantic.coarse_unit.chinese_def` | 中文释义，用于列表或详情展示。 |
| `progress_percent` | number | `learning.user_unit_states.progress_percent` | `0..100`，保留两位小数。 |
| `last_progress_at` | string \| null | `learning.user_unit_states.last_progress_at` | 最近一次推进 progress 的事件时间。新词可能为空。 |

MVP 不返回 `display_text`。前端如需展示 fallback，可以自行按明确规则使用 `chinese_label ?? label`，但后端不伪造数据库不存在的显示字段。

---

## 7. 排序规则

### 7.1 Mastered 排序

业务规则：

```text
按 label 字典序升序
```

推荐 SQL 排序：

```sql
order by lower(cu.label) asc, cu.label asc, s.coarse_unit_id asc
```

说明：

- `lower(cu.label)` 让大小写差异不影响主要字典序。
- `cu.label` 保证大小写不同但 lower 后相同的 label 仍有稳定顺序。
- `s.coarse_unit_id` 是最终 tie breaker，保证分页稳定。

### 7.2 Unmastered 排序

业务规则：

```text
先按 progress_percent 从大到小
相同 progress_percent 时按 label 字典序升序
```

推荐 SQL 排序：

```sql
order by
  s.progress_percent desc,
  lower(cu.label) asc,
  cu.label asc,
  s.coarse_unit_id asc
```

`coarse_unit_id` 仍必须作为最终 tie breaker。

---

## 8. Cursor 分页

MVP 不建议使用 offset pagination。

原因：

- 学习状态会随着用户学习动作持续更新。
- offset 分页在数据变化时容易重复或漏项。
- mastered / unmastered 排序都可以用 keyset cursor 稳定表达。

### 8.1 Cursor 格式

对外返回 opaque token，例如 base64url 编码后的 JSON。前端不解析 cursor。

Mastered cursor 内部内容：

```json
{
  "bucket": "mastered",
  "label_key": "abandon",
  "label": "abandon",
  "coarse_unit_id": 101
}
```

Unmastered cursor 内部内容：

```json
{
  "bucket": "unmastered",
  "progress_percent": 64.25,
  "label_key": "abandon",
  "label": "abandon",
  "coarse_unit_id": 101
}
```

`label_key` 对应 `lower(label)`。

### 8.2 Mastered 下一页条件

```sql
and (
  lower(cu.label) > sqlc.arg(cursor_label_key)
  or (
    lower(cu.label) = sqlc.arg(cursor_label_key)
    and cu.label > sqlc.arg(cursor_label)
  )
  or (
    lower(cu.label) = sqlc.arg(cursor_label_key)
    and cu.label = sqlc.arg(cursor_label)
    and s.coarse_unit_id > sqlc.arg(cursor_coarse_unit_id)
  )
)
```

第一页不带 cursor 时不加该条件。

### 8.3 Unmastered 下一页条件

因为 `progress_percent` 是降序，下一页条件是小于上一页最后一项的进度；进度相同时再比较 label 和 ID：

```sql
and (
  s.progress_percent < sqlc.arg(cursor_progress_percent)
  or (
    s.progress_percent = sqlc.arg(cursor_progress_percent)
    and lower(cu.label) > sqlc.arg(cursor_label_key)
  )
  or (
    s.progress_percent = sqlc.arg(cursor_progress_percent)
    and lower(cu.label) = sqlc.arg(cursor_label_key)
    and cu.label > sqlc.arg(cursor_label)
  )
  or (
    s.progress_percent = sqlc.arg(cursor_progress_percent)
    and lower(cu.label) = sqlc.arg(cursor_label_key)
    and cu.label = sqlc.arg(cursor_label)
    and s.coarse_unit_id > sqlc.arg(cursor_coarse_unit_id)
  )
)
```

---

## 9. 查询草案

### 9.1 Mastered

以下是不带 cursor 的第一页查询草案。后续页需要追加第 8.2 节中的 cursor 条件。

```sql
select
  s.coarse_unit_id,
  cu.kind,
  cu.label,
  cu.pos,
  cu.chinese_label,
  cu.chinese_def,
  s.progress_percent,
  s.last_progress_at
from learning.user_unit_states s
join semantic.coarse_unit cu
  on cu.id = s.coarse_unit_id
where s.user_id = sqlc.arg(user_id)
  and s.is_target = true
  and s.status = 'mastered'
  and cu.status = 'active'
order by lower(cu.label) asc, cu.label asc, s.coarse_unit_id asc
limit sqlc.arg(limit_plus_one);
```

### 9.2 Unmastered

以下是不带 cursor 的第一页查询草案。后续页需要追加第 8.3 节中的 cursor 条件。

```sql
select
  s.coarse_unit_id,
  cu.kind,
  cu.label,
  cu.pos,
  cu.chinese_label,
  cu.chinese_def,
  s.progress_percent,
  s.last_progress_at
from learning.user_unit_states s
join semantic.coarse_unit cu
  on cu.id = s.coarse_unit_id
where s.user_id = sqlc.arg(user_id)
  and s.is_target = true
  and s.status in ('new', 'learning', 'reviewing')
  and cu.status = 'active'
order by
  s.progress_percent desc,
  lower(cu.label) asc,
  cu.label asc,
  s.coarse_unit_id asc
limit sqlc.arg(limit_plus_one);
```

实际实现时应读取 `limit + 1` 条：

- 如果返回行数大于 `limit`，则 `has_more = true`。
- 对前端只返回前 `limit` 条。
- `next_cursor` 使用返回给前端的最后一条 item 生成。

---

## 10. 索引建议

当前 Learning Engine 已有面向 Recommendation due 查询的索引，但它不完全覆盖这两个前端分页排序。

MVP 可以先依赖每个用户的 target unit 数量较小这一前提，完成 read API。若真实用户目标词规模达到数万，需要补充索引或读模型。

建议补充 Learning Engine 侧索引：

```sql
create index if not exists idx_learning_states_user_target_status_progress
on learning.user_unit_states (
  user_id,
  is_target,
  status,
  progress_percent desc,
  coarse_unit_id
);
```

`semantic.coarse_unit.label` 的排序索引属于 semantic owner 范围。若 mastered 列表在大用户词表下排序成本明显，应由 semantic 侧提供类似索引：

```sql
create index if not exists idx_semantic_coarse_unit_label
on semantic.coarse_unit (lower(label), label, id);
```

如果不希望 Learning Engine 热路径 join semantic 表，后续可以建立专用 read model，例如 `learning.user_unit_progress_view` 或物化读模型。但 MVP 不先引入。

---

## 11. 错误处理

### 11.1 参数错误

- `limit < 1`：返回参数错误。
- `limit > 100`：MVP 建议直接截断为 `100`，或返回参数错误；实现时二选一保持一致。
- `cursor` 解码失败：返回参数错误。
- `cursor` bucket 与当前 endpoint 不一致：返回参数错误。

### 11.2 数据缺失

如果某个 `user_unit_state` 无法 join 到 `semantic.coarse_unit`：

- MVP 查询直接不返回该行。
- 同时后端应记录 warning 或 metrics。
- 不应构造 fake label，也不应返回 `display_text`。

---

## 12. 与现有 `ListUserUnitStates` 的关系

当前 `ListUserUnitStates` 更接近内部状态读取接口，返回的是 Learning Engine 自己的 `UserUnitState` 状态模型。

本 API 是面向前端展示的 read API，特点是：

- 会 join `semantic.coarse_unit`。
- 返回字段是展示列表契约，不暴露完整状态模型。
- 有明确 mastered / unmastered bucket。
- 有稳定 keyset pagination。

因此不建议直接扩展现有 `ListUserUnitStates` 的响应结构来服务前端。更推荐新增一个专用 usecase：

```text
ListUserUnitProgress
```

或者在应用层使用同一个 repository，但保持 DTO 独立。

---

## 13. 成功标准

MVP 完成后应满足：

1. 前端可以分页读取 mastered target units，顺序稳定为 `label` 字典序。
2. 前端可以分页读取 unmastered target units，顺序稳定为 `progress_percent desc + label asc`。
3. 返回项包含 `coarse_unit_id`、`kind`、`label`、`pos`、`chinese_label`、`chinese_def`、`progress_percent`、`last_progress_at`。
4. mastered 判定只依赖 `status = 'mastered'`。
5. unmastered 判定只包含 `new`、`learning`、`reviewing`。
6. `suspended` 和 `is_target = false` 默认不出现在两个列表里。
7. 分页使用 cursor，不使用 offset。
8. 后端不返回 `display_text`，不伪造不存在的展示字段。
