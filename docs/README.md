# 文档导航

当前项目的唯一权威设计文档集合为以下四份：

1. [视频推荐系统总设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/视频推荐系统总设计.md)
2. [学习引擎设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/学习引擎设计.md)
3. [推荐模块设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/推荐模块设计.md)
4. [Catalog-数据库设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/Catalog-数据库设计.md)

除上述四份外，其余历史设计稿、讨论稿、对比稿均视为非权威参考材料，不可作为当前实现依据。

## 阅读顺序

如果是第一次接手项目，建议按下面顺序阅读：

1. [视频推荐系统总设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/视频推荐系统总设计.md)
2. [学习引擎设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/学习引擎设计.md)
3. [推荐模块设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/推荐模块设计.md)
4. [Catalog-数据库设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/Catalog-数据库设计.md)

## 文档说明

### [视频推荐系统总设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/视频推荐系统总设计.md)

系统总览文档。定义整体目标、三域边界、共享读模型和系统级实施顺序。

### [学习引擎设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/学习引擎设计.md)

Learning engine 的权威设计文档。定义学习事件、状态归约、Replay 与 Recommendation 读取契约。

### [推荐模块设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/推荐模块设计.md)

Recommendation 的权威设计文档。定义需求规划、多路候选生成、证据解析、排序选择和 Recommendation 自有读模型。

### [Catalog-数据库设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/Catalog-数据库设计.md)

Catalog 的权威最终设计文档。定义 `catalog` schema 的最终边界、数据模型、入库流程、轻量聚合逻辑与内容域读路径契约。

## 运行时现状参考

### [当前数据库Schema现状.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/当前数据库Schema现状.md)

基于 `.env` 指向的 live DB 做只读探查得到的现状快照。它描述当前实例里实际存在的业务 schema、`public` 中的遗留对象、真实表结构、索引、约束和与最新版设计的明显差异。

### [当前实现现状.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/当前实现现状.md)

基于当前仓库代码整理的实现快照。它描述当前代码已经实现到哪一层、哪些基础设施已经落地、哪些模块仍然只是骨架，以及仓库现状与 live DB 现状之间的区别。

## 历史归档

### [archive/README.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/archive/README.md)

已废弃历史设计稿与过渡文档的索引页。若需要追溯旧结构、旧 Recommendation 假设或 Catalog 改造过程，请从这里进入，不要直接把归档文档当作当前实现依据。
