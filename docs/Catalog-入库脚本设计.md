# Catalog 入库脚本设计文档

## 1. 文档目标

本文档定义当前阶段 `catalog` 数据库入库脚本的实现方案。

这里的“入库脚本”特指：

- 读取本地已有的切片视频元数据
- 读取本地已有的 transcript JSON
- 读取父视频与切片来源信息
- 生成符合 `catalog` schema 的标准化行数据
- 将单个 clip 以事务方式写入数据库
- 将每次执行写入 `catalog.video_ingestion_records`

本文档不定义：

- 视频切片生成
- transcript 生成
- coarse unit 映射算法
- recommendation / recall 逻辑
- 用户行为投影写入

本脚本的目标是：

> 把已经准备好的本地 clip 资产与 transcript 数据，稳定、可校验、可追踪地导入 `catalog` schema。

---

## 2. 适用前提

当前脚本建立在以下前提上：

1. 视频已经完成切片，数据库中只存切片视频。
2. 每个 clip 已经有稳定的 `source_clip_key`。
3. 每个 clip 已经有本地可读的 transcript JSON。
4. transcript JSON 中已经包含 sentence 和 span 级时间信息。
5. span 中的 `coarse_id` 已经由上游流程产出。
6. 数据库 schema 已按 `docs/Catalog-数据库设计.md` 建好。

因此脚本不负责“生成内容”，只负责“导入和归一化”。

---

## 3. 脚本边界

### 3.1 脚本负责什么

- 读取本地 manifest / 文件路径 / transcript JSON
- 校验输入元数据与 transcript 结构
- 归一化为数据库标准行
- 聚合构建 `catalog.video_unit_index`
- 用单 clip 单事务写入：
  - `catalog.videos`
  - `catalog.video_transcripts`
  - `catalog.video_transcript_sentences`
  - `catalog.video_semantic_spans`
  - `catalog.video_unit_index`
- 写 `catalog.video_ingestion_records`

### 3.2 脚本不负责什么

- 不创建 `semantic.coarse_unit`
- 不修复错误 transcript
- 不做 recommendation 审计
- 不写 `learning.*`
- 不写 `recommendation.*`
- 不做批量 job 调度系统

---

## 4. 总体执行模型

脚本采用：

**单 clip 单事务 replace 写入**

对每个 clip，执行顺序统一为：

```text
读取输入
  ↓
校验
  ↓
归一化
  ↓
构建 transcript 摘要与 unit 索引
  ↓
开启事务
  ↓
upsert videos
  ↓
replace video_transcripts
  ↓
replace video_transcript_sentences
  ↓
replace video_semantic_spans
  ↓
replace video_unit_index
  ↓
写 video_ingestion_records
  ↓
提交事务
```

失败时：

- 回滚当前 clip 的业务写入
- 将本次执行标记为 `failed`
- 记录 `error_code` 与 `error_message`

---

## 5. 模块拆分

为了保证脚本可维护、可测试、可替换，当前建议拆成 7 个 Python 文件。

目录如下：

```text
scripts/catalog_ingest/
  __init__.py
  main.py
  models.py
  manifest_loader.py
  validator.py
  normalizer.py
  index_builder.py
  repository.py
```

### 5.1 `main.py`

职责：

- 提供脚本入口
- 解析命令行参数
- 组装数据库连接
- 按 clip 逐条编排完整导入链路
- 汇总执行结果并设置退出码

它只做编排，不做业务细节。

### 5.2 `models.py`

职责：

- 定义脚本内部的数据结构
- 明确模块间传递的数据边界

建议放这些类型：

- 原始输入对象
- 归一化后的 clip 数据对象
- transcript 句子对象
- semantic span 对象
- unit index 聚合对象
- 入库结果对象

### 5.3 `manifest_loader.py`

职责：

- 从本地读取 manifest 或输入目录
- 组装单个 clip 的原始输入对象
- 解析：
  - 视频资产路径
  - transcript JSON 路径
  - `source_clip_key`
  - `parent_video_name`
  - `parent_video_slug`
  - `clip_seq`
  - `source_start_ms`
  - `source_end_ms`
  - `title`
  - `description`
  - `clip_reason`
  - `language`
  - `thumbnail_url`
  - `publish_at`

它不负责业务校验，只负责“把输入读出来”。

### 5.4 `validator.py`

职责：

- 校验原始输入是否满足 Catalog 文档要求
- 尽早发现不能入库的坏数据

至少要校验：

- `source_clip_key` 非空
- `parent_video_name` 非空
- `parent_video_slug` 非空
- `title` 非空
- `duration_ms > 0`
- `hls_master_playlist_path` 非空
- `transcript_object_path` 非空
- `transcript_checksum` 非空
- transcript 顶层存在 `sentences`
- sentence 的 `index / text / start / end` 合法
- span 的 `index / text / start / end` 合法
- span 时间必须落在所属 sentence 区间内
- 非空 `coarse_id` 在 `semantic.coarse_unit` 中存在

输出应是：

- 校验通过
- 或结构化异常，带明确错误码

### 5.5 `normalizer.py`

职责：

- 把原始输入转成数据库标准行模型
- 生成：
  - `catalog.videos` row
  - `catalog.video_transcripts` row
  - `catalog.video_transcript_sentences` rows
  - `catalog.video_semantic_spans` rows

这个模块不做数据库写入。

它只负责把输入转成“数据库应该长什么样”。

### 5.6 `index_builder.py`

职责：

- 从 sentence / span 标准行中派生聚合结果
- 生成：
  - transcript 摘要统计
  - `catalog.video_unit_index` rows

至少包括：

- `full_text`
- `sentence_count`
- `semantic_span_count`
- `mapped_span_count`
- `unmapped_span_count`
- `mapped_span_ratio`
- `mention_count`
- `sentence_count`
- `first_start_ms`
- `last_end_ms`
- `coverage_ms`
- `coverage_ratio`
- `sentence_indexes`
- `evidence_sentence_indexes`
- `evidence_span_indexes`
- `sample_surface_forms`

### 5.7 `repository.py`

职责：

- 管理数据库连接与事务
- 提供单 clip 写库方法
- 将 replace / upsert 细节收口到一个地方

必须保证：

- 单 clip 在一个事务内完成
- 成功时提交
- 失败时回滚
- 审计记录和业务写入状态一致

---

## 6. 输入模型

脚本应围绕“单 clip 输入对象”工作。

建议最小输入字段如下：

```text
source_clip_key
parent_video_name
parent_video_slug
clip_seq
source_start_ms
source_end_ms
title
description
clip_reason
language
duration_ms
hls_master_playlist_path
thumbnail_url
publish_at
transcript_object_path
transcript_checksum
transcript_format_version
transcript_json
source_name
context
```

说明：

- `transcript_object_path` 是数据库中记录的对象路径
- `transcript_json` 是本地当前真正读取到的原文对象
- `source_name` 和 `context` 用于写 `video_ingestion_records`

---

## 7. 业务唯一键与数据库主键

入库脚本里必须明确区分：

- `source_clip_key`
- `video_id`

### 7.1 `source_clip_key`

`source_clip_key` 是业务侧稳定唯一键。

它负责回答：

- 这个外部 clip 是谁
- 本次导入是否命中了同一个 clip

它应由输入稳定生成，并在重复导入时保持不变。

### 7.2 `video_id`

`video_id` 是数据库内部主键。

它负责：

- 作为 `catalog.videos` 的主键
- 作为其他子表的外键关联键

### 7.3 当前推荐规则

在当前“父文件 + clip_id + transcript 文件”方案下，推荐：
在当前实现里，这条规则直接固定为正式入库规则：

```text
source_clip_key = <parent_video_slug>#clip<clip_id>
```

而 `video_id` 仍由数据库生成或在 upsert 后取回。

### 7.4 当前 `clip_seq`

当前样本下：

```text
clip_seq = clip_id
```

因为输入中的 `clip_id` 已经天然表示该 clip 在父视频中的顺序。
当前实现里，这不是“临时假设”，而是正式入库规则。

---

## 8. 数据库写入策略

### 8.1 写入顺序

单 clip 固定按以下顺序写入：

1. `catalog.videos`
2. `catalog.video_transcripts`
3. `catalog.video_transcript_sentences`
4. `catalog.video_semantic_spans`
5. `catalog.video_unit_index`
6. `catalog.video_ingestion_records`

### 8.2 `catalog.videos`

按 `source_clip_key` 做幂等 upsert。

如果已存在，则更新：

- `parent_video_name`
- `parent_video_slug`
- `clip_seq`
- `source_start_ms`
- `source_end_ms`
- `title`
- `description`
- `clip_reason`
- `language`
- `duration_ms`
- `hls_master_playlist_path`
- `thumbnail_url`
- `status`
- `visibility_status`
- `publish_at`
- `updated_at`

### 8.3 其余四张内容表

对以下表采用 replace 策略：

- `catalog.video_transcripts`
- `catalog.video_transcript_sentences`
- `catalog.video_semantic_spans`
- `catalog.video_unit_index`

也就是：

- 先按 `video_id` 删除旧明细
- 再写入新明细

原因是 transcript 和 unit index 是从当前输入完整派生的，不适合做局部 patch。

### 8.4 审计表

每次执行都写一条新的 `catalog.video_ingestion_records`。

这意味着审计是“执行历史”，不是“当前快照”。

建议规则：

- 开始时先插入 `running`
- 成功后更新为 `succeeded`
- 失败后更新为 `failed`
- 无变化时可直接记为 `skipped`

---

## 9. 幂等与跳过策略

幂等锚点使用：

- `source_clip_key`

跳过条件建议为：

- `source_clip_key` 已存在
- transcript checksum 未变化
- HLS 路径未变化
- 主要元数据未变化

满足时：

- 不重写四张内容表
- 写一条 `video_ingestion_records`
- `status = skipped`

Replace 条件建议为：

- transcript checksum 变化
- HLS 路径变化
- 时长变化
- transcript 结构化内容变化
- 标题或描述变化

这里的标题直接使用工程标题，也就是 transcript 文件 basename 去掉 `.json` 后的值。
当前阶段不额外引入展示标题层。

这里的 `hls_master_playlist_path` 和 `transcript_object_path` 允许先写入带前缀的占位路径。
后续资产上传到存储桶后，再通过独立补写流程更新为真实对象路径。

---

## 10. 错误模型

脚本应优先输出结构化错误，而不是只有异常堆栈。

建议至少区分：

- `manifest_invalid`
- `transcript_invalid`
- `coarse_unit_missing`
- `db_connect_failed`
- `db_tx_failed`
- `db_write_failed`
- `index_build_failed`
- `unknown_error`

每个错误都应尽量带：

- `error_code`
- `error_message`
- `source_clip_key`
- 失败阶段

---

## 11. CLI 设计建议

当前阶段 CLI 不需要复杂。

建议支持：

```text
--manifest <path>
--input-dir <path>
--source-name <text>
--limit <n>
--dry-run
--clip-key <source_clip_key>
```

说明：

- `--manifest` 用于显式清单输入
- `--input-dir` 用于扫描目录
- `--clip-key` 用于单条重跑
- `--dry-run` 只校验和归一化，不写库

---

## 12. 第一阶段开发范围

第一阶段只做 MVP 必需能力：

1. 读取 manifest
2. 校验 transcript
3. 归一化四张内容表所需数据
4. 构建 `video_unit_index`
5. 单 clip 事务写库
6. 写 `video_ingestion_records`
7. 支持 `dry-run`

当前明确不做：

- 并发导入
- 批次级 job 管理
- 失败自动重试
- 对象存储上传
- 富日志平台集成
- 指标系统

---

## 13. 文件职责清单

当前阶段各文件职责应固定如下：

```text
scripts/catalog_ingest/main.py
  只做 CLI 和总编排

scripts/catalog_ingest/models.py
  只放内部数据结构

scripts/catalog_ingest/manifest_loader.py
  只做输入读取

scripts/catalog_ingest/validator.py
  只做校验

scripts/catalog_ingest/normalizer.py
  只做标准化

scripts/catalog_ingest/index_builder.py
  只做摘要和 unit index 聚合

scripts/catalog_ingest/repository.py
  只做事务和 SQL 写入
```

不要把所有逻辑堆到 `main.py`。

---

## 14. 后续扩展方向

如果第一阶段稳定，后续可继续加：

- 单元测试
- fixture transcript
- integration test
- 并发执行器
- 更细粒度的 evidence 生成
- 从本地文件切换到对象存储输入

但这些都应建立在当前单 clip 入库链路先稳定的前提上。

---

## 15. 结论

当前 Catalog 入库脚本最合适的实现方式是：

**以 Python 编写一个单 clip 单事务的导入脚本，将读取、校验、标准化、索引构建、事务写库和审计写入明确拆分到独立模块中，并以 `scripts/catalog_ingest/` 作为脚本目录根。**

这样做的直接好处是：

- 边界清楚
- 错误定位清楚
- 易于 dry-run
- 易于后续补测试
- 不会把导入脚本写成一个不可维护的大文件
