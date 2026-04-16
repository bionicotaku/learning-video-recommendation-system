# Catalog 数据库改造文档（MVP / Delta 版）

状态：DEPRECATED
说明：本文档仅保留为历史参考，作为 `docs/archive/Catalog-数据库设计.md` 与《全新设计-Catalog-数据库设计.md》之间的对齐改造过程记录，不再作为当前系统实现依据。当前 Catalog 设计以《全新设计-Catalog-数据库设计.md》为准。

---

## 1. 文档目标

本文档只回答一个问题：

> **在不推翻现有 `catalog` 总体设计的前提下，为了让新的 Recommendation 主链路稳定落地，`catalog` 需要做哪些数据库改造。**

这里的“改造”包括：

- schema 级字段调整
- 索引补强
- 迁移策略
- 回填策略
- 读路径兼容策略
- 与 Recommendation 的数据契约变化

本文档**不覆盖**：

- Recommendation 自有表的完整 DDL
- Learning engine 表设计
- 上层业务 API 设计

这些不属于 `catalog` 本文档的主范围。

---

## 2. 改造背景与核心判断

当前 `catalog` 的主体结构其实是正确的：

```text
catalog.videos
    ↓
catalog.video_transcripts
    ↓
catalog.video_transcript_sentences
    ↓
catalog.video_semantic_spans
    ↓
catalog.video_unit_index
```

再辅以：

- `catalog.video_ingestion_records`
- `catalog.video_user_states`

这条链条已经很好地满足了你在 schema 文档里定义的内容域目标：`catalog` 只负责切片视频内容资产、transcript 标准读模型、Recall-ready 视频级 coarse unit 索引、入库审计、以及用户对视频的互动状态投影，不负责推荐投放状态，也不负责原始视频实体或媒体/AI 流水线状态。

因此，这次改造的结论不是“重做 catalog”，而是：

> **保留现有 `catalog` 的主骨架，只对少数关键点做修正，让它更稳地支撑新的 Recommendation 视频主链路。**

同时，需要明确一个边界判断：新的 Recommendation 主链路虽然是“供给感知的需求规划 + 多路候选生成 + 视频级排序与选择”，但它的 Recommendation own 对象——例如 video recommendation run/item、video serving state——仍然不应塞回 `catalog`，而应放在 `recommendation` schema 中。`catalog` 继续只做内容事实层与内容索引层。

---

## 3. 总体改造结论

如果把本次 Catalog 改造压缩成一句话：

> **少改 `catalog` 主体结构，只修正 `video_unit_index` 的 evidence 表达方式，补强 Recommendation 读路径需要的索引，并用确定性的迁移与回填策略，把 `catalog` 从“Recall-ready”提升为“更稳的 Recommendation 输入层”。**

更具体地说，本次改造分成四类：

### 第一类：明确不改的部分

这些结构保持不变：

- `catalog.videos`
- `catalog.video_transcripts`
- `catalog.video_transcript_sentences`
- `catalog.video_semantic_spans`
- `catalog.video_ingestion_records`
- `catalog.video_user_states`

并继续坚持：

- 不新增 `catalog.video_segments`
- 不新增 `catalog.segment_unit_mappings`
- 不在 `catalog` 中引入 Recommendation 投放状态
- 不在 `video_unit_index` 里提前固化高层语义评分字段如 `role/context_relevance/teachability/confidence`

这些判断都与当前 schema 文档一致。

### 第二类：必须调整的部分

必须调整的是：

- `catalog.video_unit_index` 的 evidence 表达方式

### 第三类：必须补强的部分

必须补强的是：

- 若现网尚未实际创建，则把设计稿中建议的若干索引提升为“必须落地”
- 为 Recommendation 新读路径补少量更实用的索引

### 第四类：配套但不属于 catalog 的新增对象

这些对象不是本次 Catalog migration 的范围，但会与这次改造强耦合：

- `recommendation.v_recommendable_video_units`
- `recommendation.v_unit_video_inventory`
- `recommendation.user_video_serving_states`
- `recommendation.video_recommendation_runs`
- `recommendation.video_recommendation_items`

它们在本文档里只做“配套说明”，不作为 Catalog DDL 主体。

---

## 4. Catalog 改造的核心原则

### 4.1 继续把 `video_semantic_spans` 当作最细粒度事实层

新的 Recommendation 需要细粒度证据，但这不意味着必须新建 `catalog.video_segments` 这类物理表。当前 schema 已经明确：

- `video_semantic_spans` 是 transcript 中最细粒度语义事实
- `video_unit_index` 是 Recall 主入口索引
- `video_segments` / `segment_unit_mappings` 属于旧版“视频下再拆片段”的模型，不再建立

因此，Catalog 继续使用：

- `video_semantic_spans` 作为最细事实层
- `video_unit_index` 作为视频级粗召回索引

Recommendation 内部说的“细粒度 evidence”，在你这套 schema 里，应理解为 **span evidence / sentence-window evidence**，而不是必须增加一张新 segment 表。

### 4.2 Catalog 只存“可回查、可复用、低语义承诺”的证据

这条原则直接决定 `video_unit_index` 的改法。

当前 Recommendation 最终一定需要“best evidence”之类的概念做：

- explanation
- jump-to
- debug
- 最终审计

但 **Catalog 不应该提前固化 Recommendation 解释层的“best”语义**。因为目前“什么叫 best”并没有被内容域稳定定义：

- 最早出现的？
- 最长覆盖的？
- 最清晰句子的？
- 最适合学习解释的？
- 对 ranker 最有贡献的？

这些都不应在 Catalog 层提前写死。

因此，Catalog 层只应存：

> **一组无歧义、可逆、可回查的代表性 evidence refs**

而不是提前存一个强语义的 `best_*` 字段。

### 4.3 迁移应以 `video_semantic_spans` 为权威来源，不信任旧 evidence 数组的语义

因为当前 `video_unit_index` 里的：

- `evidence_sentence_indexes`
- `evidence_span_indexes`

并不能无歧义表达某个具体 span。`span_index` 只在某个 `sentence_index` 内唯一，单独一组 `span_index` 没有完整定位意义。

因此回填时的权威来源必须是：

- `catalog.video_semantic_spans`

而不是直接把旧数组拼凑成新结构。

---

## 5. 核心改动一：重做 `catalog.video_unit_index` 的 evidence 表达

这次 Catalog 改造里，最关键、最值得明确的一点，就是这里。

### 5.1 现状问题

当前 `catalog.video_unit_index` 中的证据字段是：

- `evidence_sentence_indexes integer[]`
- `evidence_span_indexes integer[]`

这个表达方式有两个问题。

#### 问题一：丢失配对关系

`video_semantic_spans` 的主键是：

- `video_id`
- `sentence_index`
- `span_index`

而你现在把 `sentence_index[]` 和 `span_index[]` 分开存，意味着后续读取者并不知道：

- 第一个 sentence index 对应的是哪个 span index
- 两组数组是不是按位置一一对应
- 如果长度不等该如何解释

这在模型层面是不完整表达。

#### 问题二：过早逼近“best”语义，但又没有真正定义 best

现在这两个数组其实已经隐含“这些是代表性证据”的意思，但又没有清晰定义它们与最终 Recommendation 的 `best evidence` 之间的关系，导致中间语义很模糊。

### 5.2 改造目标

把这两个字段替换成：

- `evidence_span_refs jsonb`

并把它定义成：

> **一组轻量、无歧义的 span 引用，不承诺 best，不承诺顺序即排名，只承诺能稳定回查。**

### 5.3 推荐的新字段定义

建议在 `catalog.video_unit_index` 中新增：

```sql
evidence_span_refs jsonb not null default '[]'::jsonb
```

推荐的 JSON 结构：

```json
[
  {
    "sentence_index": 3,
    "span_index": 1
  },
  {
    "sentence_index": 4,
    "span_index": 0
  }
]
```

这是第一版最推荐的最小结构。

### 5.4 为什么不建议当前就加 `best_evidence_sentence_index` / `best_evidence_span_index`

不是不能加，而是**现在不值得加**。

因为在当前阶段：

- Catalog 的职责是事实与轻聚合索引
- Recommendation 的职责是 rank / explanation / final item construction

如果在 `catalog.video_unit_index` 里直接加 `best_*`，就相当于提前把 Recommendation 解释层的策略固化进内容索引层。这会带来两个问题：

第一，后面如果 best 的定义变了，你要回刷整张索引表。
第二，Catalog 与 Recommendation 的边界会变模糊。

因此当前更好的分层是：

- `catalog.video_unit_index`：保存 `evidence_span_refs`
- `recommendation.video_recommendation_items`：在最终 run/item 阶段保存该轮使用的 `best_evidence_*`

### 5.5 `evidence_span_refs` 的语义约束

建议写进应用层 contract：

1. 每个元素必须至少包含：
   - `sentence_index`
   - `span_index`

2. 当前阶段不要求包含：
   - `start_ms`
   - `end_ms`
   - `surface_form`
   - `relevance_score`

3. 单行建议最多保存 **1~5 个 refs**
   - 不要把所有 evidence 都塞进去
   - 它只是代表性引用集合，不是完整 span 明细

4. JSON 数组中的顺序是**稳定顺序**，不是“推荐意义上的排名顺序”

5. 所有 refs 必须能在 `catalog.video_semantic_spans` 中找到唯一行

### 5.6 `sentence_indexes` 是否保留

保留。
`sentence_indexes` 仍然有价值，因为它表示“这个 unit 在视频中涉及了哪些句子”，是视频级索引的一部分。它的语义与 `evidence_span_refs` 不冲突：

- `sentence_indexes`：覆盖范围
- `evidence_span_refs`：小样本代表性引用

### 5.7 推荐的 DDL 变更方式

第一阶段只做**增量迁移**，先不删旧字段。

```sql
alter table catalog.video_unit_index
  add column if not exists evidence_span_refs jsonb not null default '[]'::jsonb;
```

暂时保留：

- `evidence_sentence_indexes`
- `evidence_span_indexes`

等 reader 全部切到新字段，再做删除。

---

## 6. 核心改动二：索引从“建议”升级为“必须”

当前 schema 文档里已经给过一些索引建议，但在新的 Recommendation 主链路下，部分索引应该从“建议”提升为“必需”，否则读路径会很别扭。

### 6.1 `catalog.video_unit_index`：新增 `(coarse_unit_id, video_id)` 索引

当前 `video_unit_index` 已建议：

- 主键 `(video_id, coarse_unit_id)`
- `(coarse_unit_id, mention_count desc, coverage_ratio desc)`
- `(video_id)`

新的 Recommendation 主链路里，Exact / Bundle / SoftFuture 典型的第一跳查询是：

> 给一组 `coarse_unit_id`，先找命中这些 unit 的候选 `video_id`

这时一个更实用的索引是：

```sql
create index if not exists idx_video_unit_index_unit_video
on catalog.video_unit_index (coarse_unit_id, video_id);
```

### 为什么这个索引重要

因为 `(coarse_unit_id, mention_count desc, coverage_ratio desc)` 更偏“按单 unit 排序”，而 Bundle/交集类查询第一步更关心：

- 某个 unit 对应哪些 videos
- 多个 unit 最后聚合到哪些 videos

用 `(coarse_unit_id, video_id)` 更自然。

---

### 6.2 `catalog.video_semantic_spans`：新增 unit-driven 证据回查索引

当前 schema 文档已经建议：

- `(coarse_unit_id, video_id)` where `coarse_unit_id is not null`
- `(video_id, coarse_unit_id)` where `coarse_unit_id is not null`

这些建议我认为都应该实际创建。

此外，为了让 Recommendation 在确定 `video_id + coarse_unit_id` 后更容易回查“最佳 evidence 候选”，建议再补一个：

```sql
create index if not exists idx_video_semantic_spans_unit_video_start
on catalog.video_semantic_spans (coarse_unit_id, video_id, start_ms)
where coarse_unit_id is not null;
```

### 为什么这个索引值得补

因为新主链路下，Recommendation 很可能会这样用：

1. 用 `video_unit_index` 找候选 video
2. 选中某个 video 后，再从 `video_semantic_spans` 里找该 `video_id + coarse_unit_id` 的更细粒度 evidence

这个索引正好支持这种“按 unit + video 回查并按时间排序”的模式。

---

### 6.3 `catalog.videos`：为“可推荐视频池”补一个 partial index

新的 Recommendation 里，Fallback 或 quality gate 常见的过滤条件是：

- `status = active`
- `visibility_status = public`
- `publish_at <= now()`

当前 schema 文档中已有：

- `(status)`
- `(visibility_status, publish_at)`
- `(created_at desc)`

但若 Recommendation 以后频繁要从 `catalog.videos` 里拉“可推荐视频池”，我建议再加一个 partial index：

```sql
create index if not exists idx_videos_recommendable
on catalog.videos (publish_at desc, duration_ms)
where status = 'active'
  and visibility_status = 'public';
```

### 为什么值得加

因为这类过滤非常稳定，属于 Recommendation/Fallback 的热路径候选集合，不值得每次都走宽泛索引。

---

### 6.4 `catalog.video_user_states`：不改字段，但要确保读索引实际存在

如果 Recommendation 会把：

- 最近是否看过
- 最近观看时间
- 历史最大观看比例

作为轻量 penalty 读入，那么 `video_user_states` 上的：

- `(user_id, last_watched_at desc)`

不应只是文档建议，而应确保真实创建。

不过这里不建议再额外加字段，因为当前表定义已经足够。

---

## 7. Catalog 中明确“不改”的对象

为了防止 catalog 再膨胀，下面这些要明确写成“不改项”。

### 7.1 不新增 `catalog.video_segments`

当前 schema 文档已经明确：数据库中只存切片视频，`video_id` 本身已经是最终内容对象，再建 segment 会把对象层级重新搞混。

我同意这个判断。
Recommendation 内部需要细粒度 evidence，不代表 catalog 必须增加一张 segment 物理表。

### 7.2 不新增 `catalog.segment_unit_mappings`

因为这层职责已经被：

- `catalog.video_semantic_spans`
- `catalog.video_unit_index`

共同覆盖。

### 7.3 不在 `video_unit_index` 中增加高层语义评分字段

当前不增加：

- `role`
- `context_relevance`
- `teachability_score`
- `confidence_score`

原因不变：这些都不是当前 transcript JSON 中的稳定事实。当前 Recommendation 需要的教育适配度、context quality、future value 等特征，应在 Recommendation 层组合已有确定性事实，而不是提前写进 Catalog。

### 7.4 不把 Recommendation 投放状态塞回 `catalog.video_user_states`

当前 schema 文档已经明确：

- `catalog.video_user_states` 是用户与视频互动状态的聚合投影
- 系统推荐曝光状态应留在 `recommendation.user_video_serving_states`

这个边界必须继续保持。

---

## 8. 数据迁移设计

这一节是工程上最重要的部分之一。

### 8.1 总原则

本次迁移应采用：

> **加字段 → 回填 → 双读兼容 → reader 切换 → 清理旧字段**

而不是一次性直接 drop old columns。

### 8.2 迁移步骤

#### Step 1：增加新列与新索引

执行 additive migration：

- `add column evidence_span_refs jsonb`
- `create index idx_video_unit_index_unit_video`
- `create index idx_video_semantic_spans_unit_video_start`
- `create index idx_videos_recommendable`

#### Step 2：更新写入逻辑

更新 Catalog 入库/replace 逻辑，使新写入的视频：

- 继续写现有字段
- 同时写 `evidence_span_refs`

此阶段可以选择：

- 暂时双写旧 evidence arrays 与新 jsonb
- 或直接只写新 jsonb，旧列仅保留不再更新

推荐更稳妥的做法是：

> 先双写一个版本

#### Step 3：回填历史数据

历史数据不要尝试“zip 旧数组还原新 jsonb”。
推荐以 `catalog.video_semantic_spans` 为唯一权威来源重新计算。

#### Step 4：切换 reader

所有依赖 evidence 的新读路径改用：

- `evidence_span_refs`

而不是：

- `evidence_sentence_indexes`
- `evidence_span_indexes`

#### Step 5：观测与比对

至少跑一段时间，确认：

- 新旧写入无偏差
- Recommendation explanation/jump-to 读路径正常
- 查询性能可接受

#### Step 6：删除旧列

确认所有读路径都切换后，再 drop：

- `evidence_sentence_indexes`
- `evidence_span_indexes`

---

## 9. 历史数据回填设计

### 9.1 权威来源

历史回填必须以：

- `catalog.video_semantic_spans`

为准。

### 9.2 为什么不建议直接从旧数组转换

因为旧数组的表达不完整：

- 它们不保证 position 对齐语义
- 也不保证一定对应某个真实 `(sentence_index, span_index)` 对

而 `video_semantic_spans` 才是真实最细事实层。

### 9.3 回填原则

当前阶段不定义 best，所以回填目标不是“找最优证据”，而是：

> **为每个 `(video_id, coarse_unit_id)` 生成一组稳定、可回查、轻量的代表性 refs。**

### 9.4 推荐的回填选择规则

我建议用一个**稳定且低语义承诺**的规则：

1. 只看 `coarse_unit_id is not null` 的 spans
2. 对同一个 `(video_id, coarse_unit_id, sentence_index)`，只保留该 sentence 中最早的一个 span
3. 再按 `sentence_index ASC, span_index ASC` 排序
4. 取前 `K` 个，推荐 `K = 5`
5. 组装成 `jsonb_agg`

这样做有几个好处：

- 不引入 best 语义
- 稳定、可重复生成
- 尽量让 evidence refs 覆盖多个 sentence，而不是全落在一个句子里
- 仍然保持 lightweight

### 9.5 推荐回填 SQL 草案

```sql
with ranked as (
  select
    video_id,
    coarse_unit_id,
    sentence_index,
    span_index,
    row_number() over (
      partition by video_id, coarse_unit_id, sentence_index
      order by start_ms asc, span_index asc
    ) as rn_in_sentence
  from catalog.video_semantic_spans
  where coarse_unit_id is not null
),
picked as (
  select
    video_id,
    coarse_unit_id,
    sentence_index,
    span_index,
    row_number() over (
      partition by video_id, coarse_unit_id
      order by sentence_index asc, span_index asc
    ) as rn
  from ranked
  where rn_in_sentence = 1
),
refs as (
  select
    video_id,
    coarse_unit_id,
    jsonb_agg(
      jsonb_build_object(
        'sentence_index', sentence_index,
        'span_index', span_index
      )
      order by sentence_index asc, span_index asc
    ) as evidence_span_refs
  from picked
  where rn <= 5
  group by video_id, coarse_unit_id
)
update catalog.video_unit_index vui
set evidence_span_refs = refs.evidence_span_refs
from refs
where vui.video_id = refs.video_id
  and vui.coarse_unit_id = refs.coarse_unit_id;
```

### 9.6 为什么这个回填规则可接受

因为在当前阶段：

- `evidence_span_refs` 只是 Recall-ready 索引的代表性引用集合
- Recommendation 后续仍然可以按自己的规则从 refs 或 spans 里动态挑 `best_evidence_ref`
- Catalog 不需要承诺“这 5 个里面第一个就是最优”

---

## 10. DDL 变更草案

这里给出一版更完整的 DDL 草案。

### 10.1 增量迁移阶段

```sql
alter table catalog.video_unit_index
  add column if not exists evidence_span_refs jsonb not null default '[]'::jsonb;

create index if not exists idx_video_unit_index_unit_video
on catalog.video_unit_index (coarse_unit_id, video_id);

create index if not exists idx_video_semantic_spans_unit_video_start
on catalog.video_semantic_spans (coarse_unit_id, video_id, start_ms)
where coarse_unit_id is not null;

create index if not exists idx_videos_recommendable
on catalog.videos (publish_at desc, duration_ms)
where status = 'active'
  and visibility_status = 'public';
```

### 10.2 清理阶段（reader 全切换后）

```sql
alter table catalog.video_unit_index
  drop column if exists evidence_sentence_indexes,
  drop column if exists evidence_span_indexes;
```

---

## 11. 推荐读路径如何变化

这部分很重要，因为它说明 Catalog 改造后，Recommendation 应该怎么用。

### 11.1 Exact / Bundle / SoftFuture 的第一跳

Recommendation 仍然先从：

- `catalog.video_unit_index`

做粗召回，按 `coarse_unit_id` 找候选视频。

### 11.2 需要 explanation / jump-to / 精细证据时

再通过：

- `evidence_span_refs`
- 回查 `catalog.video_semantic_spans`
- 必要时 join `catalog.video_transcript_sentences`

获取更细粒度定位信息。

### 11.3 Catalog 不负责返回“最终 best evidence”

如果 Recommendation 在最终 `video_recommendation_items` 中需要：

- `best_evidence_start_ms`
- `best_evidence_end_ms`
- `best_segment_like_range`

则应在 Recommendation 聚合阶段基于：

- `evidence_span_refs`
- 或 `video_semantic_spans`

动态选出，而不是要求 Catalog 提前固化。

---

## 12. 验收与校验清单

这次改造上线前，至少应做以下检查。

### 12.1 数据正确性检查

1. `video_unit_index.evidence_span_refs` 中每个 ref 都能在 `video_semantic_spans` 中找到唯一记录
2. `evidence_span_refs` 长度不超过约定上限
3. refs 中的 `sentence_index` 都包含在 `sentence_indexes` 中
4. 引入 `evidence_span_refs` 后，`mention_count/sentence_count/coverage_ms/coverage_ratio` 不发生变化
5. 对于无 mapped spans 的行，`evidence_span_refs` 应为空数组而不是 null

### 12.2 读路径兼容检查

1. 旧 reader 仍然可运行（双写/兼容阶段）
2. 新 reader 能只依赖 `evidence_span_refs` 正常构造 explanation/jump-to
3. Recommendation 不再依赖旧 evidence 数组字段

### 12.3 性能检查

1. `coarse_unit_id -> video candidates` 查询是否命中新索引
2. `video_id + coarse_unit_id -> spans` 回查是否命中 `idx_video_semantic_spans_unit_video_start`
3. Fallback 候选池查询是否命中 `idx_videos_recommendable`

---

## 13. 本次 Catalog 改造之外，但必须同步规划的对象

虽然本文档只写 Catalog 改造，但为了防止误解，这里明确一下配套对象。

### 13.1 Recommendation 侧应新增的对象

不在 Catalog migration 内，但需要配套规划：

- `recommendation.v_recommendable_video_units`
- `recommendation.v_unit_video_inventory`
- `recommendation.user_video_serving_states`
- `recommendation.video_recommendation_runs`
- `recommendation.video_recommendation_items`

其中 `recommendation.v_recommendable_video_units` 的 owner、字段 contract、过滤规则与刷新策略，以《全新设计-推荐模块设计.md》中的权威定义为准；本文仅保留其作为配套对象的说明，不重复定义字段清单。

### 13.2 为什么这些不放在 catalog

因为当前边界已经明确：

- `catalog` 负责内容事实、内容索引、内容互动投影
- Recommendation 自己的 serving state 与 recommendation audit 必须归 `recommendation` schema

否则 Catalog 会重新膨胀成“内容 + 推荐混合域”，这与当前整体设计方向是冲突的。

---

## 14. 实施顺序建议

如果从工程上排顺序，我建议这样推进。

### Phase 1：Catalog 先做增量改造

- 加 `evidence_span_refs`
- 加新索引
- 改入库写法支持新字段

### Phase 2：历史回填

- 基于 `video_semantic_spans` 回填
- 做一致性校验

### Phase 3：Recommendation 新读路径切换

- 先切 explanation/debug 读路径
- 再切最终 Recommendation item 构建读路径

### Phase 4：清理旧 evidence 字段

- 确认无 reader 依赖后 drop old columns

---

## 15. 最终结论

这次 Catalog 数据库改造的核心不是“重做内容库”，而是：

> **在保持现有 `videos -> transcripts -> sentences -> semantic_spans -> video_unit_index` 主骨架不变的前提下，修正 `video_unit_index` 的 evidence 表达方式，使其从“语义不完整的两个数组”升级为“无歧义、可回查、低语义承诺的 `evidence_span_refs jsonb`”，并补齐 Recommendation 新读路径真正需要的少量索引。**

因此最终判断可以压成三句：

第一，`catalog` 主体结构是对的，不需要推翻。
第二，最关键的 schema 改动是把 `video_unit_index` 的 evidence 表达改对。
第三，Recommendation own 的 serving/audit 仍然不进入 `catalog`，Catalog 只负责把内容事实和 Recall-ready 索引做稳。

下一步最自然的是把这份文档继续落成一版 **Catalog migration SQL 草案**，直接按 migration 文件顺序写出来。
