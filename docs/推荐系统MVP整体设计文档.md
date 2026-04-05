# learning-video-recommendation-system

# 基于视频内容的学习推荐系统 MVP 整体设计文档

## 1. 文档目的

本文档描述当前 MVP 阶段的**整体系统设计**。

系统的核心目标是：

> 围绕用户当前学习目标，结合学习状态与视频内容，生成适合当前阶段的学习任务。

这套系统不是传统信息流推荐系统，也不是纯粹的单词记忆工具，而是一个面向“**通过视频学习学习内容**”场景的学习系统。

这里的“学习内容”统一落在：

- `semantic.coarse_unit`

之上，当前至少支持：

- `word`
- `phrase`
- `grammar`

本文档重点解决 5 个问题：

1. 系统整体应该拆成哪些模块
2. Learning engine 和 Recommendation 的边界是什么
3. Recommendation 内部三层能力如何配合
4. 系统对外最终返回什么
5. 当前 MVP 的范围和后续文档关系是什么

## 2. 产品目标

### 2.1 核心目标

系统希望帮助用户在视频语境中高效学习学习内容，并同时兼顾：

- 记忆保留
- 新内容推进
- 学习负担控制
- 内容消费体验
- 推荐结果可解释性
- 系统行为可调试性

### 2.2 MVP 阶段的设计原则

#### 1. 先保证学习闭环成立

优先打通：

- 学习目标建立
- 学习状态维护
- 推荐任务生成
- 学习反馈回流

而不是一开始追求复杂模型。

#### 2. 先做可控系统，不做黑盒系统

当前重点是构建一个规则清晰、行为可解释、易于调试的系统，而不是直接引入复杂机器学习模型。

#### 3. 先围绕统一学习单元建模

MVP 阶段的稳定抽象是：

- `semantic.coarse_unit`

而不是只服务单词学习的专用模型。

#### 4. 最终输出应是学习任务，而不是单纯视频列表

系统推荐的终点不是：

- “你可能想看什么视频”

而是：

- “你现在适合完成什么学习任务”

#### 5. Recommendation 应作为完整后端模块对外提供能力

对上层业务 API 来说，Recommendation 是一个整体模块。

上层不分别调用：

- 学习内容调度器
- 视频召回层
- 学习任务层

而是只调用 Recommendation 模块，获取最终学习任务结果。

#### 6. MVP 不支持用户级调度配置

MVP 阶段明确不支持按用户持久化 Recommendation 调度配置。

当前 Recommendation 统一使用模块默认值来决定：

- session limit 默认值
- daily new quota
- daily review soft / hard limit
- timezone 相关日界线

这样做的目的，是先把跨模块边界、状态读取、推荐生成和审计闭环做稳定，而不是在 MVP 阶段引入额外的用户级配置表和配置管理链路。

后续如果需要扩展，这类配置仍应归 Recommendation 自己维护，不进入 Learning engine。

## 3. 整体系统定位

从最终系统边界看，当前 MVP 应拆成两个平级模块：

1. `Learning engine`
2. `Recommendation`

### 3.1 Learning engine

回答：

- 用户当前对哪些目标学习单元处于什么状态
- 下一次什么时候该复习
- 当前掌握程度如何

它负责：

- 记录学习事件
- 维护学习状态
- replay 重建状态

### 3.2 Recommendation

回答：

- 在当前时刻，应该给用户推什么学习任务

它负责：

- 读取 Learning engine 的学习状态
- 读取视频内容
- 生成最终学习任务
- 维护自己的推荐投放状态和推荐审计

### 3.3 Recommendation 内部三层

Recommendation 内部仍然拆成三层能力：

1. 学习内容调度器
2. 视频召回层
3. 学习任务层

但这三层仅用于 Recommendation 内部组织，不构成对上层暴露的独立服务。

## 4. 系统边界与调用关系

## 4.1 外部边界

上层业务 API 对两个模块的调用关系应是：

- 学习反馈写回时调用 Learning engine
- 生成推荐结果时调用 Recommendation

前端不直接调用这两个模块。

## 4.2 内部边界

### Learning engine 对外暴露

- 记录标准化学习事件
- 重建学习状态
- 读取用户学习状态

### Recommendation 对外暴露

- 生成当前学习任务推荐结果

### Recommendation 内部包含

- 学习内容调度器
- 视频召回层
- 学习任务层

## 4.3 正确的调用关系

### 学习反馈回流链路

```text
前端行为
  ↓
[上层业务 API / 事件结算服务]
  ↓
[Learning engine]
  ├── 写 learning.unit_learning_events
  └── 写 learning.user_unit_states
```

### 推荐生成链路

```text
前端
  ↓
[上层业务 API]
  ↓
[Recommendation]
  ├── 读取 learning.user_unit_states
  ├── 学习内容调度器
  ├── 视频召回层
  ├── 学习任务层
  ├── 写 recommendation.user_unit_serving_states
  └── 写 recommendation.scheduler_runs / items
  ↓
最终学习任务结果
  ↓
[上层业务 API 返回前端]
```

## 4.4 为什么要这样设计

这样设计有 5 个直接好处：

1. Learning engine 与 Recommendation 的 owner 清晰
2. Recommendation 不再污染学习状态表
3. replay 只重建学习域，不误伤推荐投放状态
4. Recommendation 内部三层仍可独立迭代
5. 上层业务 API 不需要感知 Recommendation 内部流水线

## 5. 整体系统架构

当前 MVP 建议由 4 份文档共同描述：

1. **整体设计文档**
2. **学习引擎设计文档**
3. **学习内容推荐模块设计文档**
4. **学习内容推荐模块工程实现稿**

其中本文档是总览文档，负责描述跨模块边界和总架构。

### 5.1 模块级视角

```text
用户学习行为 + 视频内容索引
        ├──> [Learning engine]
        │       ├── learning.unit_learning_events
        │       └── learning.user_unit_states
        │
        └──> [Recommendation]
                ├── 读取 learning.user_unit_states
                ├── 学习内容调度器
                ├── 视频召回层
                ├── 学习任务层
                ├── recommendation.user_unit_serving_states
                └── recommendation.scheduler_runs / items
```

### 5.2 Recommendation 内部数据流

```text
learning.user_unit_states + semantic.coarse_unit + 视频内容索引
                ↓
          [学习内容调度器]
                ↓
      输出：目标学习内容列表
                ↓
            [视频召回层]
                ↓
        输出：候选视频列表
                ↓
            [学习任务层]
                ↓
      输出：最终学习任务列表
```

### 5.3 核心判断

从内部数据流上看，系统不是“直接推荐视频”，而是：

> 先生成学习目标，再为学习目标找到最适合的视频内容，最后包装成学习任务。

## 6. 核心概念定义

## 6.1 目标学习内容

指当前应该进入学习范围的学习单元，底层统一使用：

- `semantic.coarse_unit`

每个目标学习内容当前可以带有：

- `is_target`
- `target_source`
- `target_source_ref_id`
- `target_priority`

这些信息当前保存在 `learning.user_unit_states` 中，由 Learning engine 维护。

## 6.2 学习事件

指用户一次真实学习行为的标准化记录，例如：

- 学习了哪个学习单元
- 学习时间
- 学习类型
- 是否答对
- 本次质量评分
- 来源任务或来源视频

它由 Learning engine 维护，落在：

- `learning.unit_learning_events`

## 6.3 学习状态

指系统对每个“用户-学习单元”当前状态的聚合结果，例如：

- 当前属于 `new / learning / reviewing / mastered / suspended`
- 当前进度
- 当前掌握分
- 下次复习时间
- 最近表现是否稳定

它由 Learning engine 维护，落在：

- `learning.user_unit_states`

## 6.4 推荐投放状态

指 Recommendation 为了避免重复推荐、控制冷却时间而维护的投放信息。

当前 MVP 只明确需要：

- `last_recommended_at`

它不属于 Learning engine，不在学习状态表中，而应落在：

- `recommendation.user_unit_serving_states`

## 6.5 目标学习内容列表

指由 Recommendation 内部的学习内容调度器输出的、本轮应优先学习的一组学习内容。

它不是最终用户看到的结果，而是 Recommendation 内部中间层结果。

## 6.6 候选视频列表

指视频召回层基于目标学习内容集合召回的一批视频内容。

当前视频召回的基本单位应是：

- 视频片段

而不是长视频整段。

## 6.7 学习任务

学习任务是 Recommendation 模块最终输出给上层业务 API 的结果实体。

它不是纯视频实体，而是一条带有学习语义的任务记录，例如：

- 推荐学习哪个视频
- 本视频覆盖哪些重点学习内容
- 哪些是复习内容
- 哪些是新内容
- 为什么推荐这条任务

## 7. 模块职责划分

## 7.1 Learning engine

### 目标

维护用户对目标学习单元的稳定学习状态。

### 主要职责

- 维护 `learning.unit_learning_events`
- 维护 `learning.user_unit_states`
- 接收并处理标准化学习事件
- 根据学习历史更新状态
- 提供 full replay

### 输出

- 用户学习状态
- 下一次复习时间
- 掌握程度
- 对 Recommendation 稳定可读的状态输入

### 不负责

- 不生成推荐批次
- 不处理视频召回
- 不生成最终学习任务
- 不维护 `last_recommended_at`

## 7.2 Recommendation 内部的学习内容调度器

### 目标

从用户全部目标学习内容中，筛选出当前最值得学习的一组目标内容。

### 主要职责

- 读取 `learning.user_unit_states`
- 根据复习节奏决定当前 due review
- 根据配额规则控制 new 引入
- 计算优先级和原因码
- 输出本轮目标学习内容列表

### 输出

一个带有推荐类型、优先级、权重和原因码的目标学习内容列表。

### 不负责

- 不写学习状态
- 不处理学习事件
- 不做 replay

## 7.3 视频召回层

### 目标

基于目标学习内容列表，召回适合承载这些学习目标的视频片段。

### 主要职责

- 使用 transcript 与学习内容映射关系建立检索基础
- 根据目标学习内容覆盖情况召回候选视频
- 评估视频对当前学习目标的承载能力
- 输出候选视频列表给学习任务层

### 输出

候选视频列表，以及每个候选视频对目标学习内容的覆盖关系与匹配信号。

### 不负责

- 不决定用户当前该学什么
- 不维护学习状态
- 不直接面向上层业务 API 暴露接口

## 7.4 学习任务层

### 目标

将目标学习内容和候选视频组织成最终学习任务。

### 主要职责

- 将“视频内容”与“学习目标”绑定
- 生成任务级解释信息
- 控制最终输出结果的多样性与冗余
- 构造最终学习任务实体

### 输出

最终学习任务列表。

### 不负责

- 不负责底层复习状态维护
- 不负责底层 transcript 解析
- 不负责学习事件处理

## 8. 核心数据流

## 8.1 学习反馈写入阶段

Learning engine 接收上层业务 API 标准化后的学习事件，包括：

- `exposure`
- `lookup`
- `new_learn`
- `review`
- `quiz`

然后：

- 追加写入 `learning.unit_learning_events`
- 归约更新 `learning.user_unit_states`

## 8.2 Recommendation 输入阶段

Recommendation 接收上层业务 API 提供的推荐请求上下文，包括：

- 用户标识
- 当前 session 限制
- 当前请求上下文

并自行读取：

- `learning.user_unit_states`
- `semantic.coarse_unit`
- 视频内容索引
- `recommendation.user_unit_serving_states`

## 8.3 学习目标生成阶段

Recommendation 内部的学习内容调度器根据当前状态输出一组当前最适合学习的目标学习内容。

这些目标内容通常分为两类：

- 复习内容
- 新内容

同时附带：

- 权重
- 推荐原因
- 调度属性

## 8.4 视频召回阶段

视频召回层接收目标学习内容列表，然后基于视频 transcript 与学习内容映射，找出：

- 覆盖这些目标内容的视频
- 对当前学习目标匹配度更高的视频
- 信息负担相对合适的视频
- 语境质量更清晰的视频

## 8.5 学习任务生成阶段

学习任务层将候选视频与目标学习内容重新组织，构造出最终学习任务。

每条任务至少包含：

- 视频实体
- 覆盖的目标学习内容
- 重点复习内容 / 新内容
- 推荐原因

## 8.6 Recommendation 审计与 serving state 更新

Recommendation 生成结果后，应维护自己的表：

- `recommendation.scheduler_runs`
- `recommendation.scheduler_run_items`
- `recommendation.user_unit_serving_states`

其中：

- `last_recommended_at` 应在这里更新
- 不应回写到 `learning.user_unit_states`

## 9. 学习反馈与状态回流

## 9.1 前端行为不直接写入 Learning engine

前端产生的原始行为，例如：

- 看完一个视频
- 点开某个学习内容解释
- 完成一次回忆题
- 完成一次测验
- 结束一轮学习任务

都不直接进入 Learning engine。

## 9.2 上层业务 API 负责行为标准化

上层业务 API / 学习事件结算服务负责把前端原始行为转换成标准化学习事件。

## 9.3 Learning engine 负责状态更新

Learning engine 接收这些标准化学习事件，更新 `learning.user_unit_states`，并影响下一轮 Recommendation 输入。

## 9.4 Recommendation 不回写学习状态

这是当前新的硬边界：

- Recommendation 不写 `learning.unit_learning_events`
- Recommendation 不写 `learning.user_unit_states`

Recommendation 只能写自己的表。

## 10. MVP 的核心推荐逻辑

### 10.1 推荐的真实终点是学习任务

虽然 Recommendation 内部存在“学习内容调度”和“视频召回”，但从产品视角看，最终推荐对象应是学习任务，而不是单独的学习内容项，也不是纯视频内容。

### 10.2 学习内容列表只是中间表示层

学习内容列表的作用是作为：

- Recommendation 内部调度层的输出
- 视频召回层的输入
- 学习任务层的学习目标依据

它不是对外主结果。

### 10.3 视频内容是学习载体，不是推荐目标本身

视频在这里不是纯娱乐内容，而是学习媒介。

因此视频召回不能只考虑“用户想看什么”，还必须考虑：

- 是否覆盖当前目标学习内容
- 是否适合当前学习负担
- 是否有合理学习收益

### 10.4 Recommendation 对外返回的是任务结果，不是内部中间结果

Recommendation 对上层业务 API 的主返回结果应是最终学习任务。

内部中间结果如：

- 目标学习内容列表
- 候选视频列表
- 中间分数
- 原因码

主要用于模块内部流转、调试与解释。

## 11. 当前数据库映射

## 11.1 `auth.users`

用户主体表。

## 11.2 `catalog.videos`

视频主表，是视频内容实体。

## 11.3 `catalog.video_user_states`

用户对视频互动状态的聚合表，未来可辅助 Recommendation 做去重和重复曝光控制。

## 11.4 `semantic.coarse_unit`

统一学习内容主表。

它对整体设计的直接影响是：

- 学习目标不应再写死为单词
- Recommendation 的调度和召回都应统一基于 `coarse_unit`

## 11.5 当前表 owner 划分

### Learning engine 持有

- `learning.unit_learning_events`
- `learning.user_unit_states`

### Recommendation 持有

- `recommendation.user_unit_serving_states`
- `recommendation.scheduler_runs`
- `recommendation.scheduler_run_items`

## 12. MVP 范围定义

### 12.1 当前包含的能力

#### Learning engine 侧

- 维护学习事件真相层
- 维护学习状态总表
- 支持 full replay

#### Recommendation 侧

- 从状态总表中生成目标学习内容列表
- 基于目标学习内容召回候选视频
- 将候选视频与目标学习内容组织成学习任务
- 输出最终学习任务列表
- 维护推荐审计和推荐投放状态

#### 闭环侧

- 能接收标准化学习事件
- 能根据学习反馈更新学习状态
- 能影响下一轮 Recommendation 结果

### 12.2 当前不包含的能力

- 复杂个性化深度学习排序
- 重度兴趣建模
- 跨模态 embedding 召回
- 复杂 session 多轮交互编排
- 增量 replay
- Recommendation 之外的复杂投放实验系统

## 13. 系统设计上的关键判断

### 13.1 为什么先以学习内容为核心，而不是直接以视频为核心

因为学习产品首先需要确定“学什么”，再决定“通过什么内容学”。

### 13.2 为什么必须先拆出 Learning engine

如果 Learning engine 与 Recommendation 不拆开，就会出现：

- 推荐模块直接维护学习状态
- replay 污染推荐逻辑
- `last_recommended_at` 混入学习状态表
- owner 不清晰

把 Learning engine 先拆出来，是当前系统边界正确化的前提。

### 13.3 为什么 Recommendation 仍然要作为一个整体模块对外提供能力

如果上层业务 API 需要分别调用：

- 调度器
- 视频召回层
- 学习任务层

就会把 Recommendation 的内部组织暴露到外部边界上，造成强耦合。

### 13.4 为什么学习任务层必须存在

如果只有“学习内容调度 + 视频召回”，系统很容易停留在“给用户一个视频列表”。

但教育产品需要的是任务表达，而不是纯内容流表达。

### 13.5 为什么 `last_recommended_at` 必须属于 Recommendation

因为它表示的是：

- 最近一次被推荐系统投放是什么时候

这不是学习事件，也不是学习状态，而是 Recommendation 的投放状态。

## 14. 潜在风险与设计注意点

### 14.1 不要把“视频包含目标学习内容”误当成“视频适合学习目标”

覆盖关系不是最终适配关系。

### 14.2 不要把“学习内容列表”误当成最终产品结果

它只是 Recommendation 内部中间层。

### 14.3 不要过早做复杂推荐模型

当前阶段先把结构和闭环做对。

### 14.4 不要让 Recommendation 回写 Learning engine

这是当前新的硬性边界。

### 14.5 不要把 Recommendation 的投放状态塞回学习状态表

例如：

- `last_recommended_at`

必须属于 Recommendation 自己的 serving state。

## 15. 后续文档关系

### 15.1 Learning engine 文档

- [学习引擎设计文档](/Users/evan/Downloads/learning-video-recommendation-system/docs/学习引擎设计文档.md)

包含：

- 学习事件
- 学习状态
- SM-2
- 状态迁移
- replay
- Learning engine 工程边界

### 15.2 Recommendation 设计文档

- [学习调度系统设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/学习调度系统设计.md)

包含：

- Recommendation 模块边界
- 调度、召回、任务三层
- 推荐投放状态
- Recommendation 审计表

### 15.3 Recommendation 工程实现稿

- [学习调度系统工程实现稿.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/学习调度系统工程实现稿.md)

包含：

- Recommendation 工程分层
- repository / sqlc / migration
- usecase 设计
- Recommendation 测试与验收

## 16. 总结

当前 MVP 的整体系统边界可以概括成一句话：

> Learning engine 负责维护“用户对目标学习单元学得怎么样”，Recommendation 负责基于这些状态与视频内容生成“现在该完成什么学习任务”，并维护 Recommendation 自己的投放状态与审计数据。

在这个最终版本里：

- `learning.unit_learning_events` 和 `learning.user_unit_states` 属于 Learning engine
- Recommendation 只读取 Learning engine 输出
- Recommendation 内部仍然保留“学习内容调度器 -> 视频召回层 -> 学习任务层”的三层流水线
- 系统最终对外返回的是学习任务，而不是内部中间结果
