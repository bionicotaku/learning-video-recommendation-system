# API 设计文档索引

本目录统一收口已实现和后续实现时需要遵守的 API 设计文档。

重要状态说明：

- **API 基座已落地。** 当前仓库已有 `internal/api` 目录、HTTP server bootstrap、router、middleware、handler、API DTO mapper 和 API 层测试。
- **学习事件上报 API 已实现基础 HTTP 入口。** 当前已包含 learning interaction batch、quiz attempt、self mark mastered 三条写入 endpoint。
- **移动端 MVP 不实现 CORS。** 当前入口面向原生客户端；如未来增加 Web 前端，再单独增加 CORS middleware 与 allowlist 配置。
- **Feed、End Quiz、Catalog watch-progress、Video Interactions、Unit Progress 已落地。** Unit Progress 提供 mastered / unmastered 两个分页读取 endpoint。
- **认证 principal adapter 已支持 GCP API Gateway userinfo。** 后端仍不自行验证 JWT 签名；生产由 Gateway 验证 JWT，后端解析 `X-Apigateway-Api-Userinfo`。

## 总体规范

- [API模块总体设计规范.md](API模块总体设计规范.md)：`internal/api` 的统一入口规范。后续所有 API endpoint group 都必须遵守。

## 业务 API 设计

- [学习事件上报API设计.md](学习事件上报API设计.md)：learning interactions batch 与 quiz attempt 单点上报。
- [Unit-Progress-API-MVP设计.md](Unit-Progress-API-MVP设计.md)：用户学习单元进度分页读取。
- [Catalog-观看进度上报MVP设计.md](Catalog-观看进度上报MVP设计.md)：视频观看进度上报。
- [Video-Interactions-API-MVP设计.md](Video-Interactions-API-MVP设计.md)：视频点赞/取消点赞、收藏/取消收藏。
- [Feed-API-MVP设计.md](Feed-API-MVP设计.md)：feed 页面获取推荐视频列表的前端展示契约。
- [End-Quiz-批量取题API-MVP设计.md](End-Quiz-批量取题API-MVP设计.md)：视频末尾按 `video_id + coarse_unit_ids` 批量取 quiz 题。

具体业务 API 文档只定义 endpoint 字段、业务语义、成功边界和前端样例；通用认证、错误 envelope、状态码、handler 结构和测试要求统一看总体规范。

## 已实现 Endpoint 总览

当前 `internal/api` 已实现以下 HTTP endpoint：

### Feed / 推荐流

| Method | Path | 说明 |
|---|---|---|
| `POST` | `/api/feed` | 获取当前用户 feed 推荐视频列表；请求只接受 `target_video_count` 和 `client_context`，API facade 调用 Recommendation 生成推荐计划，并补齐 Catalog / semantic 展示字段。 |

### End Quiz / 视频末尾取题

| Method | Path | 说明 |
|---|---|---|
| `POST` | `/api/videos/end-quiz` | 按 `video_id + coarse_unit_ids` 批量读取视频末尾 quiz 候选题；只读 Catalog，不写学习进度。 |

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
| `POST` | `/api/video-watch-progress` | 上报视频观看进度；Catalog 同事务维护 watch session ledger 与视频消费投影。 |

### Learning Events / 学习事件写入

| Method | Path | 说明 |
|---|---|---|
| `POST` | `/api/learning-interactions:batch` | 批量写入 exposure / lookup raw learning interactions；HTTP success 只承诺 raw fact accepted。 |
| `POST` | `/api/quiz-attempts` | 写入一次 quiz attempt raw fact；Learning Engine normalization 作为 best-effort 同步尝试。 |
| `POST` | `/api/learning-units:mark-mastered` | 写入 self-mark mastered raw fact；走 dedicated Analytics writer 和 self-mark normalizer path。 |

### Unit Progress / 学习单元进度读取

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/learning/unit-progress/mastered` | 分页读取当前用户已掌握学习单元；从 principal 取 `user_id`，Learning Engine reducer read usecase join `semantic.coarse_unit` 返回展示字段。 |
| `GET` | `/api/learning/unit-progress/unmastered` | 分页读取当前用户尚未掌握的目标学习单元；使用 cursor keyset pagination。 |

## 当前 API 实现状态

| 文档 | 设计文档状态 | API 实现状态 | 说明 |
|---|---|---|---|
| [API模块总体设计规范.md](API模块总体设计规范.md) | 已写入 | 已实现基座 | `internal/api` 基座、server bootstrap、router、middleware、错误响应、测试底座已落地。 |
| [学习事件上报API设计.md](学习事件上报API设计.md) | 已写入 | 已实现基础入口 | 已包含 `POST /api/learning-interactions:batch`、`POST /api/quiz-attempts`、`POST /api/learning-units:mark-mastered`；HTTP success 只承诺 raw accepted。 |
| [Unit-Progress-API-MVP设计.md](Unit-Progress-API-MVP设计.md) | 已写入 | 已实现 | 已包含 `GET /api/learning/unit-progress/mastered`、`GET /api/learning/unit-progress/unmastered`；API handler 从 principal 取 `user_id`，Learning Engine reducer read usecase join `semantic.coarse_unit` 返回展示字段。 |
| [Catalog-观看进度上报MVP设计.md](Catalog-观看进度上报MVP设计.md) | 已写入 | 已实现 | 已包含 `POST /api/video-watch-progress`；Catalog 同事务维护 watch session ledger 与视频消费投影。 |
| [Video-Interactions-API-MVP设计.md](Video-Interactions-API-MVP设计.md) | 已写入 | 已实现 | 已包含 `PUT/DELETE /api/videos/{video_id}/like` 与 `PUT/DELETE /api/videos/{video_id}/favorite`；Catalog 同事务维护用户状态与互动计数。 |
| [Feed-API-MVP设计.md](Feed-API-MVP设计.md) | 已写入 | 已实现 | 已包含 `POST /api/feed`；请求只接受 `target_video_count` 和 `client_context`，API facade 调用 Recommendation 并批量补齐 Catalog / semantic 展示字段。 |
| [End-Quiz-批量取题API-MVP设计.md](End-Quiz-批量取题API-MVP设计.md) | 已写入 | 已实现 | 已包含 `POST /api/videos/end-quiz`；Catalog read usecase 批量读取 video-context / unit-generic quiz 候选并 fallback。 |

## 未开始范围

以下内容在 API 层仍未实现：

- 后端内置生产级 JWT verifier；当前模型仍是 trusted Gateway userinfo principal。
- 完整 OpenAPI / 客户端 SDK 生成。

后续新增 endpoint 时，应继续遵守 [API模块总体设计规范.md](API模块总体设计规范.md)，并按对应业务 API 文档逐个 endpoint 落地。
