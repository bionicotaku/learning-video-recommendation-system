# API 设计文档索引

本目录统一收口尚未实现或后续实现时需要遵守的 API 设计文档。

重要状态说明：

- **API 实现状态：全部未开始。** 当前仓库还没有 `internal/api` 目录、HTTP server、router、middleware、handler、API DTO mapper 或 API 层测试。
- **已完成只表示设计文档已落盘。** 文档中的 endpoint、request / response、错误语义和调用链都是未来实现契约，不表示接口已经可调用。
- **当前没有“API 层部分完成”的 endpoint。** 个别下游模块已经有 application usecase 或数据库能力，但这些都不是 API 层实现。

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
| [API模块总体设计规范.md](API模块总体设计规范.md) | 已写入 | 未开始 | 只定义未来 `internal/api` 的统一结构、认证上下文、错误响应、validation、测试和跨模块编排规则。 |
| [学习事件上报API设计.md](学习事件上报API设计.md) | 已写入 | 未开始 | 未来包含 `POST /api/learning-interactions:batch` 与 `POST /api/quiz-attempts`；当前没有 HTTP handler。下游 `analytics` / `learningengine normalizer` 的部分应用层能力已存在，但 API 入口未实现。 |
| [Learning-Engine-Unit-Progress-API-MVP设计.md](Learning-Engine-Unit-Progress-API-MVP设计.md) | 已写入 | 未开始 | 只定义未来读取用户学习单元进度的分页契约；当前没有 HTTP handler。 |
| [Catalog-观看进度上报MVP设计.md](Catalog-观看进度上报MVP设计.md) | 已写入 | 未开始 | 只定义未来观看进度上报与聚合边界；当前没有 HTTP handler。 |

## 未开始范围

以下内容在 API 层均未实现：

- `internal/api` 模块目录和目录级 `README.md` / `doc.go`
- HTTP server bootstrap 与 router 装配
- 认证 principal 解析 middleware
- request id / logging / timeout / panic recovery middleware
- endpoint handler
- API request / response DTO
- API DTO 到业务 usecase DTO 的 mapper
- API 层 validation
- 统一 error envelope 实现
- API unit / integration / E2E tests

因此，后续实现任一 API 时，应先以 [API模块总体设计规范.md](API模块总体设计规范.md) 建立 `internal/api` 基础结构，再按对应业务 API 文档逐个 endpoint 落地。
