# Recommendation

`internal/recommendation` 是 Recommendation 模块根目录。

当前它是 Recommendation 的模块外壳和子模块容器，不直接承载具体推荐实现。

## 1. 当前子模块

目前 Recommendation 下面已经落地的子模块只有一个：

1. `scheduler`

对应目录：

- [internal/recommendation/scheduler](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler)

这个子模块当前负责：

- 读取 Learning engine 的学习状态
- 生成推荐批次
- 维护 Recommendation 自己的 serving state
- 维护 Recommendation 自己的推荐审计

## 2. 为什么要下沉到 `scheduler/`

当前 Recommendation 在设计上并不只包含 scheduler。

按照整体设计，它后续至少还会包含：

1. `scheduler`
2. `recall`
3. `task`

所以把当前已实现的推荐逻辑整体下沉到：

- `internal/recommendation/scheduler`

目的是提前把 Recommendation 的模块根和子能力边界分开，避免后续在根目录继续堆积代码。

## 3. 当前目录结构

```text
internal/recommendation/
  README.md
  doc.go
  docs/
  scheduler/
    README.md
    doc.go
    application/
    domain/
    infrastructure/
    test/
```

## 4. 代码阅读顺序

如果你是第一次接手 Recommendation，建议按下面顺序读：

1. [internal/recommendation/README.md](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/README.md)
2. [推荐-模块代码实现详解.md](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/docs/推荐-模块代码实现详解.md)
3. [internal/recommendation/scheduler/README.md](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/README.md)
4. `scheduler/application/usecase`
5. `scheduler/domain/service`
6. `scheduler/infrastructure/persistence/query`
7. `scheduler/test/unit`
8. `scheduler/test/integration`
9. `scheduler/test/scenario`

## 4.1 docs 目录说明

`internal/recommendation/docs/` 当前用于放 Recommendation 模块的补充说明文档。

建议优先阅读：

- [推荐-模块代码实现详解.md](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/docs/推荐-模块代码实现详解.md)

它的定位不是设计稿，而是：

- 面向新人上手
- 对齐当前真实代码
- 解释文件结构、调用链、事务边界、数据流和测试入口

## 5. 当前边界

Recommendation 模块整体仍然遵守这些边界：

- 只读 `learning.*`
- 只写 `recommendation.*`
- 不回写 Learning engine
- 不维护学习事件和学习状态

## 6. 后续扩展要求

以后继续实现 Recommendation 时：

- scheduler 继续留在 `internal/recommendation/scheduler`
- recall 应新增到 `internal/recommendation/recall`
- task 应新增到 `internal/recommendation/task`

不要再把新的推荐能力直接堆回 `internal/recommendation` 根目录。
