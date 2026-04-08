# Catalog 本地 JSON 入库方案

## 1. 文档目标

本文档定义当前阶段基于**两类本地 JSON 文件**的 `catalog` 入库方案。

当前目标不是读取视频文件本体，也不是上传对象存储，而是：

- 读取父视频切片描述文件
- 读取每个 clip 的 transcript JSON
- 将数据映射到 `catalog` schema
- 支撑后续 Python 入库脚本实现

本文档只解决当前这条数据链路，不扩展到更通用的媒体入库系统。

---

## 2. 实际样本

当前仓库中的样本文件为：

- 父视频切片描述文件：
  - [The Office (US) (2005) - S01E01 - Pilot (1080p BluRay x265 Silence).json](/Users/evan/Downloads/learning-video-recommendation-system/scripts/catalog_ingest/samples/The%20Office%20%28US%29%20%282005%29%20-%20S01E01%20-%20Pilot%20%281080p%20BluRay%20x265%20Silence%29.json)
- clip transcript 文件：
  - [The Office (US) (2005) - S01E01 - Pilot (1080p BluRay x265 Silence)-clip1.json](/Users/evan/Downloads/learning-video-recommendation-system/scripts/catalog_ingest/samples/The%20Office%20%28US%29%20%282005%29%20-%20S01E01%20-%20Pilot%20%281080p%20BluRay%20x265%20Silence%29-clip1.json)

从样本中确认到的事实如下：

### 2.1 父视频描述文件结构

顶层只有：

- `clips`

每个 `clip` 至少包含：

- `clip_id`
- `start_index`
- `end_index`
- `start_time`
- `end_time`
- `buffered_start_time`
- `buffered_end_time`
- `reasoning`

例如首个 clip：

```json
{
  "clip_id": 1,
  "start_index": 0,
  "end_index": 22,
  "start_time": 55416,
  "end_time": 113628,
  "buffered_start_time": 54516,
  "buffered_end_time": 114370
}
```

### 2.2 transcript 文件结构

顶层只有：

- `sentences`

每个 sentence 至少包含：

- `index`
- `text`
- `explanation`
- `tokens`
- `start`
- `end`

每个 token 至少包含：

- `text`
- `explanation`
- `index`
- `start`
- `end`
- `semanticElement`

而 `semanticElement` 至少包含：

- `baseForm`
- `dictionary`
- `coarse_id`
- `reason`

### 2.3 时间单位确认

样本中的时间字段和 `Catalog` 文档是一致的，使用**毫秒**。

### 2.4 时间轴语义确认

当前 clip transcript 中 sentence / token 的 `start`、`end` 不是 clip 内相对时间，而是**沿用父视频绝对时间轴**。

证据：

- 父文件中 clip1 的 `buffered_start_time = 54516`
- clip1 transcript 中首句 `start = 55416`
- clip1 transcript 中末句 `end = 113628`

这说明 transcript 的时间可以直接用于：

- `source_start_ms`
- `source_end_ms`
- sentence `start_ms / end_ms`
- span `start_ms / end_ms`

不需要额外做 clip 内归零换算。

---

## 3. 输入目录约定

当前方案采用两个输入目录。

### 3.1 目录 A：父视频切片描述目录

目录 A 中每个文件都表示一个父视频来源，例如：

```text
The Office (US) (2005) - S01E01 - Pilot (1080p BluRay x265 Silence).json
```

这个文件**不直接入库为实体**。
它只提供：

- 父视频来源名
- clip 列表
- 每个 clip 的切片位置

### 3.2 目录 B：clip transcript 目录

目录 B 中每个文件都表示一个最终 clip transcript，例如：

```text
The Office (US) (2005) - S01E01 - Pilot (1080p BluRay x265 Silence)-clip1.json
```

这个文件对应一条最终会写入 `catalog.videos` 的切片视频。

### 3.3 文件匹配规则

对目录 A 中某个父文件：

```text
<parent_name>.json
```

其某个 clip 的 transcript 文件名规则为：

```text
<parent_name>-clip<clip_id>.json
```

例如：

- 父文件：`The Office ... Silence).json`
- `clip_id = 1`
- transcript 文件：`The Office ... Silence)-clip1.json`

这条规则当前已在样本中成立。

### 3.4 目录扫描边界

当前脚本应明确采用以下扫描边界：

- 只扫描目录根下的 `.json` 文件
- 忽略隐藏文件
- 忽略非 `.json` 文件
- 不递归扫描子目录

这样可以避免把临时文件、编辑器备份文件、其他调试 JSON 混进正式导入输入。

### 3.5 文件匹配异常

当前方案还必须明确三类异常：

- 父文件里存在 clip，但目录 B 中找不到对应 transcript 文件
- 目录 B 中存在 transcript 文件，但目录 A 中没有对应父文件或 clip
- 理论上应唯一匹配的 transcript 文件，实际出现多个候选

建议当前策略为：

- 缺 transcript：直接记 `skipped`
- 孤儿 transcript：默认不入库，汇总到脚本结束报告
- 多候选：直接视为失败，不做猜测匹配

---

## 4. 核心建模判断

### 4.1 父文件不是数据库实体

父视频切片描述文件不应在数据库里建实体表。

原因：

- `catalog` 当前只存切片视频
- 原始视频不入库
- 父文件只是导入输入与来源信息

因此：

- 文件名进入 `parent_video_name`
- 规范化后进入 `parent_video_slug`
- 不单独建 `parent_videos` 表

### 4.2 transcript 文件代表最终 clip

目录 B 中每个 transcript 文件对应一条最终 clip。

因此一条 clip 输入对象，必须同时绑定：

- 一个父文件中的一个 `clip`
- 一个 transcript 文件

### 4.3 入库对象仍然是切片视频

最终写入 `catalog.videos` 的主对象仍是 clip，不是 transcript，不是父视频。

---

## 5. 字段映射方案

以下映射以单个 clip 为单位。

### 5.0 主键与业务唯一键

这里必须先区分两个不同概念：

- `source_clip_key`：业务侧稳定唯一键
- `video_id`：数据库内部主键

当前规则下：

- `source_clip_key` 由脚本根据 `parent_video_slug + clip_id` 稳定生成
- `video_id` 是 `catalog.videos` 的 UUID 主键

它们的职责不同：

- `source_clip_key` 用于幂等 upsert，回答“这个外部 clip 是谁”
- `video_id` 用于数据库内的外键关联，回答“数据库里这条视频记录是谁”

因此导入流程应是：

1. 先根据输入生成 `source_clip_key`
2. 按 `source_clip_key` 查找或 upsert `catalog.videos`
3. 取得对应的 `video_id`
4. 用 `video_id` 写入其他子表

当前不应把 `source_clip_key` 当主键用，也不应把 `video_id` 用作外部幂等键。

这里还应补充两条明确规则：

- `clip_id` 当前必须是正整数
- `source_clip_key` 的生成公式固定为 `<parent_video_slug>#clip<clip_id>`，脚本内不要出现第二套拼接规则

## 5.1 `catalog.videos`

### `source_clip_key`

建议使用：

```text
<parent_video_slug>#clip<clip_id>
```

例如：

```text
the-office-us-2005-s01e01-pilot-1080p-bluray-x265-silence#clip1
```

原因：

- 比直接用 transcript 文件名更稳定
- 不依赖扩展名
- 可读性足够
- 幂等锚点清晰

当前不建议直接把 transcript 文件名原文作为 `source_clip_key`。

### `parent_video_name`

取父文件 basename 去掉 `.json` 后的字符串。

例如：

```text
The Office (US) (2005) - S01E01 - Pilot (1080p BluRay x265 Silence)
```

### `parent_video_slug`

由 `parent_video_name` 规范化得到。

规则建议：

- 全部转小写
- 非字母数字统一转 `-`
- 连续 `-` 合并
- 去掉首尾 `-`

若规范化结果为空字符串，则应直接报错，不允许写入空 slug。

### `clip_seq`

当前样本下，直接使用 `clip_id`。

也就是说：

```text
clip_seq = clip_id
```

原因是当前父文件中的 `clip_id` 本身就在表达：

- 该 clip 在父视频中的顺序编号

因此：

- `clip_id` 是输入 JSON 中的字段名
- `clip_seq` 是数据库中的标准字段名

在当前这套输入里，两者数值相同，不需要额外转换。

只有在未来上游 `clip_id` 不再等于顺序号时，才需要重新区分。

### `source_start_ms`

使用：

- `buffered_start_time`

### `source_end_ms`

使用：

- `buffered_end_time`

### `duration_ms`

使用：

```text
buffered_end_time - buffered_start_time
```

原因：

- 当前 clip transcript 的覆盖区间与主学习片段一致
- 但真正对外可播放的切片更接近 buffered 区间
- 这样与文档中的“切片视频对象”更一致

### `title`

当前建议使用 transcript 文件 basename 去掉 `.json` 后的值。

例如：

```text
The Office (US) (2005) - S01E01 - Pilot (1080p BluRay x265 Silence)-clip1
```

原因：

- 它天然唯一到 clip 粒度
- 先保证工程稳定，不必提前设计更复杂展示标题

后续如果有单独展示标题策略，再调整。

因此当前应把 `title` 明确视为工程标题，并直接作为当前阶段的正式入库值。
后续即使展示层要引入单独标题策略，也不影响当前这套入库规则。

### `description`

当前建议留空，或仅在未来有独立视频描述来源时再写入。

当前不建议把切片理由混入 `description`。

### `clip_reason`

直接使用父文件中对应 clip 的：

- `reasoning`

这是切片对象本身的来源解释，不是普通描述字段。

因此当前映射明确为：

```text
clip_reason = clip.reasoning
```

### `language`

当前固定：

- `en`

### `hls_master_playlist_path`

当前没有真实对象存储路径时，允许临时使用占位值，但必须有**明确前缀**，不能和 transcript 路径混为同一语义。

建议：

```text
placeholder://video/<title>
```

这是当前阶段的正式入库值。
后续视频真正上传到存储桶后，再以补写方式更新为真实对象路径。

### `thumbnail_url`

当前留空。

### `status`

当前写：

- `active`

### `visibility_status`

当前写：

- `public`

### `publish_at`

当前留空。

---

## 5.2 `catalog.video_transcripts`

### `video_id`

由 `catalog.videos` upsert 结果回填。

### `transcript_object_path`

当前没有真实对象存储路径时，允许临时使用占位值，但同样必须有独立前缀。

建议：

```text
placeholder://transcript/<transcript_file_name_without_ext>
```

这是当前阶段的正式入库值。
后续 transcript 真正上传到存储桶后，再以补写方式更新为真实对象路径。

### `transcript_checksum`

对 transcript 文件原始字节做内容哈希。

建议：

- `sha256`

这里不要先做 JSON pretty-print、字段重排或 Unicode 归一化后再算哈希，否则同一文件内容在不同脚本实现里可能得出不同 checksum。

### `transcript_format_version`

当前写：

- `1`

### 其余统计字段

由 transcript 内容派生：

- `full_text`
- `sentence_count`
- `semantic_span_count`
- `mapped_span_count`
- `unmapped_span_count`
- `mapped_span_ratio`

---

## 5.3 `catalog.video_transcript_sentences`

每个 sentence 映射为一行：

- `video_id`：回填
- `sentence_index`：`sentence.index`
- `text`：`sentence.text`
- `start_ms`：`sentence.start`
- `end_ms`：`sentence.end`
- `explanation`：`sentence.explanation`

---

## 5.4 `catalog.video_semantic_spans`

每个 token 映射为一行：

- `video_id`：回填
- `sentence_index`：所属 sentence 的 `index`
- `span_index`：`token.index`
- `text`：`token.text`
- `start_ms`：`token.start`
- `end_ms`：`token.end`
- `explanation`：`token.explanation`
- `coarse_unit_id`：`token.semanticElement.coarse_id`
- `base_form`：`token.semanticElement.baseForm`
- `dictionary_text`：`token.semanticElement.dictionary`

当前不把 `semanticElement.reason` 落到主表。

---

## 5.5 `catalog.video_unit_index`

完全由 `video_semantic_spans` 聚合生成，不直接从原始 JSON 写入。

聚合规则沿用 `docs/Catalog-数据库设计.md`：

- `(video_id, coarse_unit_id)` 分组
- 统计 `mention_count`
- 统计 `sentence_count`
- 计算 `first_start_ms`
- 计算 `last_end_ms`
- 合并 span 区间得到 `coverage_ms`
- 计算 `coverage_ratio`
- 生成 `sentence_indexes`
- 生成 `evidence_sentence_indexes`
- 生成 `evidence_span_indexes`
- 生成 `sample_surface_forms`

这里需要补一个实现约定：

- `evidence_sentence_indexes` 与 `evidence_span_indexes` 必须按相同位置一一对应解释

原因是单独的 `span_index` 在不同 sentence 内可能重复；当前 schema 没有单独的复合 evidence 结构，所以脚本实现里必须保证两组数组等长，且第 `i` 个 sentence index 与第 `i` 个 span index 组成一条 evidence。

---

## 5.6 `catalog.video_ingestion_records`

每次单 clip 执行写一条审计记录。

建议映射：

- `ingestion_record_id`：新 UUID
- `source_clip_key`：按本方案生成
- `video_id`：成功后回填
- `source_name`：当前目录来源标识，可由 CLI 传入
- `status`：`running / succeeded / failed / skipped`
- `warning_codes`：脚本生成
- `error_code`：脚本生成
- `error_message`：脚本生成
- `context`：记录父文件名、transcript 文件名、clip_id 等上下文
- `started_at`
- `finished_at`
- `created_at`

---

## 6. 读取策略

脚本不读视频文件本体。

当前读取策略应为：

1. 扫描目录 A 下所有父视频描述文件
2. 扫描目录 B 下所有 transcript 文件，并预构造成“文件名集合 + 文件名到 Path 的索引”
3. 对每个父文件读取 `.clips`
4. 对每个 clip 生成 transcript 文件名
5. 先用目录 B 的文件名集合判断对应 transcript 是否存在
6. 若存在，再通过文件名索引拿到具体 Path 并读取 transcript JSON
7. 若 transcript 文件不存在，则直接记 `skipped`
8. 若存在，则进入校验和导入流程

此外还应增加两条收尾规则：

9. 记录所有未被任何父文件消费的 transcript 文件
10. 输出本次扫描汇总：父文件数、clip 数、成功匹配数、缺失 transcript 数、孤儿 transcript 数

这里的“缺失 transcript 数”就是被记为 `skipped` 的 clip 数。

---

## 7. 校验规则

当前方案下，必须补充以下特定校验。

### 7.1 父文件级校验

- 顶层必须有 `clips`
- `clips` 必须为数组
- 每个 clip 必须有：
  - `clip_id`
  - `buffered_start_time`
  - `buffered_end_time`
- 同一父文件内 `clip_id` 必须唯一
- `clip_id` 必须为正整数
- `buffered_start_time` 和 `buffered_end_time` 必须为整数毫秒值

### 7.2 transcript 文件匹配校验

- transcript 文件名必须能由父文件名和 `clip_id` 推导
- 每个父文件中的 clip，都应该能唯一匹配到一个 transcript 文件

### 7.3 时间一致性校验

- `buffered_end_time > buffered_start_time`
- transcript 中最小时间应不早于 `buffered_start_time` 太多
- transcript 中最大时间应不晚于 `buffered_end_time` 太多

这里建议把“太多”落成明确参数：

- `time_tolerance_ms`

默认可设为：

- `0`

也就是默认要求 transcript 时间轴完全落在 buffered 区间内；只有在后续遇到真实脏数据时，再通过 CLI 参数显式放宽。

当前样本里：

- transcript 覆盖区间落在 buffered 区间内

因此当前脚本应把“transcript 时间不在 buffered 区间内”视为异常数据。

### 7.4 transcript 结构校验

- 顶层必须有 `sentences`
- sentence `index / text / start / end` 必须存在
- token `index / text / start / end` 必须存在
- token 时间若超出 sentence 区间，不阻断入库，但必须记 warning
- `semanticElement` 不一定所有字段都非空，但结构应存在
- 同一视频内 `sentence.index` 必须唯一
- 同一句内 `token.index` 必须唯一
- `sentence.text` 不能为空字符串
- `token.text` 不能为空字符串

当前还建议明确：

- 空 transcript 直接视为失败，不写入空内容视频
- sentence 可以没有 token
- token 可以没有 `coarse_id`
- 当前已知上游 `AssemblyAI -> 1transcript-raw -> 2cleaned-data` 可能产生 `token.end > sentence.end` 的脏时间；该类问题当前固定记 `warning_code = token_time_outside_sentence`

### 7.5 coarse_unit 校验

- `coarse_id != null` 时，必须能在 `semantic.coarse_unit` 查到

实现上建议在脚本开始时批量加载一份 `semantic.coarse_unit.id` 集合到内存，避免逐 token 打数据库。

---

## 8. 幂等策略

幂等锚点统一使用：

- `source_clip_key`

不要使用：

- 文件完整路径
- transcript 占位路径
- 直接随机 UUID

因为这些都不适合作为稳定业务锚点。

### 8.1 replace 写入策略

当前脚本应明确采用“单 clip 单事务 replace 写入”：

1. 先按 `source_clip_key` 查 `catalog.videos`
2. 若不存在，则插入 `videos` 并获得 `video_id`
3. 若存在，则复用已有 `video_id`
4. 事务内删除该 `video_id` 旧的 transcript / sentence / span / unit_index 行
5. 重新写入本次计算出的新行
6. 更新 `catalog.videos` 的主字段
7. 写入一条新的 `video_ingestion_records`

这样可以保证：

- `video_id` 稳定
- 子表内容总是与当前导入输入一致
- 不需要做复杂的行级 diff

### 8.2 跳过策略

当前方案还应明确什么时候记 `skipped`。

建议规则为：

- 若 `source_clip_key` 已存在
- 且 `transcript_checksum` 未变化
- 且 `videos` 主表关键字段未变化

则本次导入可直接记为 `skipped`，不重复 replace 写入。

这里的关键字段至少包括：

- `parent_video_name`
- `parent_video_slug`
- `clip_seq`
- `source_start_ms`
- `source_end_ms`
- `duration_ms`
- `title`
- `clip_reason`
- `language`
- `hls_master_playlist_path`

---

## 9. 为什么当前方案可行

这个方案成立的关键原因有四个。

### 9.1 不读视频本体也能满足 Catalog 当前主事实

当前 `catalog` 最关键的是：

- 来源
- 时长
- HLS 路径占位
- transcript 路径占位
- transcript 结构化内容
- coarse unit 聚合索引

这些都可以由两类 JSON 推导出来。

### 9.2 父文件和 transcript 文件已经天然分层

父文件负责：

- clip 切片信息
- 来源信息

transcript 文件负责：

- sentence / token / semanticElement

这个分层和数据库设计是兼容的。

### 9.3 时间轴已经是毫秒绝对时间

这省掉了大量 clip-relative 到 parent-relative 的换算复杂度。

### 9.4 MVP 阶段允许对象路径占位

在真实桶路径尚未接入时，用明确前缀的占位路径即可，不会阻塞数据库入库和索引构建。

---

## 10. 当前明确不做的事

当前方案明确不做：

- 读取视频文件计算时长
- 校验视频文件是否真实存在
- 上传对象存储
- 生成缩略图
- 推导真实播放 URL
- 修正 transcript 内容
- 生成父视频实体

## 10.1 当前仍待确认的问题

这份方案当前只剩 1 个已知设计口子需要记录。

### 10.1.1 evidence 字段的表达力仍然偏弱

当前 schema 只能存 `evidence_sentence_indexes` 和 `evidence_span_indexes` 两个数组。
即使脚本里按位置配对解释，也不如结构化 evidence 对象清晰。
这不阻塞 MVP，但后续如果 evidence 要直接返回给上层接口，可能需要在主设计文档里进一步收紧或调整表示方式。

---

## 11. 推荐脚本落地顺序

建议按以下顺序实现：

1. `manifest_loader.py`
   支持目录 A + 目录 B 的文件匹配
2. `models.py`
   定义单 clip 输入对象
3. `validator.py`
   做父文件、transcript、coarse_id 校验
4. `normalizer.py`
   映射四张主内容表
5. `index_builder.py`
   构建 `video_unit_index`
6. `repository.py`
   事务写库
7. `main.py`
   串起来并支持 CLI

---

## 12. 结论

当前最合适的本地入库方案是：

**以父视频切片描述文件作为来源与切片边界输入，以 clip transcript 文件作为结构化内容输入，通过 `parent_name + clip_id -> transcript 文件名` 的规则完成匹配，不读取视频本体，直接构建 `catalog` 所需的内容资产、transcript 读模型、unit 索引和单视频入库审计。**

这套方案的前提已经被当前样本验证成立，可以直接作为下一步 Python 脚本实现依据。

---

## 13. 实现设计

本节把脚本实现层面的约束统一收口到 `scripts/catalog_ingest/README.md`。
后续 `scripts/catalog_ingest/` 下的 Python 文件都应以这里为准。

### 13.1 适用前提

当前脚本建立在以下前提上：

1. 数据库中只存切片视频，不存原始长视频实体。
2. 每个 clip 的业务唯一键都按固定规则生成：`source_clip_key = <parent_video_slug>#clip<clip_id>`。
3. 每个 clip 对应一个本地可读的 transcript JSON。
4. transcript JSON 中已经包含 sentence 和 span 级时间信息。
5. span 中的 `coarse_id` 已由上游流程产出。
6. 数据库 schema 已按 [docs/Catalog-数据库设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/Catalog-数据库设计.md) 建好。

因此脚本只负责导入、校验、标准化和写库，不负责生成内容。

### 13.2 脚本边界

脚本负责：

- 读取本地输入目录和 JSON 文件
- 组装单 clip 输入对象
- 校验输入元数据和 transcript 结构
- 归一化生成 `catalog` 所需行数据
- 聚合构建 `catalog.video_unit_index`
- 以单 clip 单事务方式写入：
  - `catalog.videos`
  - `catalog.video_transcripts`
  - `catalog.video_transcript_sentences`
  - `catalog.video_semantic_spans`
  - `catalog.video_unit_index`
- 写 `catalog.video_ingestion_records`

脚本不负责：

- 创建 `semantic.coarse_unit`
- 修复错误 transcript
- recommendation 审计
- `learning.*` 或 `recommendation.*` 写入
- 批量 job 管理
- 对象存储上传

### 13.3 总体执行模型

脚本采用单 clip 单事务 replace 写入。

每个 clip 的固定执行顺序是：

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

实现上当前还固定两条性能约束，但都不改变业务语义：

- transcript 目录先整体预扫描成“文件名集合 + 文件名到 Path 的索引”，避免逐 clip 反复访问文件系统
- 脚本启动后一次性批量加载本批 `source_clip_key` 对应的已有数据库状态，并复用同一个数据库连接；不要退回到“每个 clip 单独 connect + 单独 select”
- `skipped / failed` 审计记录允许在主循环中先收集，再按固定批次批量写入；`running / succeeded` 仍保留在单 clip 事务内原子完成
- 对于批量写入的 `skipped / failed`，`started_at / finished_at` 必须在“结果被判定”的那一刻生成，不能等到批量 flush 时再生成
- 主循环异常退出前，必须对内存中尚未落库的终态审计做最后一次 flush，避免把已经判定完成的 `skipped / failed` 静默丢掉

### 13.4 模块拆分

当前阶段目录职责固定如下：

```text
scripts/catalog_ingest/
  README.md
  __init__.py
  main.py
  models.py
  manifest_loader.py
  validator.py
  normalizer.py
  index_builder.py
  repository.py
  samples/
```

各文件边界如下：

- [main.py](/Users/evan/Downloads/learning-video-recommendation-system/scripts/catalog_ingest/main.py)
  只做 CLI 参数解析、数据库连接组装、总编排和退出码管理
- [models.py](/Users/evan/Downloads/learning-video-recommendation-system/scripts/catalog_ingest/models.py)
  只放脚本内部数据结构
- [manifest_loader.py](/Users/evan/Downloads/learning-video-recommendation-system/scripts/catalog_ingest/manifest_loader.py)
  只做输入目录扫描和单 clip 输入对象组装
- [validator.py](/Users/evan/Downloads/learning-video-recommendation-system/scripts/catalog_ingest/validator.py)
  只做输入校验和结构化错误输出
- [normalizer.py](/Users/evan/Downloads/learning-video-recommendation-system/scripts/catalog_ingest/normalizer.py)
  只做标准化行数据生成
- [index_builder.py](/Users/evan/Downloads/learning-video-recommendation-system/scripts/catalog_ingest/index_builder.py)
  只做 transcript 摘要和 `video_unit_index` 聚合
- [repository.py](/Users/evan/Downloads/learning-video-recommendation-system/scripts/catalog_ingest/repository.py)
  只做事务和 SQL 写入

不要把所有逻辑堆到 `main.py`。

### 13.5 输入模型

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

其中：

- `transcript_object_path` 是数据库中记录的对象路径
- `transcript_json` 是本地当前真正读取到的 transcript 原文对象
- `source_name` 和 `context` 用于写 `catalog.video_ingestion_records`

### 13.6 业务唯一键与数据库主键

入库脚本里必须明确区分：

- `source_clip_key`
- `video_id`

`source_clip_key` 是业务侧稳定唯一键，负责回答“这个外部 clip 是谁”。
`video_id` 是数据库内部主键，负责 `catalog.videos` 主键和其他子表外键关联。

当前正式规则如下：

```text
source_clip_key = <parent_video_slug>#clip<clip_id>
clip_seq = clip_id
```

### 13.7 数据库写入策略

单 clip 固定按以下顺序写入：

1. `catalog.videos`
2. `catalog.video_transcripts`
3. `catalog.video_transcript_sentences`
4. `catalog.video_semantic_spans`
5. `catalog.video_unit_index`
6. `catalog.video_ingestion_records`

`catalog.videos` 按 `source_clip_key` 做幂等 upsert。
若已存在，则更新主表字段并复用既有 `video_id`。

以下四张内容表采用 replace 策略：

- `catalog.video_transcripts`
- `catalog.video_transcript_sentences`
- `catalog.video_semantic_spans`
- `catalog.video_unit_index`

也就是：

- 先按 `video_id` 删除旧明细
- 再写入新明细

原因是 transcript 和 unit index 都由当前输入完整派生，不适合做局部 patch。

审计表采用执行历史模型：

- 开始时插入 `running`
- 成功后更新为 `succeeded`
- 失败后更新为 `failed`
- 无变化时记为 `skipped`
- 缺少对应 transcript 文件时也记为 `skipped`

### 13.8 幂等与跳过策略

幂等锚点固定使用：

- `source_clip_key`

无变化跳过条件为：

- `source_clip_key` 已存在
- `transcript_checksum` 未变化
- `hls_master_playlist_path` 未变化
- 主要元数据未变化

满足时：

- 不重写四张内容表
- 写一条 `video_ingestion_records`
- `status = skipped`

另外当前还固定一条跳过规则：

- 若按父文件名和 `clip_id` 推导出的 transcript 文件不存在，则不写 `catalog.videos` 和其余内容表，只写一条 `video_ingestion_records`
- 该记录的 `status = skipped`
- 建议 `error_code = transcript_missing`
- `context` 中记录父文件名、`clip_id` 和期望 transcript 文件名

### 13.9 错误模型

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

另外，非阻断性脏数据不记为 `failed`，而是记为 warning。

当前固定 warning 至少包括：

- `token_time_outside_sentence`

warning 的处理方式是：

- 继续执行并允许入库成功
- 将 warning code 写入 `catalog.video_ingestion_records.warning_codes`
- 将具体 sentence/token 位置和时间明细写入审计 `context`

### 13.10 CLI 设计建议

当前阶段 CLI 不需要复杂。

建议支持：

```text
--parents-dir <path>
--transcripts-dir <path>
--source-name <text>
--limit <n>
--dry-run
--clip-key <source_clip_key>
```

其中：

- `--parents-dir` 扫描父视频切片描述目录
- `--transcripts-dir` 扫描 clip transcript 目录
- `--clip-key` 用于单条重跑
- `--dry-run` 只校验和归一化，不写库

### 13.11 第一阶段开发范围

第一阶段只做 MVP 必需能力：

1. 读取两个输入目录
2. 按文件名规则完成父文件与 transcript 匹配
3. 校验 transcript 结构
4. 归一化四张内容表所需数据
5. 构建 `video_unit_index`
6. 单 clip 事务写库
7. 写 `video_ingestion_records`
8. 支持 `dry-run`

当前明确不做：

- 并发导入
- 批次级 job 管理
- 失败自动重试
- 对象存储上传
- 富日志平台集成
- 指标系统

### 13.12 后续扩展方向

如果第一阶段稳定，后续可继续加：

- 单元测试
- fixture transcript
- integration test
- 并发执行器
- 更细粒度的 evidence 生成
- 从本地文件切换到对象存储输入

但这些都应建立在当前单 clip 入库链路先稳定的前提上。
