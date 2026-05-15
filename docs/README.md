# 文档导航

当前项目的唯一权威设计文档集合为以下四份：

1. [视频推荐系统总设计.md](视频推荐系统总设计.md)
2. [学习引擎设计.md](学习引擎设计.md)
3. [推荐模块设计.md](推荐模块设计.md)
4. [Catalog-数据库设计.md](Catalog-数据库设计.md)

除上述四份外，其余历史设计稿、讨论稿、对比稿均视为非权威参考材料，不可作为当前实现依据。

## 阅读顺序

如果是第一次接手项目，建议按下面顺序阅读：

1. [视频推荐系统总设计.md](视频推荐系统总设计.md)
2. [学习引擎设计.md](学习引擎设计.md)
3. [推荐模块设计.md](推荐模块设计.md)
4. [Catalog-数据库设计.md](Catalog-数据库设计.md)

## 文档说明

### [视频推荐系统总设计.md](视频推荐系统总设计.md)

系统总览文档。定义整体目标、三域边界、共享读模型和系统级实施顺序。

### [学习引擎设计.md](学习引擎设计.md)

Learning engine 的权威设计文档。定义学习事件、状态归约、Replay 与 Recommendation 读取契约。

### [推荐模块设计.md](推荐模块设计.md)

Recommendation 的权威设计文档。定义需求规划、多路候选生成、证据解析、排序选择和 Recommendation 自有读模型。

### [Catalog-数据库设计.md](Catalog-数据库设计.md)

Catalog 的权威最终设计文档。定义 `catalog` schema 的最终边界、数据模型、入库流程、轻量聚合逻辑与内容域读路径契约。

## 专题设计草案

### [练习题绑定与生成设计.md](练习题绑定与生成设计.md)

练习题绑定与生成的 MVP 语义设计草案。它按生成方案、触发方案、题目形态三个模块说明 per 单元题与 per 视频 * 单元题、视频末尾小测、Feed 复习卡、学习模式练习和 AI 生成题型边界；暂不定稿数据库表结构或具体 API。

### [学习互动信号语义设计.md](学习互动信号语义设计.md)

视频学习互动信号的 MVP 语义设计草案。它从 Recommendation 返回的 `learning_units` 出发，定义字幕曝光、字幕 lookup、lookup 弹窗附加行为、习题答题、analytics raw interaction log、learning evidence 与后续 reducer 之间的边界；暂不定稿具体 API、存储或代码模块。

### [学习互动信号架构图.md](学习互动信号架构图.md)

学习互动信号链路的 MVP 架构图草案。它用简化图和详细设计图说明前端互动收集、后端接入、`analytics.learning_interaction_events`、`analytics.quiz_events`、标准化分流、Practice attempt、learning evidence 与 Learning engine 之间的组件关系。

### [学习引擎Normalizer设计.md](学习引擎Normalizer设计.md)

Learning Engine normalizer 的 MVP 设计草案。它定义 `internal/learningengine/normalizer` 子模块如何 read-only 读取 `analytics.*` raw fact，把 quiz、lookup、exposure 和 self mark 解释成 Learning Engine normalized event，并在暂不增加 checkpoint 的前提下依赖 source 幂等约束与 anti-join 查询完成可重试归一化。

## 运行时现状参考

### [当前数据库Schema现状.md](当前数据库Schema现状.md)

基于 `.env` 指向的 live DB 做只读探查得到的现状快照。它描述当前实例里实际存在的业务 schema、`public` 中的遗留对象、真实表结构、索引、约束和与最新版设计的明显差异。

### [当前实现现状.md](当前实现现状.md)

基于当前仓库代码整理的实现快照。它描述当前代码已经实现到哪一层、哪些基础设施已经落地、哪些模块仍然只是骨架，以及仓库现状与 live DB 现状之间的区别。

## 历史归档

### [archive/README.md](archive/README.md)

已废弃历史设计稿与过渡文档的索引页。若需要追溯旧结构、旧 Recommendation 假设或 Catalog 改造过程，请从这里进入，不要直接把归档文档当作当前实现依据。
