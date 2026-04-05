# 文档导航

当前 `docs/` 目录只保留现行文档。

## 阅读顺序

如果是第一次接手项目，建议按下面顺序阅读：

1. [推荐系统MVP整体设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/推荐系统MVP整体设计.md)
2. [模块统一文件结构规范.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/模块统一文件结构规范.md)
3. [学习引擎设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/学习引擎设计.md)
4. [学习引擎工程实现.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/学习引擎工程实现.md)
5. [推荐-学习调度模块设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/推荐-学习调度模块设计.md)
6. [推荐-学习调度模块工程实现.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/推荐-学习调度模块工程实现.md)
7. [数据表说明.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/数据表说明.md)
8. [推荐-视频召回模块设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/推荐-视频召回模块设计.md)

## 文档说明

### [推荐系统MVP整体设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/推荐系统MVP整体设计.md)

总览文档。说明整个 MVP 的模块边界、调用关系和系统范围。

### [模块统一文件结构规范.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/模块统一文件结构规范.md)

统一说明 `internal/` 下顶层模块和子模块应遵守的文件结构标准，以及哪些目录按需启用、哪些职责必须有固定落点。

### [学习引擎设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/学习引擎设计.md)

Learning engine 的设计文档。说明：

- `learning.unit_learning_events`
- `learning.user_unit_states`
- 学习事件模型
- 状态归约
- full replay

### [学习引擎工程实现.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/学习引擎工程实现.md)

Learning engine 的工程实现说明。重点是目录结构、分层职责、reducer、repository、SQL、migration 和测试验收。

### [推荐-学习调度模块设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/推荐-学习调度模块设计.md)

Recommendation 模块设计文档。说明：

- 如何读取 Learning engine 状态
- 如何生成推荐批次
- Recommendation 自己维护哪些表

### [推荐-学习调度模块工程实现.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/推荐-学习调度模块工程实现.md)

Recommendation 的工程实现说明。重点是目录结构、分层职责、repository 接口、SQL 设计和工程约束。

### [数据表说明.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/数据表说明.md)

全局数据库对象说明文档。

### [推荐-视频召回模块设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/推荐-视频召回模块设计.md)

视频召回层专项设计文档，属于 Recommendation 内部后续能力设计。

## 当前结构

当前系统按两个平级模块组织：

1. `Learning engine`
2. `Recommendation`

当前 MVP 的关键约束是：

- Learning engine 维护学习事件和学习状态
- Recommendation 只读取 Learning engine 业务表，不回写 Learning engine
- Recommendation 维护自己的 serving state 和推荐审计
- MVP 不支持用户级 Recommendation 调度配置
- Learning engine 的 replay 只支持 full replay

## 文档维护原则

- `docs/` 主目录只保留现行文档
- 过程型任务清单、重构基线、阶段性验收报告不再保留在主目录
- 新文档应优先描述当前结构，不重复保留已废弃路径和历史迁移过程
