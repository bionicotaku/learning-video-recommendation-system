# 文档导航

当前项目的唯一权威设计文档集合为以下四份：

1. [全新设计-总设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/全新设计-总设计.md)
2. [全新设计-学习引擎设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/全新设计-学习引擎设计.md)
3. [全新设计-推荐模块设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/全新设计-推荐模块设计.md)
4. [全新设计-Catalog-数据库设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/全新设计-Catalog-数据库设计.md)

除上述四份外，其余历史设计稿、讨论稿、对比稿均视为非权威参考材料，不可作为当前实现依据。

## 阅读顺序

如果是第一次接手项目，建议按下面顺序阅读：

1. [全新设计-总设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/全新设计-总设计.md)
2. [全新设计-学习引擎设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/全新设计-学习引擎设计.md)
3. [全新设计-推荐模块设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/全新设计-推荐模块设计.md)
4. [全新设计-Catalog-数据库设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/全新设计-Catalog-数据库设计.md)

## 文档说明

### [全新设计-总设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/全新设计-总设计.md)

系统总览文档。定义整体目标、三域边界、共享读模型和系统级实施顺序。

### [全新设计-学习引擎设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/全新设计-学习引擎设计.md)

Learning engine 的权威设计文档。定义学习事件、状态归约、Replay 与 Recommendation 读取契约。

### [全新设计-推荐模块设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/全新设计-推荐模块设计.md)

Recommendation 的权威设计文档。定义需求规划、多路候选生成、证据解析、排序选择和 Recommendation 自有读模型。

### [全新设计-Catalog-数据库设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/全新设计-Catalog-数据库设计.md)

Catalog 的权威最终设计文档。定义 `catalog` schema 的最终边界、数据模型、入库流程、轻量聚合逻辑与内容域读路径契约。
