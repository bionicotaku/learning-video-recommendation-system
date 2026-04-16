# Catalog 数据库最终设计文档

## 0. 文档信息

文档名称：Catalog 数据库最终设计文档
适用阶段：MVP
目标读者：后端、数据、测试、Learning engine 对接方、Recommendation 对接方
文档目标：定义当前系统中 `catalog` schema 的最终职责边界、数据模型、入库流程、后处理逻辑、幂等策略、读路径契约与实施约束，使读者在**不了解任何历史设计文档**的前提下，也能完整理解并实施当前版本的 Catalog。当前系统的业务前提是：数据库不再接收待处理的原始视频上传，而是接收**已经离线处理完成的切片内容资产**；每个切片已经具备最终可播放的 HLS 产物、transcript JSON、sentence 级时间轴、span 级时间轴，以及 span 到 `semantic.coarse_unit` 的映射结果。因此，Catalog 的职责是承接**内容事实、结构化 transcript 读模型、Recall-ready 索引、入库审计、用户对视频的互动状态投影**，而不是承接媒体/AI 流水线状态机或 Recommendation 的投放状态。

本文档是当前 Catalog 的唯一权威设计文档。`docs/Catalog-数据库设计.md` 与 `docs/Catalog数据库改造.md` 仅保留为历史参考，不再作为当前实现依据。

## 1. 设计目标

Catalog 的设计目标有五个。第一，数据库中的视频对象必须直接对应最终可播放、可推荐、可学习的切片视频，而不是原始长视频的处理中间态。第二，数据库要保留 transcript 的权威输入来源，但业务读路径必须建立在结构化读模型之上，而不是每次回读整坨 JSON。第三，Catalog 要为 Recommendation 提供稳定的 Recall-ready 索引，但只存**确定性事实与轻聚合结果**，不提前固化 Recommendation 层的高语义判断。第四，入库链路必须具备幂等、可替换、可审计能力。第五，Catalog 必须保持边界清晰：它是内容事实域与内容索引域，不是推荐域，也不是学习状态域。

## 2. 设计背景与边界

旧式 `catalog.videos` 往往围绕“上传后处理流水线”组织，表内既保存原始文件信息，也保存媒体转码状态、AI 分析状态、产出状态和发布状态。这种模式适合“平台内自己接收原始视频并驱动流水线”的系统，但与当前场景不匹配。当前现实情况是：原始视频不进数据库，长视频在库外已完成切片，每个切片的视频 HLS 与 transcript JSON 也已经生成。因此，数据库应从“流水线状态机”收缩为“内容资产与索引层”。

Catalog 只负责以下五类内容：切片视频内容资产主记录、transcript 的标准化读模型、从 semantic span 聚合而来的 Recall-ready 视频级 coarse unit 索引、单 clip 入库审计、用户对视频互动状态的聚合投影。Catalog 不负责 Learning state、Recommendation serving state、Recommendation audit、原始视频实体、媒体 / AI 流水线状态机，也不提前维护高层语义评分字段。Recommendation 自己的 serving state 与 recommendation audit 必须留在 `recommendation` schema，Learning engine 自己的学习事件与学习状态必须留在 `learning` schema。

## 3. 核心设计原则

第一，数据库中只存切片视频，不存原始视频实体。`catalog.videos` 中的一行就代表一个最终可播放、可推荐、可学习的切片视频。原始长视频不作为数据库实体存在，不建立父表。`parent_video_name` 与 `parent_video_slug` 仅表示来源信息，不承载关系实体语义。

第二，数据库存事实，不存媒体 / AI 流水线状态机。`catalog` 中不再维护 `pending_upload`、媒体处理状态、分析处理状态、job_id 等旧架构字段。这些都不是当前内容资产的稳定事实。

第三，transcript 原始 JSON 继续保留在对象存储中，数据库只存其标准读模型与必要的上游元数据。也就是说，数据库不会把整坨 JSON 作为主读模型，而会拆成三层：`catalog.video_transcripts`、`catalog.video_transcript_sentences`、`catalog.video_semantic_spans`。同时，HLS 路径属于视频播放主资产，应放在 `catalog.videos`；transcript 原始 JSON 的对象路径、checksum、格式版本属于 transcript 上游元数据，应放在 `catalog.video_transcripts`。

第四，最细粒度事实层是 `semantic span`，不是传统 token。当前 transcript JSON 中的最细单元更接近带时间戳的语义跨度，而非传统 tokenizer token。它可能是单个单词、短语、带解释的语义片段，或可映射到 `semantic.coarse_unit` 的知识片段。因此，数据库应以 `catalog.video_semantic_spans` 作为最细事实层。

第五，先存确定性事实，再做轻量聚合，不伪造高层标签。当前 Catalog 可以稳定获得的事实包括：sentence 文本与时间、span 文本与时间、span explanation、`coarse_unit_id`、`base_form`。因此当前阶段应优先存这些事实，并生成直接服务 Recall 的轻聚合索引。当前不应在 Catalog 中提前固化 `role`、`context_relevance`、`teachability_score`、`confidence_score` 等高层语义字段；如果后续需要，应通过新的 enrichment pipeline 补充。

第六，内容域时间单位统一使用毫秒。当前 transcript JSON 中的时间字段是毫秒级整数，因此 Catalog 内容域统一使用 `duration_ms`、`start_ms`、`end_ms`、`coverage_ms`、`source_start_ms`、`source_end_ms`。这既符合输入事实，也避免无意义的精度膨胀。

第七，`video_unit_index` 中的证据表达必须是**可回查、低语义承诺**的引用集合，而不是提前固化 Recommendation 解释层中的“best evidence”。当前最终方案采用 `evidence_span_refs jsonb`，每个元素至少包含 `sentence_index` 与 `span_index`，用于无歧义回查 `catalog.video_semantic_spans`。Catalog 不承诺其中任何一个 ref 是“全局 best”。

## 4. 最终表清单与关系总览

当前最终设计下，`catalog` schema 保留 7 张业务表：

1. `catalog.videos`
2. `catalog.video_transcripts`
3. `catalog.video_transcript_sentences`
4. `catalog.video_semantic_spans`
5. `catalog.video_unit_index`
6. `catalog.video_ingestion_records`
7. `catalog.video_user_states`

表间关系如下：

```text
catalog.videos
    ├── 1:1  -> catalog.video_transcripts
    ├── 1:N  -> catalog.video_transcript_sentences
    ├── 1:N  -> catalog.video_semantic_spans
    ├── 1:N  -> catalog.video_unit_index
    └── 1:N  -> catalog.video_user_states
```

未来最重要的读路径有三条：视频详情页走 `videos -> transcripts -> sentences -> semantic_spans`；Recommendation / Recall 走 `coarse_unit_id -> video_unit_index -> candidate videos -> spans / sentences`；用户视频消费状态读取走 `video_user_states`。

## 5. `catalog.videos`

`catalog.videos` 是切片视频内容资产主表。每一行都表示一个最终可播放、可推荐、可学习的切片视频。原始长视频不入库，不建立父表。该表的核心职责是承接视频内容资产本身，而不是 transcript 上游元数据，也不是入库审计。

建议字段如下：

- `video_id uuid not null default gen_random_uuid()`：主键
- `source_clip_key text not null`：外部 clip 稳定唯一键
- `parent_video_name text not null`：原始来源视频名称，仅来源信息
- `parent_video_slug text not null`：规范化来源名
- `clip_seq integer null`：切片在原始视频中的顺序
- `source_start_ms integer null`：在原始视频中的起始偏移
- `source_end_ms integer null`：在原始视频中的结束偏移
- `title text not null`：标题
- `description text null`：描述
- `clip_reason text null`：切片原因说明
- `language text not null default 'en'`：主语言
- `duration_ms integer not null`：视频时长，毫秒
- `hls_master_playlist_path text not null`：HLS 主清单路径
- `thumbnail_url text null`：缩略图
- `status text not null default 'active'`：`active / inactive / deleted`
- `visibility_status text not null default 'public'`：`public / unlisted / private`
- `publish_at timestamptz null`：发布时间
- `created_at timestamptz not null default now()`
- `updated_at timestamptz not null default now()`

必要约束应包括：`primary key (video_id)`、`unique (source_clip_key)`、`check (duration_ms > 0)`、`check (source_end_ms is null or source_start_ms is null or source_end_ms > source_start_ms)`、`check (status in ('active','inactive','deleted'))`、`check (visibility_status in ('public','unlisted','private'))`。推荐索引包括：`unique(source_clip_key)`、`(parent_video_slug, clip_seq)`、`(status)`、`(visibility_status, publish_at)`、`(created_at desc)`，以及为 Recommendation/Fallback 候选池提供的 partial index：`idx_videos_recommendable on catalog.videos (publish_at desc, duration_ms) where status = 'active' and visibility_status = 'public'`。`catalog.videos` 中不保存 transcript 原始对象路径，因为那属于 transcript 上游元数据，应放在 `catalog.video_transcripts`。

## 6. `catalog.video_transcripts`

`catalog.video_transcripts` 是每个视频的一行 transcript 顶层摘要表。它同时保存 transcript 原始对象路径、checksum、格式版本，以及由 sentence/span 派生出的 summary 统计。原始 transcript JSON 仍保留在对象存储中，这张表只是数据库侧的摘要与元数据入口。

建议字段如下：

- `video_id uuid not null`：主键，同时外键到 `catalog.videos(video_id)`
- `transcript_object_path text not null`：transcript JSON 对象路径
- `transcript_checksum text not null`：transcript JSON 哈希
- `transcript_format_version integer not null default 1`
- `full_text text not null`：所有 sentence 拼接后的完整文本
- `sentence_count integer not null`
- `semantic_span_count integer not null`
- `mapped_span_count integer not null`
- `unmapped_span_count integer not null`
- `mapped_span_ratio numeric(6,5) not null`
- `created_at timestamptz not null default now()`
- `updated_at timestamptz not null default now()`

必要约束应包括：`primary key (video_id)`、`foreign key (video_id) references catalog.videos(video_id) on delete cascade`、`check (sentence_count >= 0)`、`check (semantic_span_count >= 0)`、`check (mapped_span_count >= 0)`、`check (unmapped_span_count >= 0)`、`check (mapped_span_ratio between 0 and 1)`。这里的 `full_text`、`sentence_count`、`semantic_span_count`、`mapped_span_count`、`unmapped_span_count`、`mapped_span_ratio` 都属于入库时的轻量后处理结果。

## 7. `catalog.video_transcript_sentences`

`catalog.video_transcript_sentences` 承接 transcript JSON 中的 `sentences[]`，形成句子级时间轴读模型。它主要服务于视频详情展示、按句跳转、句子 explanation 展示，以及 Recommendation 在 `evidence_span_refs` 解析后组装 sentence-window evidence。

建议字段如下：

- `video_id uuid not null`
- `sentence_index integer not null`
- `text text not null`
- `start_ms integer not null`
- `end_ms integer not null`
- `explanation text null`
- `created_at timestamptz not null default now()`

必要约束应包括：`primary key (video_id, sentence_index)`、`foreign key (video_id) references catalog.videos(video_id) on delete cascade`、`check (sentence_index >= 0)`、`check (start_ms >= 0)`、`check (end_ms > start_ms)`。推荐索引包括：`(video_id, start_ms)`、`(video_id, end_ms)`。由于 `video_semantic_spans` 要引用句子主键，因此 `video_transcript_sentences` 也是最细事实层的父表之一。

## 8. `catalog.video_semantic_spans`

`catalog.video_semantic_spans` 是 transcript 中最细粒度的语义事实表，承接 JSON 中 sentence 下的 `tokens[]`，但在数据库内部命名为 **semantic spans**。它不是普通 tokenizer token 表，而是具备时间、文本、解释和 coarse unit 映射能力的语义跨度表。它是 Recommendation 细粒度 evidence 的最终权威来源。

建议字段如下：

- `video_id uuid not null`
- `sentence_index integer not null`
- `span_index integer not null`
- `text text not null`
- `start_ms integer not null`
- `end_ms integer not null`
- `explanation text null`
- `coarse_unit_id bigint null`
- `base_form text null`
- `dictionary_text text null`
- `created_at timestamptz not null default now()`

必要约束应包括：`primary key (video_id, sentence_index, span_index)`、`foreign key (video_id, sentence_index) references catalog.video_transcript_sentences(video_id, sentence_index) on delete cascade`、`foreign key (coarse_unit_id) references semantic.coarse_unit(id) on delete restrict`、`check (span_index >= 0)`、`check (start_ms >= 0)`、`check (end_ms > start_ms)`。应用层还应确保：span 时间落在所属 sentence 区间内；同一 `(video_id, sentence_index, span_index)` 唯一；非空 `coarse_unit_id` 必须真实存在。推荐索引包括：`(video_id, sentence_index)`、`(video_id, start_ms)`、`(coarse_unit_id, video_id) where coarse_unit_id is not null`、`(video_id, coarse_unit_id) where coarse_unit_id is not null`，以及 Recommendation 证据回查需要的 `idx_video_semantic_spans_unit_video_start on (coarse_unit_id, video_id, start_ms) where coarse_unit_id is not null`。

`semanticElement.reason` 不建议进入主查询表。它可以保留在对象存储中的 transcript 原始 JSON 里，若未来需要更强调试链路，可另建 debug 对象，不应污染当前主事实表。

## 9. `catalog.video_unit_index`

`catalog.video_unit_index` 是 Catalog 中最重要的 Recall-ready 索引表。它由 `catalog.video_semantic_spans` 聚合生成，表示“某个视频覆盖了哪些 coarse unit，以及覆盖强度与轻量证据引用是什么”。Recall 当前的主入口就是：`target coarse_unit_ids -> catalog.video_unit_index -> candidate videos`。Recommendation 需要 explanation / jump-to / 精细证据时，再基于 `evidence_span_refs` 回查 `catalog.video_semantic_spans` 与 `catalog.video_transcript_sentences`。

建议字段如下：

- `video_id uuid not null`
- `coarse_unit_id bigint not null`
- `mention_count integer not null`
- `sentence_count integer not null`
- `first_start_ms integer not null`
- `last_end_ms integer not null`
- `coverage_ms integer not null`
- `coverage_ratio numeric(6,5) not null`
- `sentence_indexes integer[] not null default '{}'`
- `evidence_span_refs jsonb not null default '[]'::jsonb`
- `sample_surface_forms text[] not null default '{}'`
- `created_at timestamptz not null default now()`
- `updated_at timestamptz not null default now()`

这里特别强调：最终设计中，`evidence_sentence_indexes` 与 `evidence_span_indexes` 这两个旧字段不再是逻辑 schema 的一部分。它们的问题在于丢失了 `(sentence_index, span_index)` 的配对关系，无法无歧义定位真实 span，也过早逼近了“best evidence”语义。最终设计用 `evidence_span_refs jsonb` 替代，定义为一组轻量、无歧义、可回查的 span 引用，不承诺 best，不承诺顺序即排名，只承诺能稳定回查。每个元素至少包含 `sentence_index` 与 `span_index`；当前阶段不要求额外保存 `start_ms/end_ms`、`surface_form` 或 `relevance_score`；每行建议最多保存 1~5 个 refs；JSON 数组的顺序是稳定顺序，而不是推荐意义上的排序；所有 refs 必须能在 `catalog.video_semantic_spans` 中找到唯一行。

必要约束应包括：`primary key (video_id, coarse_unit_id)`、`foreign key (video_id) references catalog.videos(video_id) on delete cascade`、`foreign key (coarse_unit_id) references semantic.coarse_unit(id) on delete cascade`、`check (mention_count > 0)`、`check (sentence_count > 0)`、`check (coverage_ms > 0)`、`check (coverage_ratio between 0 and 1)`、`check (last_end_ms > first_start_ms)`。推荐索引包括：`primary key (video_id, coarse_unit_id)`、`(coarse_unit_id, mention_count desc, coverage_ratio desc)`、`(video_id)`，以及为多 unit 候选聚合与 Bundle 路径补强的 `idx_video_unit_index_unit_video on (coarse_unit_id, video_id)`。

在语义上，`video_unit_index` 只保留确定性聚合事实：`mention_count`、`sentence_count`、`coverage_ms`、`coverage_ratio`、`sentence_indexes`、`sample_surface_forms` 与 `evidence_span_refs`。当前不在这里存 `role`、`context_relevance`、`teachability_score`、`confidence_score`。这些都不是当前输入中稳定存在的事实。如果未来需要，应通过 enrichment pipeline 另行补充。

## 10. `catalog.video_ingestion_records`

`catalog.video_ingestion_records` 是单视频级的入库审计表，用于记录某个 clip 的一次入库执行结果。它是单 clip 审计对象，不是批次级 job 表。系统当前不维护批次级 ingestion job；如果未来确实需要，再单独扩展。

建议字段如下：

- `ingestion_record_id uuid not null`
- `source_clip_key text not null`
- `video_id uuid null`
- `source_name text null`
- `status text not null`：`running / succeeded / failed / skipped`
- `warning_codes text[] not null default '{}'`
- `error_code text null`
- `error_message text null`
- `context jsonb not null default '{}'::jsonb`
- `started_at timestamptz not null`
- `finished_at timestamptz null`
- `created_at timestamptz not null default now()`

必要约束应包括：`primary key (ingestion_record_id)`、`foreign key (video_id) references catalog.videos(video_id) on delete set null`、`check (status in ('running','succeeded','failed','skipped'))`。推荐索引包括：`(source_clip_key, started_at desc)`、`(video_id)`、`(status, started_at desc)`。

## 11. `catalog.video_user_states`

`catalog.video_user_states` 是用户对视频互动状态的聚合投影表。它是读模型，不是事件真相表。其职责是描述用户是否点赞、收藏、观看、观看了多少、是否看完过等互动事实投影。它**不承接系统推荐曝光状态**；推荐曝光状态应继续放在 `recommendation.user_video_serving_states`。

建议字段如下：

- `user_id uuid not null`
- `video_id uuid not null`
- `has_liked boolean not null default false`
- `has_bookmarked boolean not null default false`
- `has_watched boolean not null default false`
- `liked_at timestamptz null`
- `bookmarked_at timestamptz null`
- `first_watched_at timestamptz null`
- `last_watched_at timestamptz null`
- `watch_count integer not null default 0`
- `completed_count integer not null default 0`
- `last_watch_ratio numeric(6,5) null`
- `max_watch_ratio numeric(6,5) null`
- `updated_at timestamptz not null default now()`

必要约束应包括：`primary key (user_id, video_id)`、`foreign key (user_id) references auth.users(id) on delete cascade`、`foreign key (video_id) references catalog.videos(video_id) on delete cascade`、`check (watch_count >= 0)`、`check (completed_count >= 0)`、`check (last_watch_ratio is null or (last_watch_ratio between 0 and 1))`、`check (max_watch_ratio is null or (max_watch_ratio between 0 and 1))`。推荐索引包括：`primary key (user_id, video_id)`、`(video_id)`、`(user_id, last_watched_at desc)`。这里明确不保留旧版 `occurred_at`，因为它无法清晰表示 like/bookmark/watch 三类行为中到底是哪一个事件时间。

## 12. 原始 transcript JSON 的保留策略

原始 transcript JSON 仍然保留在对象存储中，作为权威原始输入与审计来源。数据库不把整坨 JSON 存成主读模型，而是依赖 `catalog.video_transcripts`、`catalog.video_transcript_sentences`、`catalog.video_semantic_spans` 这三层结构化读模型对外提供查询能力。HLS 路径放在 `catalog.videos`，transcript 原始 JSON 路径、checksum、format version 放在 `catalog.video_transcripts`。这种拆分的原因很简单：HLS 属于视频播放主资产，transcript 路径属于 transcript 原始来源与 transcript 读模型的上游元数据。

## 13. 入库流程设计

Catalog 的入库采用**单 clip 单事务 replace 写入，并为每次执行记录单条入库审计**。这意味着：每个 clip 的一次执行都由 `video_ingestion_records` 记录；每个 clip 在数据库中要么整体成功，要么整体不落地；transcript 展开和 `video_unit_index` 聚合都在同一事务内完成。

单 clip 入库步骤应固定为：先读取 manifest 与 transcript JSON；再校验内容资产元数据，包括 `source_clip_key`、`parent_video_name`、`title`、`duration_ms`、`hls_master_playlist_path`、`transcript_object_path`、`transcript_checksum` 必须合法；再校验 transcript 结构，包括存在 `sentences`、sentence 的 `index/text/start/end` 合法、`end_ms > start_ms`、span 的 `index/text/start/end` 合法、span 时间必须落在所属 sentence 时间区间内、若 `coarse_id` 非空则必须存在于 `semantic.coarse_unit`；之后在应用层归一化生成 `full_text`、sentence rows、semantic span rows 和 `video_unit_index` 聚合结果；最后在同一事务中依次 upsert `catalog.videos`、replace `catalog.video_transcripts`、replace `catalog.video_transcript_sentences`、replace `catalog.video_semantic_spans`、replace `catalog.video_unit_index`，成功则写 `status = succeeded` 并关联 `video_id`，失败则回滚并写 `status = failed`、`error_code` 与 `error_message`。

## 14. 轻量后处理逻辑

这里的后处理不是另一条重型异步流水线，而是单 clip 入库过程中的轻量派生计算。首先，根据 sentence 与 spans 生成 transcript 摘要：`full_text`、`sentence_count`、`semantic_span_count`、`mapped_span_count`、`unmapped_span_count`、`mapped_span_ratio`，写入 `catalog.video_transcripts`。其次，从 `catalog.video_semantic_spans` 中所有 `coarse_unit_id is not null` 的行，按 `(video_id, coarse_unit_id)` 聚合出 `mention_count`、`sentence_count`、`first_start_ms`、`last_end_ms`、`sentence_indexes`、`sample_surface_forms`。再次，计算 `coverage_ms` 时，不能简单累加 span 时长，而应先取同一 `(video_id, coarse_unit_id)` 下所有 span 区间，按开始时间排序，合并重叠区间，再对合并后的区间求总时长，然后计算 `coverage_ratio = coverage_ms / videos.duration_ms`。最后，生成轻量 evidence refs。当前最终设计中，这一步的目标不是“定义 best”，而是生成一组代表性 `evidence_span_refs`。

在生成 `evidence_span_refs` 时，推荐使用稳定且低语义承诺的规则。例如：对同一 `(video_id, coarse_unit_id, sentence_index)` 只保留该 sentence 中最早的一个 span；再按 `sentence_index ASC, span_index ASC` 排序；最后取前 `K` 个，推荐 `K = 5`。这既能保证代表性，又避免 Catalog 提前固化“best evidence”的高语义承诺。

## 15. 幂等与更新策略

Catalog 的幂等锚点是 `source_clip_key`。只要该值稳定，就能把一次导入与数据库中的唯一 clip 实体对齐。

跳过策略如下：如果 `source_clip_key` 已存在、transcript checksum 未变化、HLS 路径未变化、主要元数据未变化，则当前导入项可直接标记为 `skipped`。replace 策略如下：如果 transcript checksum 变化、HLS 路径变化、时长变化、transcript 结构化内容变化、或标题/描述变化，则执行完整 replace 写入，即重写 `video_transcripts`、`video_transcript_sentences`、`video_semantic_spans`、`video_unit_index`。这样既保证幂等，也保证当事实发生变化时，所有派生层保持一致。

## 16. 与 Recommendation / Recall 的数据契约

当前这版 Catalog 直接决定了 Recommendation 与 Recall 的主读路径。粗召回入口是：`target coarse_unit_ids -> catalog.video_unit_index -> candidate videos`；需要 explanation/jump-to/精细证据时，再通过 `evidence_span_refs` 回查 `catalog.video_semantic_spans`，必要时 join `catalog.video_transcript_sentences`。Catalog 只保证：`mention_count`、`sentence_count`、`coverage_ms`、`coverage_ratio`、`sentence_indexes`、`sample_surface_forms`、`duration_ms`、`parent_video_slug` 这些稳定信号可用。Catalog 不保证高层语义标签，也不返回“最终 best evidence”；最终 `best_evidence_start_ms/end_ms` 等字段应由 Recommendation 在本轮 run 的聚合与解释阶段动态选出。

同时要明确：Recommendation 需要的 `v_recommendable_video_units`、`v_unit_video_inventory`、`user_video_serving_states`、`video_recommendation_runs`、`video_recommendation_items` 都属于 `recommendation` schema，而不是 Catalog 的一部分。Catalog 只提供内容事实与内容索引，不拥有 Recommendation serving/audit 对象。

其中 `recommendation.v_recommendable_video_units` 的 owner、字段 contract、过滤规则与刷新策略，以《全新设计-推荐模块设计.md》中的权威定义为准。

## 17. 明确不保留或不建立的结构

为了避免新旧架构混杂，当前最终设计明确不保留旧 `videos` 流水线字段，例如 `upload_user_id`、`raw_file_reference`、`media_status`、`analysis_status`、`media_job_id`、`analysis_job_id`、各种原始文件参数与中间态错误字段。这些都不属于当前内容资产层。

当前也不建立 `catalog.video_segments`。因为数据库中只存切片视频，`video_id` 本身就是最终内容对象，再建 segment 会让对象层级混乱。与之对应，也不建立 `catalog.segment_unit_mappings`；这一层职责已经由 `catalog.video_semantic_spans` 与 `catalog.video_unit_index` 共同承担。

同样，不在当前版本的 `video_unit_index` 中提前固化 `role`、`context_relevance`、`teachability_score`、`confidence_score` 等高层语义字段；也不把 Recommendation 投放状态塞进 `catalog.video_user_states`。这些都违反当前 Catalog 的边界定义。

## 18. 当前实现约束

本文档定义的是 Catalog 的最终逻辑 schema 与最终读写契约，因此当前实现必须满足以下约束：

- `catalog.video_unit_index` 的最终证据表达以 `evidence_span_refs` 为准；`evidence_sentence_indexes` 与 `evidence_span_indexes` 不属于当前逻辑 schema 与对外契约。
- Recommendation 新读路径的正式启用前提，以《全新设计-总设计.md》与《全新设计-推荐模块设计.md》中的系统级契约为准。
- 若现网仍存在旧版结构、双写、回填或 reader 切换需求，应在单独的迁移/实施文档中描述，不纳入本最终设计文档。

## 19. 最终结论

当前 Catalog 的最终设计可以压缩成一句话：

> `catalog` 是一个只负责切片视频内容资产、transcript 结构化读模型、span 聚合索引、单 clip 入库审计和用户视频互动投影的内容事实与内容索引域；它以 `catalog.videos -> catalog.video_transcripts -> catalog.video_transcript_sentences -> catalog.video_semantic_spans -> catalog.video_unit_index` 为核心数据链，以 `video_unit_index.evidence_span_refs` 作为轻量、无歧义、可回查的证据引用表达，并通过单 clip 单事务 replace 写入、幂等的 `source_clip_key`、以及确定性的轻量聚合逻辑，稳定为 Recommendation 与 Recall 提供输入。`catalog` 不拥有 Learning state，不拥有 Recommendation serving/audit，不维护媒体/AI 流水线状态机，也不提前固化 Recommendation 解释层的高语义判断。
