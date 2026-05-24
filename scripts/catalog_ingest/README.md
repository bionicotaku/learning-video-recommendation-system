# Catalog mapped JSON 入库方案

## 目标

`scripts/catalog_ingest` 将本地 mapped clip transcript JSON 和同名 question JSON 导入 `catalog` 数据库。

当前入口只处理本地 JSON，不读取视频文件本体，不上传对象存储。视频、transcript 和封面对象路径按固定 GCS URL 模板由 transcript 文件名派生。

## 输入

脚本使用两个目录：

- `--transcripts-dir`：mapped clip transcript JSON 目录。每个 JSON 就是一条最终 clip。
- `--questions-dir`：clip question JSON 目录。文件名必须与 mapped transcript JSON 同名。

不再需要 `--parents-dir`。旧父视频切片描述文件中的 clip metadata 已经合并到每个 mapped transcript JSON 顶层。

### mapped transcript JSON

每个 transcript 文件名必须形如：

```text
<parent_video_name>-clip<clip_id>.json
```

顶层必须包含：

- `clip_id`
- `title`
- `description`
- `engagement`
- `start_index`
- `end_index`
- `start_time`
- `end_time`
- `buffered_start_time`
- `buffered_end_time`
- `duration_time`
- `reasoning`
- `sentences`

`clip_id` 必须与文件名中的 `-clipN` 一致。`duration_time` 必须等于 `buffered_end_time - buffered_start_time`。

`sentences[]` 继续承载 transcript 结构：

- sentence: `index`, `text`, `translation`, `start`, `end`, `tokens`
- token: `index`, `text`, `explanation`, `start`, `end`, `semantic_element`
- semantic element: `coarse_id`, `base_form`, `dictionary`, `translation`, `reason`

### question JSON

question 文件必须与 mapped transcript 同名。顶层必须包含：

- `source`
- `questions`
- `audit`
- `selected_coarse_unit_refs`

`questions[]` 写入 `catalog.questions`。本入口只写视频上下文题：

- `scope_type` 可省略，省略时按 `video_unit` 处理
- 如果显式提供，必须是 `video_unit`
- 每道题必须包含完整 context 字段
- `content_payload` 原样写入数据库

`selected_coarse_unit_refs.refs[]` 是 `video_unit_index.best_evidence_*` 的来源。每个 ref 至少包含：

- `coarse_unit_id`
- `target_text`
- `sentence_index`
- `token_index`
- `scores`
- `candidate_score`
- `question_reject_reason`
- `selection_reason`

`token_index` 映射为数据库中的 `span_index`。

## 字段映射

### `catalog.videos`

- `source_clip_key`：`<parent_video_slug>#clip<clip_id>`
- `parent_video_name`：从 transcript 文件名去掉末尾 `-clipN` 得到
- `parent_video_slug`：由 `parent_video_name` 规范化得到
- `clip_seq`：`clip_id`
- `source_start_ms`：`buffered_start_time`
- `source_end_ms`：`buffered_end_time`
- `source_start_sentence_index`：mapped transcript 顶层 `start_index`
- `source_end_sentence_index`：mapped transcript 顶层 `end_index`
- `title`：mapped transcript 顶层 `title`
- `description`：mapped transcript 顶层 `description`
- `clip_reason`：mapped transcript 顶层 `reasoning`
- `engagement_score`：mapped transcript 顶层 `engagement`，以 jsonb 对象保存
- `language`：当前固定 `en`
- `duration_ms`：`duration_time`
- `video_object_path`：`https://storage.googleapis.com/videos2077/test-video/portrait_videos/<transcript_file_stem>.mp4`
- `thumbnail_url`：`https://storage.googleapis.com/videos2077/test-video/cover/<transcript_file_stem>.webp`
- `status`：当前固定 `active`
- `visibility_status`：当前固定 `public`
- `publish_at`：脚本本次执行时的 UTC `now`

`engagement_score` 是内容侧打分，不等同于 `catalog.video_engagement_stats` 的用户行为统计。当前以 jsonb 保存 `drama/humor/payoff/standalone/reasoning`，不单独建索引。

### `catalog.video_transcripts`

- `transcript_object_path`：`https://storage.googleapis.com/videos2077/test-video/transcript/<transcript_file_name>.json`
- `transcript_checksum`：transcript 文件原始 bytes 的 sha256
- `transcript_format_version`：当前固定 `1`
- `sentence_count`：sentence 数
- `semantic_span_count`：token/span 数
- `mapped_span_count`：`coarse_id` 非空 span 数
- `unmapped_span_count`：`coarse_id` 为空 span 数
- `mapped_span_ratio`：`mapped_span_count / semantic_span_count`

### `catalog.video_transcript_sentences`

每个 sentence 映射为一行：

- `sentence_index`：`sentence.index`
- `start_ms`：`sentence.start`，clip-local 绝对毫秒；已相对 `buffered_start_time` 归零，不是父视频全局时间
- `end_ms`：`sentence.end`，clip-local 绝对毫秒；已相对 `buffered_start_time` 归零，不是父视频全局时间
- `text`：`sentence.text`
- `translation`：`sentence.translation`

### `catalog.video_semantic_spans`

每个 token 映射为一行：

- `sentence_index`：所属 sentence 的 `index`
- `span_index`：`token.index`
- `start_ms`：`token.start`，clip-local 绝对毫秒；已相对 `buffered_start_time` 归零，不是父视频全局时间
- `end_ms`：`token.end`，clip-local 绝对毫秒；已相对 `buffered_start_time` 归零，不是父视频全局时间
- `coarse_unit_id`：`token.semantic_element.coarse_id`
- `surface_text`：`token.text`
- `explanation`：`token.explanation`
- `base_form`：`token.semantic_element.base_form`
- `translation`：`token.semantic_element.translation`
- `dictionary`：`token.semantic_element.dictionary`
- `mapping_reason`：`token.semantic_element.reason`

### `catalog.video_unit_index`

由 `video_semantic_spans` 聚合生成：

- 按 `(video_id, coarse_unit_id)` 分组
- 统计 `mention_count`
- 统计 `sentence_count`
- 合并 span 区间得到 `coverage_ms`
- 计算 `coverage_ratio`
- 生成 `sentence_indexes`
- `best_evidence_sentence_index/span_index` 来自 `selected_coarse_unit_refs.refs[]`
- `best_evidence_start_ms/end_ms` 来自 selected ref 精确命中的 transcript token/span
- `best_evidence_scores` 来自 `selected_coarse_unit_refs.refs[].scores`
- `best_evidence_question_reject_reason` 来自 `selected_coarse_unit_refs.refs[].question_reject_reason`
- `best_evidence_selection_reason` 来自 `selected_coarse_unit_refs.refs[].selection_reason`
- `best_evidence_candidate_score` 来自 `selected_coarse_unit_refs.refs[].candidate_score`
- `best_evidence_target_text` 来自 `selected_coarse_unit_refs.refs[].target_text`

selected ref 必须能精确回查到同一 `(video_id, coarse_unit_id)` 下的 semantic span。

### `catalog.questions`

- `question_id`：脚本根据稳定输入生成 deterministic UUID
- `scope_type`：固定写入 `video_unit`
- `video_id`：当前 clip upsert 后回填
- `status`：固定写入 `active`，忽略 question JSON 中的 `status`
- `content_payload`：question JSON 原样写入
- 本次未出现但属于同一 video 的旧 `video_unit` question 会更新为 `retired`

当前 ingest 只写题库内容，不写 `analytics.quiz_events`。

question JSON 顶层 `source`、`audit` 和 `selected_coarse_unit_refs` 的选择参数元信息会写入 `catalog.video_ingestion_records.context`，用于复现和排查生成过程。

## 校验

阻断性错误：

- transcript 顶层不是对象
- 文件名不是 `<parent>-clipN.json`
- 文件名 `clip_id` 与 JSON 顶层 `clip_id` 不一致
- 缺少 required clip metadata
- `duration_time != buffered_end_time - buffered_start_time`
- transcript 为空
- transcript sentence 时间不在当前 clip 的 `0..duration_time` 区间内
- sentence/token 索引重复
- token 缺少 `semantic_element`
- 非空 `coarse_id` 不存在于 `semantic.coarse_unit`
- question 文件缺失以外的 question JSON 结构错误
- selected refs 与 transcript 中 mapped coarse unit 集合不一一对应

非阻断 warning：

- `token.start/end` 超出所属 sentence 区间时记 `token_time_outside_sentence`，继续入库

question 文件缺失时，该 clip 记为 `skipped / question_missing`，不写业务表。

## 幂等写入

幂等锚点是 `source_clip_key`。脚本按 `source_clip_key` upsert `catalog.videos`，并复用已有 `video_id`。

成功写入采用单 clip 单事务 replace：

1. 插入 running ingestion record
2. upsert `catalog.videos`
3. 删除该 `video_id` 的旧 transcript / sentence / span / unit_index 行
4. 写入当前 transcript / sentence / span / unit_index 行
5. upsert 当前 question 行
6. retire 当前 video 下本次未出现的旧 video_unit question
7. 更新 ingestion record 为 succeeded

如果 question JSON 存在，脚本不会走 unchanged skip 优化，确保题目和 selected refs 的变更能被写入。

整批处理结束后，只要本次至少有一个 clip 成功写入，脚本会调用 Recommendation owner 命令刷新 recall projection：

```bash
go run ./cmd/dbtool refresh recommendation
```

该步骤刷新 `recommendation.v_video_unit_recall_index`、`recommendation.v_unit_video_inventory` 和 `recommendation.recall_projection_metadata`。Catalog 入库不直接写 Recommendation schema；刷新失败不会回滚已经提交的 Catalog clip 事务，但脚本最终返回非零，调用方应视为需要处理的后置失败。

## 运行

```bash
.venv/bin/python -m scripts.catalog_ingest.main \
  --transcripts-dir /path/to/mapped \
  --questions-dir /path/to/questions \
  --source-name local-json
```

常用参数：

- `--dry-run`：只读取、校验和归一化，不写数据库
- `--limit N`：最多处理 N 个 clip
- `--clip-key <source_clip_key>`：只处理指定 clip
- `--time-tolerance-ms N`：允许 transcript 时间轴偏离 buffered 区间 N 毫秒
- `--skip-recommendation-refresh`：跳过成功写入后的 Recommendation recall projection 刷新

数据库连接从环境变量 `DATABASE_URL` 读取；如果环境变量不存在，则读取项目根目录 `.env`。
