下面是基于你当前最终方案整理的一版**完整设计文档**。
这版只覆盖 **`catalog` schema**，并且已经吸收了我们前面确定的所有关键结论：

- 数据库里**只存切片视频**
- 原始长视频**不入库**
- 视频已经离线处理完成，拿到的是 **HLS 路径 + transcript JSON**
- transcript JSON **带 sentence 和 span 级时间戳**
- span 已带 `coarse_id`
- 原始 transcript JSON **保留在存储桶**
- 数据库中**不存整坨 transcript JSON 作为主读模型**
- 数据库里把 transcript 拆成标准化关系表
- `catalog.videos` 是**切片内容资产主表**
- `catalog.video_transcripts` 存 **transcript 原始对象路径 + transcript 摘要**
- Recall 未来从 `catalog.video_unit_index` 读取 video-level coarse unit 索引

---

# catalog schema 最终重构设计文档

## 1. 文档目标

本文档定义当前版本内容系统中 `catalog` schema 的最终设计。
该设计服务于以下业务前提：

当前系统不再接收“待处理的原始视频上传”，而是接收**已经离线批量处理完成的切片内容资产**。每个切片已经具备：

- 最终可播放的 HLS 产物
- 对应的 transcript JSON
- transcript 中 sentence 级时间轴
- span 级时间轴
- span 对 coarse unit 的映射结果

因此，`catalog` 的职责不再是维护媒体流水线和 AI 流水线状态，而是承接**内容事实、结构化 transcript 读模型、Recall-ready 索引、批量导入审计、用户对视频的互动状态投影**。

本文档的目标是明确：

- `catalog` 的职责边界
- 最终表结构
- 字段取舍
- 入库流程
- 后处理流程
- 幂等与更新策略
- 与 Recall 的数据契约关系

---

## 2. 设计背景

旧版 `catalog.videos` 的设计是围绕“上传后处理”的流水线展开的。
它同时承载了：

- 上传记录
- 原始文件信息
- 媒体转码状态
- AI 分析状态
- 视频资产信息
- 发布信息

这种设计在“平台内自己跑媒体/AI 流水线”的模式下是合理的，但在当前新方案下已经不适用。

当前方案的现实情况是：

- 原始视频不进数据库
- 原始视频在库外已经完成切片
- 每个切片的视频 HLS 已经生成
- transcript JSON 已经生成
- span 到 coarse unit 的映射已经生成

因此数据库不应该再保留大量“处理中间态字段”，而应收缩成一个**内容资产与索引层**。

---

## 3. 设计原则

### 3.1 只存切片视频，不存原始视频实体

数据库里的每一行 `catalog.videos` 都代表一个最终可播放、可推荐、可学习的切片视频。
原始长视频不入库，不作为实体存在，不建立父表。

`parent_video_name` 和 `parent_video_slug` 仅表示来源，不承担关系实体职责。

### 3.2 数据库存事实，不存流水线状态机

数据库不再维护：

- pending_upload
- media processing
- analysis processing
- media_job_id
- analysis_job_id

这些字段属于旧架构下的平台内部流水线，不是当前内容资产的稳定事实。

### 3.3 transcript JSON 保留在对象存储中，数据库存标准读模型

原始 transcript JSON 仍然保留在存储桶中，作为权威原始输入。
数据库内部不直接保存整个 JSON blob 作为主读模型，而是拆分为：

- transcript 摘要
- sentence 级时间轴
- semantic span 明细

这样既有原始输入可追溯，也有结构化关系数据可查询。

### 3.4 以 semantic span 作为最细粒度事实

当前 JSON 中的 `tokens` 实际上不是传统 tokenizer token，而更接近带时间戳的语义跨度单元。
它们可能是：

- 单个单词
- 多词短语
- 带解释的语义片段
- 可映射 coarse unit 的知识片段

因此数据库中应将其建模为 **semantic spans**，而不是普通 token。

### 3.5 先存确定事实，再做轻聚合，不伪造高层标签

当前 JSON 明确提供的事实包括：

- sentence 文本与时间
- span 文本与时间
- span explanation
- `coarse_id`
- `baseForm`

因此当前阶段应该优先存这些确定事实，并生成直接服务 Recall 的轻聚合索引。
不应在当前版本里拍脑袋引入这些尚无可靠来源的高层字段：

- `role`
- `context_relevance`
- `teachability_score`
- `confidence_score`

以后如果有新的 enrichment pipeline，再扩展。

### 3.6 时间单位统一使用毫秒

当前 transcript JSON 中的时间字段是毫秒级整数。
因此在 catalog 内容域中，统一使用 `ms` 作为时间单位：

- `duration_ms`
- `start_ms`
- `end_ms`
- `coverage_ms`
- `source_start_ms`
- `source_end_ms`

这样最符合输入事实，也避免无意义的精度膨胀。

---

## 4. catalog 的职责边界

`catalog` 只负责以下五类内容：

第一，切片视频内容资产主记录。
第二，transcript 的标准化读模型。
第三，从 span 聚合而来的 Recall-ready 视频级 coarse unit 索引。
第四，批量导入审计。
第五，用户对视频的互动状态投影。

`catalog` 不负责：

- recommendation 审计
- learning 状态
- 系统推荐曝光状态
- 原始视频实体
- 媒体流水线状态机
- AI 流水线状态机

---

## 5. 最终表清单

最终 `catalog` schema 建议保留 8 张表：

1. `catalog.videos`
2. `catalog.video_transcripts`
3. `catalog.video_transcript_sentences`
4. `catalog.video_semantic_spans`
5. `catalog.video_unit_index`
6. `catalog.video_ingestion_jobs`
7. `catalog.video_ingestion_job_items`
8. `catalog.video_user_states`

---

## 6. 数据模型总览

### 6.1 关系结构

```text
catalog.videos
    ├── 1:1  -> catalog.video_transcripts
    ├── 1:N  -> catalog.video_transcript_sentences
    ├── 1:N  -> catalog.video_semantic_spans
    ├── 1:N  -> catalog.video_unit_index
    └── 1:N  -> catalog.video_user_states

catalog.video_ingestion_jobs
    └── 1:N  -> catalog.video_ingestion_job_items
```

### 6.2 读路径总览

未来最核心的读路径是：

- 视频详情页：`videos -> transcripts -> sentences -> semantic_spans`
- Recall：`coarse_unit_id -> video_unit_index -> videos`
- 用户视频状态：`video_user_states`
- 导入审计：`video_ingestion_jobs -> video_ingestion_job_items`

---

# 7. 表设计详解

## 7.1 `catalog.videos`

### 表职责

切片视频内容资产主表。
每一行都表示一个最终可播放、可推荐、可学习的切片视频。

### 字段定义

| 字段                       | 类型          | 可空 | 默认值              | 说明                          |
| -------------------------- | ------------- | ---- | ------------------- | ----------------------------- |
| `video_id`                 | `uuid`        | 否   | `gen_random_uuid()` | 主键                          |
| `source_clip_key`          | `text`        | 否   |                     | 外部 clip 稳定唯一键          |
| `parent_video_name`        | `text`        | 否   |                     | 原始视频名称，仅来源信息      |
| `parent_video_slug`        | `text`        | 否   |                     | 规范化来源名                  |
| `clip_seq`                 | `integer`     | 是   |                     | 在原始视频中的顺序            |
| `source_start_ms`          | `integer`     | 是   |                     | 原始视频中的起始偏移          |
| `source_end_ms`            | `integer`     | 是   |                     | 原始视频中的结束偏移          |
| `title`                    | `text`        | 否   |                     | 标题                          |
| `description`              | `text`        | 是   |                     | 描述                          |
| `language`                 | `text`        | 否   | `'en'`              | 主语言                        |
| `duration_ms`              | `integer`     | 否   |                     | 视频时长，毫秒                |
| `hls_master_playlist_path` | `text`        | 否   |                     | HLS 主清单路径                |
| `thumbnail_url`            | `text`        | 是   |                     | 缩略图                        |
| `status`                   | `text`        | 否   | `'active'`          | `active / inactive / deleted` |
| `visibility_status`        | `text`        | 否   | `'public'`          | `public / unlisted / private` |
| `publish_at`               | `timestamptz` | 是   |                     | 发布时间                      |
| `created_at`               | `timestamptz` | 否   | `now()`             | 创建时间                      |
| `updated_at`               | `timestamptz` | 否   | `now()`             | 更新时间                      |

### 必要约束

- `primary key (video_id)`
- `unique (source_clip_key)`
- `check (duration_ms > 0)`
- `check (source_end_ms is null or source_start_ms is null or source_end_ms > source_start_ms)`
- `check (status in ('active','inactive','deleted'))`
- `check (visibility_status in ('public','unlisted','private'))`

### 索引建议

- `unique(source_clip_key)`
- `(parent_video_slug, clip_seq)`
- `(status)`
- `(visibility_status, publish_at)`
- `(created_at desc)`

### 设计说明

这张表里**不再放 transcript 原始对象路径**。
原因是 transcript 原始路径更属于 transcript 原始来源和 transcript 读模型的上游元数据，应放在 `catalog.video_transcripts`。

---

## 7.2 `catalog.video_transcripts`

### 表职责

每个视频对应一行 transcript 顶层摘要。
它同时保存：

- transcript 原始对象路径
- transcript checksum
- transcript 格式版本
- transcript 摘要统计

### 字段定义

| 字段                        | 类型           | 可空 | 默认值  | 说明                                    |
| --------------------------- | -------------- | ---- | ------- | --------------------------------------- |
| `video_id`                  | `uuid`         | 否   |         | 主键，同时外键到 `catalog.videos`       |
| `transcript_object_path`    | `text`         | 否   |         | transcript JSON 对象路径                |
| `transcript_checksum`       | `text`         | 否   |         | transcript JSON 哈希                    |
| `transcript_format_version` | `integer`      | 否   | `1`     | transcript 格式版本                     |
| `full_text`                 | `text`         | 否   |         | 全部 sentence 拼接后的完整文本          |
| `sentence_count`            | `integer`      | 否   |         | 句子数                                  |
| `semantic_span_count`       | `integer`      | 否   |         | semantic span 总数                      |
| `mapped_span_count`         | `integer`      | 否   |         | coarse_unit_id 非空的 span 数           |
| `unmapped_span_count`       | `integer`      | 否   |         | coarse_unit_id 为空的 span 数           |
| `mapped_span_ratio`         | `numeric(6,5)` | 否   |         | mapped_span_count / semantic_span_count |
| `created_at`                | `timestamptz`  | 否   | `now()` | 创建时间                                |
| `updated_at`                | `timestamptz`  | 否   | `now()` | 更新时间                                |

### 必要约束

- `primary key (video_id)`
- `foreign key (video_id) references catalog.videos(video_id) on delete cascade`
- `check (sentence_count >= 0)`
- `check (semantic_span_count >= 0)`
- `check (mapped_span_count >= 0)`
- `check (unmapped_span_count >= 0)`
- `check (mapped_span_ratio >= 0 and mapped_span_ratio <= 1)`

### 设计说明

这是 transcript 的顶层摘要表，不是 JSON 原文替代品。
原始 JSON 仍保留在对象存储中，这张表只保存其**路径、checksum、格式版本**以及查询常用的 summary 信息。

---

## 7.3 `catalog.video_transcript_sentences`

### 表职责

承接 transcript JSON 中的 `sentences[]`，形成句子级时间轴读模型。

### 字段定义

| 字段             | 类型          | 可空 | 默认值  | 说明         |
| ---------------- | ------------- | ---- | ------- | ------------ |
| `video_id`       | `uuid`        | 否   |         | 所属视频     |
| `sentence_index` | `integer`     | 否   |         | 句子序号     |
| `text`           | `text`        | 否   |         | 句子原文     |
| `start_ms`       | `integer`     | 否   |         | 句子开始时间 |
| `end_ms`         | `integer`     | 否   |         | 句子结束时间 |
| `explanation`    | `text`        | 是   |         | 句子解释     |
| `created_at`     | `timestamptz` | 否   | `now()` | 创建时间     |

### 必要约束

- `primary key (video_id, sentence_index)`
- `foreign key (video_id) references catalog.videos(video_id) on delete cascade`
- `check (sentence_index >= 0)`
- `check (start_ms >= 0)`
- `check (end_ms > start_ms)`

### 索引建议

- `(video_id, start_ms)`
- `(video_id, end_ms)`

### 设计说明

句子表支撑后续：

- 句子级展示
- 句子级时间跳转
- 句子 explanation 展示
- 句子级 evidence 标注

---

## 7.4 `catalog.video_semantic_spans`

### 表职责

这是 transcript 中最细粒度的语义事实表。
它承接 JSON 中 sentence 下的 `tokens[]`，但内部命名为 **semantic spans**。

### 字段定义

| 字段              | 类型          | 可空 | 默认值  | 说明                         |
| ----------------- | ------------- | ---- | ------- | ---------------------------- |
| `video_id`        | `uuid`        | 否   |         | 所属视频                     |
| `sentence_index`  | `integer`     | 否   |         | 所属句子                     |
| `span_index`      | `integer`     | 否   |         | span 序号（来自 JSON index） |
| `text`            | `text`        | 否   |         | span 文本                    |
| `start_ms`        | `integer`     | 否   |         | span 开始时间                |
| `end_ms`          | `integer`     | 否   |         | span 结束时间                |
| `explanation`     | `text`        | 是   |         | span 解释                    |
| `coarse_unit_id`  | `bigint`      | 是   |         | 对应 coarse unit，可为空     |
| `base_form`       | `text`        | 是   |         | 基本形态                     |
| `dictionary_text` | `text`        | 是   |         | 原始 dictionary 文本，可选   |
| `created_at`      | `timestamptz` | 否   | `now()` | 创建时间                     |

### 必要约束

- `primary key (video_id, sentence_index, span_index)`
- `foreign key (video_id, sentence_index) references catalog.video_transcript_sentences(video_id, sentence_index) on delete cascade`
- `foreign key (coarse_unit_id) references semantic.coarse_unit(id) on delete restrict`
- `check (span_index >= 0)`
- `check (start_ms >= 0)`
- `check (end_ms > start_ms)`

### 应用层校验

这张表除了数据库约束，还应在导入时做额外校验：

- span 时间必须落在所属 sentence 区间内
- 同一 `(video_id, sentence_index, span_index)` 唯一
- 非空 `coarse_unit_id` 必须真实存在
- span 可以是单词，也可以是短语，不做“单词长度”假设

### 索引建议

- `(video_id, sentence_index)`
- `(video_id, start_ms)`
- `(coarse_unit_id, video_id)` where `coarse_unit_id is not null`
- `(video_id, coarse_unit_id)` where `coarse_unit_id is not null`

### 关于 `reason`

`semanticElement.reason` 不建议进入主查询表。
它保留在原始 transcript JSON 中即可。若未来需要强调试链路，再单独加 debug 表，不应污染主事实表。

---

## 7.5 `catalog.video_unit_index`

### 表职责

这是 Recall 的主入口索引表。
它由 `catalog.video_semantic_spans` 聚合生成，表示：

某个视频覆盖了哪些 coarse unit，以及覆盖强度与证据是什么。

### 字段定义

| 字段                        | 类型           | 可空 | 默认值  | 说明                            |
| --------------------------- | -------------- | ---- | ------- | ------------------------------- |
| `video_id`                  | `uuid`         | 否   |         | 所属视频                        |
| `coarse_unit_id`            | `bigint`       | 否   |         | coarse unit                     |
| `mention_count`             | `integer`      | 否   |         | 出现次数                        |
| `sentence_count`            | `integer`      | 否   |         | 涉及句子数                      |
| `first_start_ms`            | `integer`      | 否   |         | 最早出现起点                    |
| `last_end_ms`               | `integer`      | 否   |         | 最晚出现终点                    |
| `coverage_ms`               | `integer`      | 否   |         | 合并重叠区间后的总覆盖时长      |
| `coverage_ratio`            | `numeric(6,5)` | 否   |         | coverage_ms / video.duration_ms |
| `sentence_indexes`          | `integer[]`    | 否   | `'{}'`  | 涉及的句子序号                  |
| `evidence_sentence_indexes` | `integer[]`    | 否   | `'{}'`  | 代表性证据句子                  |
| `evidence_span_indexes`     | `integer[]`    | 否   | `'{}'`  | 代表性证据 span                 |
| `sample_surface_forms`      | `text[]`       | 否   | `'{}'`  | 代表性表面形式                  |
| `created_at`                | `timestamptz`  | 否   | `now()` | 创建时间                        |
| `updated_at`                | `timestamptz`  | 否   | `now()` | 更新时间                        |

### 必要约束

- `primary key (video_id, coarse_unit_id)`
- `foreign key (video_id) references catalog.videos(video_id) on delete cascade`
- `foreign key (coarse_unit_id) references semantic.coarse_unit(id) on delete cascade`
- `check (mention_count > 0)`
- `check (sentence_count > 0)`
- `check (coverage_ms > 0)`
- `check (coverage_ratio >= 0 and coverage_ratio <= 1)`
- `check (last_end_ms > first_start_ms)`

### 索引建议

- `primary key (video_id, coarse_unit_id)`
- `(coarse_unit_id, mention_count desc, coverage_ratio desc)`
- `(video_id)`

### 设计说明

当前版本**不在这里存**以下高层标签：

- `role`
- `context_relevance`
- `teachability_score`
- `confidence_score`

因为这些不是当前 transcript JSON 明确提供的事实。
当前阶段先把确定性聚合事实建稳，未来如有新 enrichment pipeline，再扩展。

---

## 7.6 `catalog.video_ingestion_jobs`

### 表职责

记录一次批量导入任务的整体状态。

### 字段定义

| 字段               | 类型          | 可空 | 默认值        | 说明                                              |
| ------------------ | ------------- | ---- | ------------- | ------------------------------------------------- |
| `ingestion_job_id` | `uuid`        | 否   |               | 主键                                              |
| `status`           | `text`        | 否   |               | `running / succeeded / partially_failed / failed` |
| `source_name`      | `text`        | 是   |               | 批次来源名称                                      |
| `total_items`      | `integer`     | 否   |               | 总 clip 数                                        |
| `succeeded_items`  | `integer`     | 否   | `0`           | 成功数                                            |
| `failed_items`     | `integer`     | 否   | `0`           | 失败数                                            |
| `context`          | `jsonb`       | 否   | `'{}'::jsonb` | 批次上下文                                        |
| `started_at`       | `timestamptz` | 否   |               | 开始时间                                          |
| `finished_at`      | `timestamptz` | 是   |               | 结束时间                                          |

### 必要约束

- `primary key (ingestion_job_id)`
- `check (status in ('running','succeeded','partially_failed','failed'))`
- `check (total_items >= 0)`
- `check (succeeded_items >= 0)`
- `check (failed_items >= 0)`

---

## 7.7 `catalog.video_ingestion_job_items`

### 表职责

记录某个 clip 在某个导入任务中的具体处理结果。

### 字段定义

| 字段               | 类型          | 可空 | 默认值 | 说明                                     |
| ------------------ | ------------- | ---- | ------ | ---------------------------------------- |
| `ingestion_job_id` | `uuid`        | 否   |        | 所属 job                                 |
| `source_clip_key`  | `text`        | 否   |        | 外部 clip key                            |
| `video_id`         | `uuid`        | 是   |        | 成功后关联的视频 ID                      |
| `status`           | `text`        | 否   |        | `running / succeeded / failed / skipped` |
| `warning_codes`    | `text[]`      | 否   | `'{}'` | 警告码                                   |
| `error_code`       | `text`        | 是   |        | 错误码                                   |
| `error_message`    | `text`        | 是   |        | 错误详情                                 |
| `started_at`       | `timestamptz` | 否   |        | 开始时间                                 |
| `finished_at`      | `timestamptz` | 是   |        | 结束时间                                 |

### 必要约束

- `primary key (ingestion_job_id, source_clip_key)`
- `foreign key (ingestion_job_id) references catalog.video_ingestion_jobs(ingestion_job_id) on delete cascade`
- `foreign key (video_id) references catalog.videos(video_id) on delete set null`
- `check (status in ('running','succeeded','failed','skipped'))`

### 设计说明

这张表负责承接导入失败、跳过、警告等运行态，不应该把这些运行态塞回 `catalog.videos` 主表。

---

## 7.8 `catalog.video_user_states`

### 表职责

用户对视频互动状态的聚合投影表。
它仍然是一个**读模型**，不是事件明细表。

### 字段定义

| 字段               | 类型           | 可空 | 默认值  | 说明                  |
| ------------------ | -------------- | ---- | ------- | --------------------- |
| `user_id`          | `uuid`         | 否   |         | 用户                  |
| `video_id`         | `uuid`         | 否   |         | 视频                  |
| `has_liked`        | `boolean`      | 否   | `false` | 是否点过赞            |
| `has_bookmarked`   | `boolean`      | 否   | `false` | 是否收藏过            |
| `has_watched`      | `boolean`      | 否   | `false` | 是否看过              |
| `liked_at`         | `timestamptz`  | 是   |         | 最近一次点赞时间      |
| `bookmarked_at`    | `timestamptz`  | 是   |         | 最近一次收藏时间      |
| `first_watched_at` | `timestamptz`  | 是   |         | 第一次观看时间        |
| `last_watched_at`  | `timestamptz`  | 是   |         | 最近一次观看时间      |
| `watch_count`      | `integer`      | 否   | `0`     | 观看次数              |
| `completed_count`  | `integer`      | 否   | `0`     | 完整看完次数          |
| `last_watch_ratio` | `numeric(6,5)` | 是   |         | 最近一次观看比例，0~1 |
| `max_watch_ratio`  | `numeric(6,5)` | 是   |         | 历史最大观看比例，0~1 |
| `updated_at`       | `timestamptz`  | 否   | `now()` | 投影更新时间          |

### 必要约束

- `primary key (user_id, video_id)`
- `foreign key (user_id) references auth.users(id) on delete cascade`
- `foreign key (video_id) references catalog.videos(video_id) on delete cascade`
- `check (watch_count >= 0)`
- `check (completed_count >= 0)`
- `check (last_watch_ratio is null or (last_watch_ratio >= 0 and last_watch_ratio <= 1))`
- `check (max_watch_ratio is null or (max_watch_ratio >= 0 and max_watch_ratio <= 1))`

### 索引建议

- `primary key (user_id, video_id)`
- `(video_id)`
- `(user_id, last_watched_at desc)`

### 设计说明

这里明确**不保留旧版 `occurred_at`**。
原因是它语义混乱，无法清晰表示 like/bookmark/watch 三类行为中到底是哪一个事件时间。

同时这张表**不承接系统推荐曝光状态**。
推荐曝光状态应继续放在 `recommendation.user_video_serving_states` 之类的 recommendation schema 表中，而不是混到 catalog 里。

---

# 8. 原始 transcript JSON 的保留策略

这是最终设计里一个必须写清楚的点。

## 8.1 原始 JSON 仍然保留

系统**不会把原始 transcript JSON 完全丢掉**。
它仍然保留在对象存储中，作为权威原始输入和审计来源。

## 8.2 数据库不存整坨 JSON 作为主读模型

数据库中不直接放一列大 JSON blob 来供业务查询。
数据库内部真正使用的是拆分后的三层标准读模型：

- `catalog.video_transcripts`
- `catalog.video_transcript_sentences`
- `catalog.video_semantic_spans`

## 8.3 对象路径应该放在哪里

最终决定如下：

- HLS 路径放在 `catalog.videos`
- transcript 原始 JSON 路径、checksum、format version 放在 `catalog.video_transcripts`

这是因为：

- HLS 属于视频播放主资产
- transcript 原始路径属于 transcript 原始来源与 transcript 读模型的上游元数据

这样职责边界最清晰。

---

# 9. 入库流程设计

## 9.1 总体策略

入库采用：

**批量 job 驱动，单 clip 单事务 replace 写入。**

含义是：

- 一次导入任务由 `video_ingestion_jobs` 记录
- 每个 clip 的处理结果由 `video_ingestion_job_items` 记录
- 每个 clip 在数据库中要么整体成功，要么整体不落地
- transcript 展开和 `video_unit_index` 聚合都在同一事务内完成

## 9.2 单 clip 导入步骤

### Step 1：读取 manifest 与 transcript JSON

输入包括：

- `source_clip_key`
- 来源信息
- HLS 路径
- transcript 路径
- transcript checksum
- transcript JSON 正文

### Step 2：校验内容资产元数据

必须校验：

- `source_clip_key` 非空
- `parent_video_name` 非空
- `title` 非空
- `duration_ms > 0`
- `hls_master_playlist_path` 非空
- `transcript_object_path` 非空
- `transcript_checksum` 非空

### Step 3：校验 transcript 结构

必须校验：

- 顶层存在 `sentences`
- sentence 的 `index / text / start / end` 合法
- `end_ms > start_ms`
- span 的 `index / text / start / end` 合法
- span 时间必须落在所属 sentence 时间区间内
- 若 `coarse_id` 非空，必须存在于 `semantic.coarse_unit`

### Step 4：归一化中间对象

在应用层生成：

- `full_text`
- sentence rows
- semantic span rows
- `video_unit_index` 聚合结果

### Step 5：开启事务并 upsert `catalog.videos`

按 `source_clip_key` 做幂等 upsert。
若该 clip 已存在，则更新其：

- 标题
- 描述
- HLS 路径
- 缩略图
- 时长
- 来源信息
- 状态/可见性字段

### Step 6：replace `catalog.video_transcripts`

按 `video_id` upsert transcript 顶层摘要与 transcript 元数据。

### Step 7：replace `catalog.video_transcript_sentences`

删除该视频旧 sentence rows，插入新 sentence rows。

### Step 8：replace `catalog.video_semantic_spans`

删除该视频旧 semantic span rows，插入新 span rows。

### Step 9：replace `catalog.video_unit_index`

删除该视频旧 coarse unit 聚合索引，写入新的聚合结果。

### Step 10：提交事务并更新 job item

成功则写：

- `status = succeeded`
- 关联 `video_id`

失败则回滚事务，写：

- `status = failed`
- `error_code`
- `error_message`

---

# 10. 后处理设计

这里的“后处理”不是另一条重型异步流水线，而是单 clip 入库过程中的**轻量派生计算**。

## 10.1 transcript 摘要生成

从 sentence 和 span 生成：

- `full_text`
- `sentence_count`
- `semantic_span_count`
- `mapped_span_count`
- `unmapped_span_count`
- `mapped_span_ratio`

写入 `catalog.video_transcripts`。

## 10.2 coarse unit 聚合生成

从 `catalog.video_semantic_spans` 中所有 `coarse_unit_id is not null` 的行，按 `(video_id, coarse_unit_id)` 聚合出：

- `mention_count`
- `sentence_count`
- `first_start_ms`
- `last_end_ms`
- `sentence_indexes`
- `sample_surface_forms`

## 10.3 覆盖时长计算

因为 span 现在带时间戳，所以可以真实计算 `coverage_ms`。
正确做法不是简单把 span 时长相加，而是：

1. 取同一 `(video_id, coarse_unit_id)` 下所有 span 区间
2. 按开始时间排序
3. 合并重叠区间
4. 对合并后的区间求总时长

然后计算：

`coverage_ratio = coverage_ms / videos.duration_ms`

## 10.4 证据位置生成

为每个 `(video_id, coarse_unit_id)` 生成轻量 evidence：

- `evidence_sentence_indexes`
- `evidence_span_indexes`

MVP 阶段不需要保存过多位置，只需保留一小组代表性证据即可。

---

# 11. 幂等与更新策略

## 11.1 幂等锚点

采用：

`source_clip_key`

作为单 clip 的业务唯一锚点。

## 11.2 跳过策略

如果发现：

- `source_clip_key` 已存在
- transcript checksum 未变化
- HLS 路径未变化
- 主要元数据未变化

则当前导入项可标记为 `skipped`。

## 11.3 replace 策略

如果发生以下任一变化：

- transcript checksum 变化
- HLS 路径变化
- 时长变化
- transcript 结构化内容变化
- 标题/描述变化

则执行完整 replace 写入：

- `video_transcripts`
- `video_transcript_sentences`
- `video_semantic_spans`
- `video_unit_index`

---

# 12. 与 Recall 的数据契约

当前这版 catalog 设计，直接决定了 Recall 的主读路径。

未来 Recall 的主入口应是：

`target coarse_unit_ids -> catalog.video_unit_index -> candidate videos`

Recall 当前可直接依赖的稳定信号包括：

- `mention_count`
- `sentence_count`
- `coverage_ms`
- `coverage_ratio`
- `sentence_indexes`
- `sample_surface_forms`
- `duration_ms`
- `parent_video_slug`

当前不应假设 catalog 已经提供这些高阶标签：

- `role`
- `context_relevance`
- `teachability_score`
- `confidence_score`

如果后续确实要做这些高层语义特征，应通过新的 enrichment pipeline 补充，而不是在当前版本里伪造。

---

# 13. 明确不再保留的旧结构

为了避免新旧架构混杂，下面这些旧结构应明确淘汰。

## 13.1 旧 `videos` 流水线字段全部删除

包括但不限于：

- `upload_user_id`
- `raw_file_reference`
- `media_status`
- `analysis_status`
- `media_job_id`
- `analysis_job_id`
- `media_emitted_at`
- `analysis_emitted_at`
- `raw_file_size`
- `raw_resolution`
- `raw_bitrate`
- `encoded_resolution`
- `encoded_bitrate`
- `difficulty`
- `summary`
- `tags`
- `raw_subtitle_url`
- `error_message`
- `version`

## 13.2 不再建立 `catalog.video_segments`

因为数据库里只存切片视频，`video_id` 本身已经是最终内容对象。
再建 segment 会让对象层级混乱。

## 13.3 不再建立 `catalog.segment_unit_mappings`

这套结构属于旧版“视频下再拆片段”的模型。
现在它由：

- `catalog.video_semantic_spans`
- `catalog.video_unit_index`

共同取代。

## 13.4 不在当前版本的 `video_unit_index` 中提前固化高层语义评分字段

暂不加入：

- `role`
- `context_relevance`
- `teachability_score`
- `confidence_score`

当前版本只保留可由输入稳定推导的确定事实。

---

# 14. 最终结论

这次 catalog 的最终重构，不是“在旧 `videos` 表上删几个字段”这么简单，而是一次**主表职责与数据链路的重建**。

新的最终数据链应该是：

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

- `catalog.video_ingestion_jobs`
- `catalog.video_ingestion_job_items`
- `catalog.video_user_states`

这套设计的核心优点是：

- 与当前真实输入完全一致
- 不再假装数据库里有媒体/AI 内部流水线
- 不再假装有原始视频实体
- 原始 transcript JSON 得到保留
- 数据库查询模型更清晰
- Recall 主读路径更直接
- 后续 Task 层也更容易做句子和 span 级交互

如果你愿意，下一步我可以直接把这份最终文档落成 **Postgres DDL 版本**，按这 8 张表完整写出建表 SQL、约束、索引和更新时间触发器。
