# API 设计文档索引

本目录统一收口已实现和后续实现时需要遵守的 API 设计文档。

重要状态说明：

- **API 基座已落地。** 当前仓库已有 `internal/api` 目录、HTTP server bootstrap、router、middleware、handler、API DTO mapper 和 API 层测试。
- **学习事件上报 API 已实现写入入口。** 当前已包含 learning interaction batch、quiz attempt、self mark mastered 三条写入 endpoint。
- **移动端 MVP 不实现 CORS。** 当前入口面向原生客户端；如未来增加 Web 前端，再单独增加 CORS middleware 与 allowlist 配置。
- **Feed、Video Detail、Video Favorites、Video History、End Quiz、Catalog watch-progress、Video Interactions、Unit Progress、Unit Collections、Active Learning Targets、Me API、User Feedback API 已落地。** Video Favorites / Video History 提供 Catalog 只读 keyset 分页列表；Unit Progress 提供 mastered / unmastered 两个分页读取 endpoint；Me API 提供 profile、累计活动统计和 7 天 activity calendar；User Feedback API 提供 5 MiB multipart 反馈上传。
- **认证 principal adapter 已支持 GCP API Gateway userinfo。** 后端仍不自行验证 JWT 签名；生产由 Gateway 验证 JWT，后端解析 `X-Apigateway-Api-Userinfo`。

## 总体规范

- [API模块总体设计规范.md](API模块总体设计规范.md)：`internal/api` 的统一入口规范。后续所有 API endpoint group 都必须遵守。

## 业务 API 设计

- [学习事件上报API设计.md](学习事件上报API设计.md)：learning interactions batch 与 quiz attempt 单点上报。
- [Unit-Progress-API-MVP设计.md](Unit-Progress-API-MVP设计.md)：用户学习单元进度分页读取。
- [Catalog-观看进度上报MVP设计.md](Catalog-观看进度上报MVP设计.md)：视频观看进度上报。
- [Video-Interactions-API-MVP设计.md](Video-Interactions-API-MVP设计.md)：视频点赞/取消点赞、收藏/取消收藏。
- [Feed-API-MVP设计.md](Feed-API-MVP设计.md)：feed 页面获取推荐视频列表的前端展示契约。
- [Video-Detail-API-MVP设计.md](Video-Detail-API-MVP设计.md)：fullscreen 播放页读取单个视频详情、播放资源、transcript URL、互动计数和当前用户状态。
- [Video-Favorites-API-MVP设计.md](Video-Favorites-API-MVP设计.md)：当前用户视频收藏列表分页读取。
- [Video-History-API-MVP设计.md](Video-History-API-MVP设计.md)：当前用户视频观看历史列表分页读取。
- [End-Quiz-批量取题API-MVP设计.md](End-Quiz-批量取题API-MVP设计.md)：视频末尾按 `video_id + coarse_unit_ids` 批量取 quiz 题。
- [Unit-Collections-API-MVP设计.md](Unit-Collections-API-MVP设计.md)：词书列表读取，以及学习目标激活接口的业务语义说明。
- [Active-Learning-Targets-API-MVP设计.md](Active-Learning-Targets-API-MVP设计.md)：当前用户 active learning target coarse unit id 列表读取。
- [Me-API-MVP设计.md](Me-API-MVP设计.md)：当前用户 profile 读取、累计活动统计、activity calendar、profile lazy repair 和 timezone 顺手更新。
- [User-Feedback-API-MVP设计.md](User-Feedback-API-MVP设计.md)：当前用户反馈上传，支持一个自定义 JSON payload 和最多 5 张 JPEG 图片。

具体业务 API 文档只定义 endpoint 字段、业务语义、成功边界和前端样例；通用认证、错误 envelope、状态码、handler 结构和测试要求统一看总体规范。

## 已实现 API 总表

当前 `internal/api` 已实现 20 个业务 HTTP endpoint。实现口径以 handler route registration 为准：

| Method | Path | 业务分组 | 主要 owner / 编排 | 成功边界 |
|---|---|---|---|---|
| `POST` | `/api/feed` | Feed / 推荐流 | API facade 编排 Recommendation、Catalog、Semantic | Recommendation 生成 plan 并写 audit / serving state，API facade 批量补齐列表 preview 字段与 learning unit 展示文本后返回 feed。 |
| `GET` | `/api/videos/{video_id}` | Video Detail / 视频详情 | Catalog | 读取单个可展示视频的播放资源、transcript URL、description、互动计数和当前用户 like / favorite 状态；不写任何状态。 |
| `GET` | `/api/video-favorites` | Video Favorites / 视频收藏列表 | Catalog | Keyset 分页读取当前用户仍收藏且仍可展示的视频列表；只返回 preview 和 `favorited_at`。 |
| `GET` | `/api/video-history` | Video History / 视频观看历史 | Catalog | Keyset 分页读取当前用户最近观看且仍可展示的视频列表；只返回 preview、`last_position_ms` 和 `last_watched_at`。 |
| `POST` | `/api/videos/end-quiz` | End Quiz / 视频末尾取题 | Catalog | 按 `video_id + coarse_unit_ids` 只读获取 quiz 候选；不写 quiz delivery、学习进度或统计。 |
| `GET` | `/api/me` | Me / 当前用户 | User | 返回 profile、累计 stats、内嵌 7 天 activity calendar；必要时 lazy repair profile，并可用合法 timezone 更新 profile。 |
| `GET` | `/api/unit-collections` | Unit Collections / 词书列表 | API facade 编排 Semantic 与 Learning Engine | 读取 active 词书集合列表，并返回当前用户 `active_collection` slug / null。 |
| `GET` | `/api/learning-targets/active-coarse-unit-ids` | Active Learning Targets / 学习目标读取 | Learning Engine reducer read model | 读取当前用户 `is_target=true AND status!='mastered'` 的 coarse unit ids，用于 fullscreen exposure 过滤。 |
| `PUT` | `/api/learning-targets/active-collection` | Learning Targets / 学习目标写入 | API facade 同事务编排 Learning Engine 与 User | 同事务切换当前 active collection target projection，并把 onboarding 状态更新为 `collection_selected`。 |
| `PUT` | `/api/videos/{video_id}/like` | Video Interactions / 视频互动 | Catalog | 幂等设置当前用户已点赞，并返回点赞状态和 `like_count`。 |
| `DELETE` | `/api/videos/{video_id}/like` | Video Interactions / 视频互动 | Catalog | 幂等取消当前用户点赞，并返回点赞状态和 `like_count`。 |
| `PUT` | `/api/videos/{video_id}/favorite` | Video Interactions / 视频互动 | Catalog | 幂等设置当前用户已收藏，并返回收藏状态和 `favorite_count`。 |
| `DELETE` | `/api/videos/{video_id}/favorite` | Video Interactions / 视频互动 | Catalog | 幂等取消当前用户收藏，并返回收藏状态和 `favorite_count`。 |
| `POST` | `/api/video-watch-progress` | Watch Progress / 观看进度 | Catalog，User stats projection | 同事务维护 watch session ledger、Catalog 视频消费投影和 User watch stats；返回 accepted。 |
| `POST` | `/api/learning-interactions:batch` | Learning Events / 学习事件写入 | Analytics，Learning Engine best-effort normalizer，User daily stats | 写入 exposure / lookup raw facts；HTTP success 只承诺 raw accepted，normalization 是同步 best-effort。 |
| `POST` | `/api/quiz-attempts` | Learning Events / 学习事件写入 | Analytics，Learning Engine best-effort normalizer，User stats projection | 写入 quiz attempt raw fact；duplicate 不重复增加 stats；HTTP success 只承诺 raw accepted。 |
| `POST` | `/api/learning-units:mark-mastered` | Learning Events / 学习事件写入 | Analytics，Learning Engine self-mark normalizer | 写入 self-mark mastered raw fact，并走 dedicated normalizer path。 |
| `GET` | `/api/learning/unit-progress/mastered` | Unit Progress / 学习单元进度读取 | Learning Engine reducer read model，Semantic 展示字段 | 分页读取当前用户已掌握学习单元。 |
| `GET` | `/api/learning/unit-progress/unmastered` | Unit Progress / 学习单元进度读取 | Learning Engine reducer read model，Semantic 展示字段 | 分页读取当前用户尚未掌握的目标学习单元。 |
| `POST` | `/api/feedback` | User Feedback / 用户反馈上传 | User | 接收当前用户 multipart feedback，校验 JSON object payload 与最多 5 张 JPEG 图片，在 5 MiB 总请求限制内把 submission 与图片二进制原子写入 `app_user.feedback_*`。 |

## 已实现 API 分组说明

### Feed / 推荐流

| Method | Path | 说明 |
|---|---|---|
| `POST` | `/api/feed` | 获取当前用户 feed 推荐视频列表；请求只接受 `target_video_count` 和 `client_context`，API facade 调用 Recommendation 生成推荐计划，并补齐 Catalog / Semantic 列表 preview 字段和 learning unit 展示文本。 |

### Video Detail / 视频详情

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/videos/{video_id}` | 读取单个可展示视频的播放资源、transcript URL、description、全局互动计数和当前用户 like / favorite 状态；只读 Catalog，不写状态。 |

### Video Library / 视频列表

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/video-favorites` | 分页读取当前用户仍收藏且仍可展示的视频列表；返回列表 preview 和 `favorited_at`，不返回播放详情字段。 |
| `GET` | `/api/video-history` | 分页读取当前用户最近观看且仍可展示的视频列表；返回列表 preview、`last_position_ms` 和 `last_watched_at`，不直接读取 Analytics watch event ledger。 |

### End Quiz / 视频末尾取题

| Method | Path | 说明 |
|---|---|---|
| `POST` | `/api/videos/end-quiz` | 按 `video_id + coarse_unit_ids` 批量读取视频末尾 quiz 候选题；只读 Catalog，不写 delivery/session、学习进度或统计。 |

### Me / 当前用户

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/me` | 读取当前用户基础 profile、累计活动统计和内嵌 7 天 activity calendar；可根据合法 `X-Client-Timezone` 顺手更新 timezone，并在 profile 缺失时 lazy repair。 |

### Unit Collections / 词书列表

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/unit-collections` | 读取当前 active 词书列表，并返回当前用户 `active_collection` slug / null；列表来自 Semantic，当前选择来自 Learning Engine。 |

### Learning Targets / 学习目标

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/learning-targets/active-coarse-unit-ids` | 读取当前用户仍可用于 exposure 上报过滤的 active target coarse unit ids；没有 active profile 时返回空列表。 |
| `PUT` | `/api/learning-targets/active-collection` | 为当前用户激活一本词书；API facade 同事务维护 Learning Engine target projection 和 User onboarding 状态。 |

### Video Interactions / 视频互动

| Method | Path | 说明 |
|---|---|---|
| `PUT` | `/api/videos/{video_id}/like` | 当前用户点赞视频；幂等 set。 |
| `DELETE` | `/api/videos/{video_id}/like` | 当前用户取消点赞视频；幂等 unset。 |
| `PUT` | `/api/videos/{video_id}/favorite` | 当前用户收藏视频；幂等 set。 |
| `DELETE` | `/api/videos/{video_id}/favorite` | 当前用户取消收藏视频；幂等 unset。 |

### Watch Progress / 观看进度

| Method | Path | 说明 |
|---|---|---|
| `POST` | `/api/video-watch-progress` | 上报视频观看进度；Catalog 同事务维护 watch session ledger、视频消费投影和 User watch stats。 |

### Learning Events / 学习事件写入

| Method | Path | 说明 |
|---|---|---|
| `POST` | `/api/learning-interactions:batch` | 批量写入 exposure / lookup raw learning interactions；HTTP success 只承诺 raw fact accepted。 |
| `POST` | `/api/quiz-attempts` | 写入一次 quiz attempt raw fact；Learning Engine normalization 作为 best-effort 同步尝试；duplicate 不重复增加 User stats。 |
| `POST` | `/api/learning-units:mark-mastered` | 写入 self-mark mastered raw fact；走 dedicated Analytics writer 和 self-mark normalizer path。 |

### Unit Progress / 学习单元进度读取

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/learning/unit-progress/mastered` | 分页读取当前用户已掌握学习单元；从 principal 取 `user_id`，Learning Engine reducer read usecase join `semantic.coarse_unit` 返回展示字段。 |
| `GET` | `/api/learning/unit-progress/unmastered` | 分页读取当前用户尚未掌握的目标学习单元；使用 cursor keyset pagination。 |

### User Feedback / 用户反馈上传

| Method | Path | 说明 |
|---|---|---|
| `POST` | `/api/feedback` | 上传当前用户反馈；接收一个自定义 JSON object payload 和最多 5 张 JPEG 图片，完整 multipart request body 限制为 5 MiB。 |

## 当前 API 实现状态

| 文档 | 设计文档状态 | API 实现状态 | 说明 |
|---|---|---|---|
| [API模块总体设计规范.md](API模块总体设计规范.md) | 已写入 | 已实现基座 | `internal/api` 基座、server bootstrap、router、middleware、错误响应、测试底座已落地。 |
| [学习事件上报API设计.md](学习事件上报API设计.md) | 已写入 | 已实现基础入口 | 已包含 `POST /api/learning-interactions:batch`、`POST /api/quiz-attempts`、`POST /api/learning-units:mark-mastered`；HTTP success 只承诺 raw accepted。 |
| [Unit-Progress-API-MVP设计.md](Unit-Progress-API-MVP设计.md) | 已写入 | 已实现 | 已包含 `GET /api/learning/unit-progress/mastered`、`GET /api/learning/unit-progress/unmastered`；API handler 从 principal 取 `user_id`，Learning Engine reducer read usecase join `semantic.coarse_unit` 返回展示字段。 |
| [Catalog-观看进度上报MVP设计.md](Catalog-观看进度上报MVP设计.md) | 已写入 | 已实现 | 已包含 `POST /api/video-watch-progress`；Catalog 同事务维护 watch session ledger 与视频消费投影。 |
| [Video-Interactions-API-MVP设计.md](Video-Interactions-API-MVP设计.md) | 已写入 | 已实现 | 已包含 `PUT/DELETE /api/videos/{video_id}/like` 与 `PUT/DELETE /api/videos/{video_id}/favorite`；Catalog 同事务维护用户状态与互动计数。 |
| [Feed-API-MVP设计.md](Feed-API-MVP设计.md) | 已写入 | 已实现 | 已包含 `POST /api/feed`；请求只接受 `target_video_count` 和 `client_context`，API facade 调用 Recommendation 并批量补齐列表 preview 字段和 learning unit 展示文本。 |
| [Video-Detail-API-MVP设计.md](Video-Detail-API-MVP设计.md) | 已写入 | 已实现 | 已包含 `GET /api/videos/{video_id}`；Catalog 只读返回播放资源、transcript URL、description、互动计数和当前用户 like / favorite 状态。 |
| [Video-Favorites-API-MVP设计.md](Video-Favorites-API-MVP设计.md) | 已写入 | 已实现 | 已包含 `GET /api/video-favorites`；Catalog 只读 keyset 分页返回当前用户收藏视频 preview 和 `favorited_at`。 |
| [Video-History-API-MVP设计.md](Video-History-API-MVP设计.md) | 已写入 | 已实现 | 已包含 `GET /api/video-history`；Catalog 只读 keyset 分页返回当前用户观看历史 preview、`last_position_ms` 和 `last_watched_at`。 |
| [End-Quiz-批量取题API-MVP设计.md](End-Quiz-批量取题API-MVP设计.md) | 已写入 | 已实现 | 已包含 `POST /api/videos/end-quiz`；Catalog read usecase 批量读取 video-context / unit-generic quiz 候选并 fallback。 |
| [Unit-Collections-API-MVP设计.md](Unit-Collections-API-MVP设计.md) | 已写入 | 已实现 | 已包含 `GET /api/unit-collections` 的词书列表契约，并记录 `PUT /api/learning-targets/active-collection` 的业务语义；代码层前者由 `unitcollections` handler 负责，后者由 `learningtargets` handler 负责。 |
| [Active-Learning-Targets-API-MVP设计.md](Active-Learning-Targets-API-MVP设计.md) | 已写入 | 已实现 | 已包含 `GET /api/learning-targets/active-coarse-unit-ids`；读取当前用户 `is_target=true AND status!='mastered'` 的 coarse unit ids，用于 fullscreen exposure 过滤。 |
| [Me-API-MVP设计.md](Me-API-MVP设计.md) | 已写入 | 已实现 | 已包含 `GET /api/me`；User 模块读取 `app_user.user_profiles`、累计 stats 和 daily stats，必要时 lazy repair，并按合法 `X-Client-Timezone` 更新 timezone。`activity_calendar` 内嵌在 `/api/me` 响应中，返回 `current_streak_days`，不返回 `days[].is_active`。 |
| [User-Feedback-API-MVP设计.md](User-Feedback-API-MVP设计.md) | 已写入 | 已实现 | 已包含 `POST /api/feedback`；由 User 模块写 `app_user.feedback_submissions` 与 `app_user.feedback_images`，总请求限制 5 MiB，图片以 `bytea` 存储。 |

## 未开始范围

以下内容在 API 层仍未实现：

- 后端内置生产级 JWT verifier；当前模型仍是 trusted Gateway userinfo principal。
- 完整 OpenAPI / 客户端 SDK 生成。

后续新增 endpoint 时，应继续遵守 [API模块总体设计规范.md](API模块总体设计规范.md)，并按对应业务 API 文档逐个 endpoint 落地。
