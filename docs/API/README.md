# API 设计文档索引

本目录统一收口已实现和后续实现时需要遵守的 API 设计文档。

重要状态说明：

- **API 基座已落地。** 当前仓库已有 `internal/api` 目录、HTTP server bootstrap、router、middleware、handler、API DTO mapper 和 API 层测试。
- **学习事件上报 API 已实现基础 HTTP 入口。** 当前已包含 learning interaction batch、quiz attempt、self mark mastered 三条写入 endpoint。
- **移动端 MVP 不实现 CORS。** 当前入口面向原生客户端；如未来增加 Web 前端，再单独增加 CORS middleware 与 allowlist 配置。
- **其他业务 API 仍是设计文档。** Unit progress 与 Catalog watch-progress 尚未实现 HTTP handler。

## 总体规范

- [API模块总体设计规范.md](API模块总体设计规范.md)：`internal/api` 的统一入口规范。后续所有 API endpoint group 都必须遵守。

## 业务 API 设计

- [学习事件上报API设计.md](学习事件上报API设计.md)：learning interactions batch 与 quiz attempt 单点上报。
- [Learning-Engine-Unit-Progress-API-MVP设计.md](Learning-Engine-Unit-Progress-API-MVP设计.md)：用户学习单元进度分页读取。
- [Catalog-观看进度上报MVP设计.md](Catalog-观看进度上报MVP设计.md)：视频观看进度上报。

具体业务 API 文档只定义 endpoint 字段、业务语义、成功边界和前端样例；通用认证、错误 envelope、状态码、handler 结构和测试要求统一看总体规范。

## 当前 API 实现状态

| 文档 | 设计文档状态 | API 实现状态 | 说明 |
|---|---|---|---|
| [API模块总体设计规范.md](API模块总体设计规范.md) | 已写入 | 已实现基座 | `internal/api` 基座、server bootstrap、router、middleware、错误响应、测试底座已落地。 |
| [学习事件上报API设计.md](学习事件上报API设计.md) | 已写入 | 已实现基础入口 | 已包含 `POST /api/learning-interactions:batch`、`POST /api/quiz-attempts`、`POST /api/learning-units:mark-mastered`；HTTP success 只承诺 raw accepted。 |
| [Learning-Engine-Unit-Progress-API-MVP设计.md](Learning-Engine-Unit-Progress-API-MVP设计.md) | 已写入 | 未开始 | 只定义未来读取用户学习单元进度的分页契约；当前没有 HTTP handler。 |
| [Catalog-观看进度上报MVP设计.md](Catalog-观看进度上报MVP设计.md) | 已写入 | 未开始 | 只定义未来观看进度上报与聚合边界；当前没有 HTTP handler。 |

## 未开始范围

以下内容在 API 层仍未实现：

- Learning Engine Unit Progress API handler。
- Catalog watch-progress API handler。
- 生产级 auth verifier；当前模型仍是 trusted upstream principal。
- 完整 OpenAPI / 客户端 SDK 生成。

后续新增 endpoint 时，应继续遵守 [API模块总体设计规范.md](API模块总体设计规范.md)，并按对应业务 API 文档逐个 endpoint 落地。
