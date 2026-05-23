# API 模块总体设计规范

## 0. 文档信息

文档状态：总体设计规范，`internal/api` 基座已实现
目标读者：后端、前端、后续接手维护的人
当前范围：定义 `internal/api` 作为统一流量入口时的 owner 边界、目录结构、请求处理规范、跨模块调用规则、错误响应规范和测试要求。后续新增 endpoint group 必须遵守本文。
当前明确不做：本文不定义具体业务 endpoint 的全部字段；具体 endpoint 字段仍由对应业务 API 文档定义。

关联文档：

- [编码和结构规范.md](../编码和结构规范.md)
- [学习事件上报API设计.md](学习事件上报API设计.md)
- [Catalog-观看进度上报MVP设计.md](Catalog-观看进度上报MVP设计.md)
- [当前实现现状.md](../当前实现现状.md)

## 1. 一句话结论

`internal/api` 是系统唯一的外部 HTTP 流量入口。它不是第五个业务域，不拥有业务表、业务规则、migration 或跨域状态。

它只负责：

- 从可信认证上下文解析 principal，并强制执行 endpoint 的认证要求；
- HTTP request 解析；
- transport-level validation；
- 调用一个或多个业务模块 application usecase；
- 把业务结果转换为稳定 HTTP response；
- 统一错误格式、状态码、日志、request id 和超时控制。

业务模块继续保持当前 owner：

- `catalog` 拥有内容事实；
- `analytics` 拥有 raw facts；
- `learningengine` 拥有学习事件和学习状态；
- `recommendation` 拥有推荐主链路、serving state 和推荐审计。

`internal/api` 只编排这些模块，不把业务规则搬进 HTTP handler。

## 2. Owner 边界

### 2.1 API 负责

- 定义对外 HTTP 路由。
- 从可信认证上下文得到当前用户身份。
- 解析 path / query / header / JSON body。
- 做 transport-level validation，例如 required 字段、JSON object 类型、数组大小、时间格式、UUID 格式、query pagination 参数。
- 调用业务模块 application usecase。
- 组合多个业务 usecase 形成一次 API 请求链路。
- 统一响应 envelope、错误码、状态码和日志字段。
- 将内部 usecase DTO 映射为前端契约 DTO。

### 2.2 API 不负责

- 不拥有数据库表、migration、SQLC、repository、tx manager。
- 不直接读写数据库。
- 不实现 reducer、normalizer、recommendation ranking、catalog ingest 等业务规则。
- 不计算 `progress_quality`、`reducer_effect`、推荐分数或学习状态。
- 不从 request body、query、path 或普通未签名 header 信任 `user_id`。
- 不把业务模块的内部 DTO 原样暴露给前端。
- 不在 handler 中临时拼复杂 SQL 或直接构造 repository。

## 3. 依赖方向

允许的依赖方向：

```text
cmd/server or future app bootstrap
        ↓
internal/api
        ↓
business module application usecases
        ↓
business module domain / infrastructure
```

禁止的依赖方向：

```text
catalog / analytics / learningengine / recommendation -> internal/api
business domain -> HTTP request / response type
business domain -> middleware / router / auth implementation
internal/api -> business module infrastructure repository directly
```

`internal/api` 可以 import 各业务模块的 application DTO / usecase interface / service constructor，但不得直接 import 持久化 repository 作为业务逻辑捷径。依赖装配应在 server bootstrap 中完成，handler 只接收已经注入的 usecase 或 facade。

## 4. 认证模型

生产默认采用“GCP API Gateway / Auth provider 完成认证，`internal/api` 解析可信 principal”的模型：

```text
frontend
  -> Authorization: Bearer <JWT>
GCP API Gateway / auth provider
  -> verify token / session / cookie
  -> attach X-Apigateway-Api-Userinfo
  -> internal/api
       -> decode userinfo payload
       -> enforce auth required
       -> pass user_id to business usecase
```

网关或 Auth provider 负责：

- 校验 token / session / cookie。
- 校验签名、过期时间、issuer、audience。
- 注入 `X-Apigateway-Api-Userinfo`，或通过受信任 runtime context 传递 identity。
- 可选执行粗粒度限流、TLS、WAF 等边缘能力。

`internal/api` 负责：

- 优先从 `X-Apigateway-Api-Userinfo` 解析 principal。
- 当且仅当 `DEV_MODE=true` 且 gateway userinfo header 缺失时，从 `Authorization: Bearer <JWT>` 解码 payload 作为本地/联调 fallback。
- 固定从 JWT payload 的 `sub` claim 取 `user_id`。
- 对需要登录的 endpoint 强制 principal 存在。
- 将 principal 中的 `user_id` 传给业务 usecase。
- 检查 path / query 中的资源是否允许当前 principal 访问。
- 为本地测试提供 fake principal middleware。

`internal/api` 不自行实现 JWT 签名校验。生产必须由 API Gateway / Auth provider 验证 JWT 后再注入 userinfo header。`DEV_MODE=true` 的 Authorization fallback 只解码 payload，不验签，只用于可信前端直连测试。

禁止：

- 从 request body 读取可信 `user_id`。
- 在 `DEV_MODE=false` 时从普通客户端可伪造 header 或 Authorization payload 读取可信 `user_id`。
- 当 gateway userinfo header 存在但格式非法时 fallback 到 Authorization。
- 让前端上传 `user_id` 来选择写入用户。
- 在业务模块中处理 HTTP 认证细节。

## 5. 目录结构

`internal/api` 是 transport/composition 模块，不使用业务域的完整 `domain` 骨架。标准结构如下：

```text
internal/api/
  README.md
  doc.go
  application/
    dto/
    service/
  infrastructure/
    http/
      auth/
      handler/
      middleware/
      request/
      response/
      router/
  test/
    unit/
    integration/
```

### 5.1 `application`

`application` 只放 API 层编排，不放业务规则。

- `dto/`：API 层内部的 request / response DTO，不等同于业务模块 DTO。
- `service/`：需要跨多个业务 usecase 的请求编排。例如学习事件上报需要先写 Analytics raw fact，再调用 Learning Engine normalizer。

如果某个 endpoint 只调用一个业务 usecase，handler 可以直接调用注入的 usecase，不需要额外加 application service。

### 5.2 `infrastructure/http`

HTTP 技术实现统一放在 `infrastructure/http`。

- `auth/`：可信 principal 解析、principal model、认证要求执行；默认不做完整 token verification。
- `handler/`：按 endpoint group 分目录，例如 `learningevents`、`recommendations`。
- `middleware/`：request id、panic recovery、timeout、logging、body size limit。
- `request/`：通用 request parsing helper，例如 JSON object decode、query pagination parse。
- `response/`：统一 success/error response writer。
- `router/`：路由注册，聚合 endpoint group。

handler 目录按 API 业务面命名，不按底层 owner 命名。示例：

```text
handler/
  learningevents/
    handler.go
    learning_interactions_batch.go
    quiz_attempts.go
    self_mark_mastered.go
  recommendations/
    video_recommendations.go
```

### 5.3 `test`

- `test/unit`：handler、request parser、response mapper、API application service 的纯单测。
- `test/integration`：真实 HTTP server + fake 或真实 usecase 装配的 API integration。
- 跨多个业务模块的真实数据库闭环仍放在 `internal/test/e2e`，不放进 `internal/api/test`。

## 6. Endpoint Group 规范

每个 endpoint group 必须有清晰边界。小型 group 可以用一个 `handler.go` 承担 route registration 和 shared helper；复杂 group 再拆出 `routes.go` / `mapper.go`：

```text
handler/<group>/
  routes.go
  <endpoint>.go
  mapper.go
  errors.go      # 只有确有必要时
```

约束：

- `routes.go` 只注册该 group 的路由。
- `<endpoint>.go` 只做单个 endpoint 的 HTTP request/response 流程。
- `mapper.go` 只做 API DTO 与业务 usecase DTO 的转换。
- 不新增 `utils.go`、`common.go`、`misc.go` 这类低信息文件。
- 同一个 endpoint 的 validation 不得散落在 handler、mapper、业务 usecase 三处。

## 7. Request 处理流程

所有 HTTP handler 必须遵守同一顺序：

```text
1. get request context
2. require and extract trusted principal
3. enforce method / content-type / body size
4. decode JSON or parse query/path params
5. transport-level validation
6. map to business usecase request
7. call injected usecase/application service
8. map result to API response
9. write JSON response
```

### 7.1 用户身份

- `user_id` 永远来自可信 principal。
- request body 中不允许出现可信 `user_id`。
- 如果前端为了调试传了 `user_id`，handler 应拒绝或忽略；具体 endpoint 文档必须明确。
- 跨用户资源访问必须在 API 层先做 path/query 参数和 principal 的一致性检查，再交给业务 usecase。

### 7.2 Validation 分层

API 层 validation 只处理 transport 契约：

- JSON 格式；
- required 字段；
- UUID / int / time 格式；
- 数组大小；
- string 枚举值；
- query pagination 上限；
- `client_context` / `event_payload` 必须是 JSON object。

业务 usecase 继续处理 owner 内业务规则：

- Analytics 判断 raw fact 是否可接受并幂等写入；
- Learning Engine normalizer 判断 raw fact 是否能进入 Learning Engine；
- Learning Engine reducer 校验 normalized event contract；
- Recommendation 判断推荐请求是否满足生成条件。

API 不应复制业务模块 validation。业务模块返回的 validation error 由 API 映射成统一错误响应。

### 7.3 Partial Success

默认策略：

- 写 API 默认不做 partial success。
- 任意 transport validation 失败时，请求整体返回 `400`，不调用业务 usecase。
- 如果某个 API 明确允许 partial success，必须在该 API 设计文档中单独说明响应格式、重试语义和幂等边界。

学习事件上报是该默认策略的一个例子：

- `POST /api/learning-interactions:batch` 整批 validation；任意一条非法则整批拒绝。
- `POST /api/quiz-attempts` 单点 completed attempt；非法则拒绝。
- `POST /api/learning-units:mark-mastered` 单点 self mark；非法则拒绝。
- `POST /api/learning-units:reset-unlearned` 单点 reset；非法则拒绝。
- `POST /api/video-watch-progress` 单点 watch session summary；非法则拒绝，成功只返回 `{ "accepted": true }`。

### 7.4 `client_context` 规范

客户端上报类 API 使用统一的 `client_context` 表达客户端环境。当前前端上传契约推荐携带四个基础字段：

```json
{
  "client_context": {
    "platform": "ios",
    "app_version": "1.3.0",
    "os_version": "18.5",
    "device_model": "iPhone16,2"
  }
}
```

推荐字段：

| 字段 | 类型 | 必需 | 说明 |
| --- | --- | --- | --- |
| `platform` | string | 建议 | 客户端平台，例如 `ios` / `android` / `web`。 |
| `app_version` | string | 建议 | App 版本。 |
| `os_version` | string | 建议 | 系统版本。 |
| `device_model` | string | 建议 | 设备型号。 |

维护规则：

- 上报类 API 包括 learning interaction batch、quiz attempt、watch progress。
- `client_context` 只要求是 JSON object；后端不在 application / DB 层固定字段集合。
- `client_context` 可以随客户端遥测演进增加字段；新增字段不应改变业务行为。
- `client_context` 只描述客户端环境，不描述业务入口。业务入口应使用 `source_surface`、`trigger_type` 或具体业务字段表达。
- 上报类 command API 优先让请求 body 自包含业务事实字段，例如 `video_id`、`watch_session_id`、`coarse_unit_id`；path 参数只用于真正的资源读取或资源子命令。
- `timezone` 不参与任何 `*_at` 时间字段解析；前端不需要上传 `timezone`。

### 7.5 时间字段规范

所有 API 必须统一区分三类时间相关字段。

#### 时间点：`*_at`

`*_at` 字段表示一个真实发生过或将要发生的瞬间，例如：

- `occurred_at`
- `shown_at`
- `completed_at`
- `started_at`
- `last_seen_at`
- `created_at`
- `updated_at`
- `next_review_at`
- `last_served_at`

统一规则：

| 层 | 契约 |
| --- | --- |
| 前端 | 用 `Date` 表达时间点；生成当前时间时使用 `new Date()`。 |
| API request JSON | string，必须是 RFC3339 / RFC3339Nano，并且必须带显式 `Z` 或 offset。 |
| API handler | 解析失败、无显式 offset、非真实日期时间时返回 `400 invalid_request`。 |
| business DTO / domain | 传入 `time.Time` 前统一转为 UTC。 |
| database | 所有时间点字段统一使用 `timestamptz`，禁止使用 `timestamp without time zone` 存业务时间点。 |
| API response JSON | string，统一输出 UTC `Z` 格式。 |

允许的 request 值：

```json
"2026-05-15T17:00:01Z"
"2026-05-15T17:00:01.234Z"
"2026-05-15T10:00:01-07:00"
```

不允许的 request 值：

```json
"2026-05-15T10:00:01"
"2026-05-15 10:00:01"
"2026-05-15"
```

前端默认应使用：

```ts
new Date().toISOString()
```

#### 时长和位置：`*_ms`

`*_ms` 字段只表示 duration 或 media position，不表示真实时间点，也不涉及时区。例如：

- `total_elapsed_ms`
- `selection_interval_ms`
- `position_ms`
- `active_watch_ms`
- `duration_ms`
- `exposure_start_ms`
- `exposure_end_ms`

统一规则：

- API JSON 使用 integer。
- 前端使用 `number`。
- Go 根据范围使用 `int32` / `int64`。
- DB 根据范围使用 `integer` / `bigint`。
- 不允许把时长、播放位置或字幕位置设计成 `*_at` / `timestamptz`。

#### `timezone` 不参与时间解析

API 不使用 `timezone` 解释 `*_at` 字段。

原因：

- `*_at` 字段必须自己携带 `Z` 或 offset，后端不需要再用 `timezone` 解释时间。
- Postgres `timestamptz` 保存的是 absolute instant，不保存原始时区名称。
- per-event `timezone` 容易让重试、replay、normalization 和本地日期统计产生隐式解释规则。

如果未来需要“用户本地日期”语义，例如 streak / daily review / local day summary，应单独设计用户设置或服务端可信配置，再由后端从 UTC 时间点派生 `local_date`。普通事件上报 payload 里的 `timezone` 即使存在，也不能用于解释事件时间。

## 8. Response 规范

所有 JSON response 使用 snake_case 字段。

成功响应由具体 endpoint 定义，但必须遵守：

- 不暴露内部 SQL row。
- 不暴露业务模块内部调试字段。
- 不返回前端不需要知道的 reducer / ranking / normalization 内部细节。
- 时间点字段统一用 UTC `Z` 格式的 RFC3339 / RFC3339Nano 字符串。
- ID 统一用字符串。

错误响应统一格式：

```json
{
  "error": {
    "code": "invalid_request",
    "message": "events must not be empty",
    "details": [
      {
        "field": "events",
        "reason": "required"
      }
    ],
    "request_id": "req_01HY..."
  }
}
```

字段含义：

| 字段 | 必需 | 说明 |
| --- | --- | --- |
| `error.code` | 是 | 稳定机器码。前端可以依赖。 |
| `error.message` | 是 | 面向开发和基础 UI 的简短说明。 |
| `error.details` | 否 | 字段级错误，主要用于 validation。 |
| `error.request_id` | 是 | 用于日志关联。 |

推荐错误码：

| `error.code` | 默认状态码 | 使用场景 |
| --- | --- | --- |
| `invalid_request` | `400` | JSON、字段类型、required、枚举、数组大小、时间格式等请求契约错误。 |
| `unauthorized` | `401` | 可信 principal 缺失，或认证适配层判定认证无效。 |
| `forbidden` | `403` | principal 存在，但无权访问目标资源。 |
| `not_found` | `404` | 资源不存在，且不需要隐藏权限信息。 |
| `conflict` | `409` | 与当前资源状态冲突，原样重试不会自然成功。 |
| `business_rule_rejected` | `422` | 请求格式正确，但业务状态拒绝；仅在 endpoint 文档明确需要时使用。 |
| `rate_limited` | `429` | 限流。 |
| `internal_error` | `500` | 未预期错误。 |
| `service_unavailable` | `503` | 依赖不可用或服务降级。 |

业务 API 文档不需要重复这张表。只有偏离默认映射时，才在具体 endpoint 文档里说明。

## 9. HTTP 状态码

统一状态码策略：

| 状态码 | 使用场景 |
| --- | --- |
| `200 OK` | 查询成功，或幂等写入返回已存在资源。 |
| `201 Created` | 单资源新建成功且对前端有资源创建语义。 |
| `202 Accepted` | 已接受但明确异步处理，结果不保证完成。MVP 暂不默认使用。 |
| `204 No Content` | 删除 / 关闭类命令成功且无需 body。 |
| `400 Bad Request` | JSON 格式、字段类型、required、枚举、数组大小等请求契约错误。 |
| `401 Unauthorized` | 可信 principal 缺失，或认证适配层判定认证无效。 |
| `403 Forbidden` | 已认证但无权访问该资源。 |
| `404 Not Found` | 资源不存在，且不需要隐藏权限信息。 |
| `409 Conflict` | 请求与当前资源状态冲突，重试同请求不会自然成功。 |
| `422 Unprocessable Entity` | 请求格式正确，但业务规则拒绝。只有当业务文档明确区分 400/422 时使用。 |
| `429 Too Many Requests` | 限流。 |
| `500 Internal Server Error` | 未预期错误。 |
| `503 Service Unavailable` | 依赖不可用或服务降级。 |

默认 validation 错误使用 `400`。业务 usecase 返回的 owner validation 如果是前端可修正输入，也映射为 `400`；只有需要明确表达“格式正确但业务状态不允许”时才用 `422` 或 `409`。

## 10. 幂等与重试

API 层只负责传递幂等键，不自己实现业务幂等。

规则：

- 前端重试写请求时必须复用同一个 `client_event_id` 或业务定义的 idempotency key。
- API handler 不得生成新的业务幂等键来掩盖前端重试错误。
- 幂等落点在 owner 模块，例如 Analytics 使用 `(user_id, client_event_id)`。
- 幂等重复不是错误，响应必须能让前端知道 raw fact 已存在或请求已被接受。

如果未来有非 raw fact 写 API，需要在具体 API 文档中明确：

- 幂等 key 来源；
- 幂等作用域；
- 幂等命中时返回 `200` 还是 `201`；
- 重试是否可能造成重复副作用。

## 11. 事务与副作用

API handler 不直接开启数据库事务。事务由业务模块 usecase 自己管理。

跨模块链路默认不做分布式事务。API 层要按业务语义定义成功边界：

- 如果第一步写入 owner fact 成功，后续派生步骤失败，是否仍返回成功；
- 失败是否依赖后台 repair/backfill；
- 是否需要日志、metric 或 future job 补偿。

学习事件上报是该规则的一个例子：

```text
raw fact accepted = API success
normalizer / reducer = synchronous best effort + pending repair
```

这类边界必须写进具体 API 文档，不能藏在 handler 实现里。

## 12. 超时、取消与并发

- 所有 usecase 调用必须使用 request context。
- Handler 不创建脱离 request context 的 goroutine 来执行业务写入。
- 如果需要异步处理，必须通过明确的 queue / job 机制设计；MVP 不在 API 层临时 fire-and-forget。
- 同一请求内可以顺序调用多个 usecase；是否并行调用必须由具体 API 文档说明，并证明不会破坏 owner 幂等和事务语义。
- HTTP server 级别必须设置 read / write / idle timeout。

## 13. 日志与观测

每个请求至少应具备：

- `request_id`
- `method`
- `path`
- `status_code`
- `duration_ms`
- principal `user_id`，如有
- endpoint group
- error code，如失败

日志不得写入：

- access token；
- 完整 Authorization header；
- 大体量 request body；
- quiz 选项详情之外的敏感用户信息；
- 未脱敏的外部凭证。

API 层日志只记录入口和编排结果。业务模块内部错误应保留原始 error chain，API response 再映射为稳定错误码。

## 14. 安全与输入限制

必须统一实现：

- body size limit；
- JSON 写 API 必须显式携带 `Content-Type: application/json`；
- JSON object 顶层检查；
- unknown field 策略由具体 endpoint 决定，但默认拒绝未知字段，避免前端误以为字段已生效；
- 所有写 API 必须认证；
- 所有用户级读 API 必须按 principal 限制用户范围；
- 不从 path/query/body 中信任 user identity。

当前移动端 MVP 只面向原生客户端，不实现 CORS middleware，也不把 CORS 作为 endpoint 落地前置条件。未来如果出现 Web 前端或 browser client，再单独增加 CORS middleware 与配置化 allowlist；不要在当前移动端链路中提前维护一套无调用方的 CORS 策略。

## 15. 新 API 设计文档必须说明什么

每新增一个业务 API 文档，都必须回答这些问题，避免把关键语义藏进 handler 实现：

| 问题 | 必须说明的内容 |
| --- | --- |
| Endpoint | method、path、是否认证、是否用户级资源。 |
| Owner chain | API 会调用哪些业务 owner 的哪些 usecase。 |
| Success boundary | HTTP success 到底承诺什么，不承诺什么。 |
| Request schema | body / query / path 字段、类型、必需性、默认值。 |
| Response schema | 成功响应字段，不允许暴露的内部字段。 |
| Idempotency | 是否需要 key、key 来源、作用域、重复时响应。 |
| Validation split | 哪些由 API transport validation 处理，哪些交给业务 usecase。 |
| Side effects | 会写哪些 owner 模块，是否有派生步骤、补偿或异步 repair。 |
| Retry guidance | 前端什么时候重试、是否复用 idempotency key。 |
| Error deviations | 是否偏离统一错误格式或状态码；没有偏离就只引用本文。 |

业务 API 文档应该专注该 endpoint 的字段和业务语义，不重复本文的通用认证、错误 envelope、状态码表、目录结构和 handler 流程。

## 16. 与业务文档的关系

本文只规定 `internal/api` 模块的一般规则。具体 endpoint 的字段、业务语义、成功边界和例子放在业务 API 文档。

例如：

- [学习事件上报API设计.md](学习事件上报API设计.md) 定义 `POST /api/learning-interactions:batch`、`POST /api/quiz-attempts`、`POST /api/learning-units:mark-mastered`、`POST /api/learning-units:reset-unlearned` 的字段、样例与业务语义。
- 本文规定这些 endpoint 在 `internal/api` 中应该如何落目录、如何认证、如何 validation、如何调用 Analytics / Learning Engine、如何统一错误响应。

业务 API 文档不得重复本文的通用规范；如果某个 endpoint 需要偏离本文，必须在该 endpoint 文档中写清原因。

## 17. 示例：学习事件 API 的落地方式

学习事件上报当前实现使用以下结构：

```text
internal/api/
  infrastructure/http/handler/learningevents/
    routes.go
    learning_interactions_batch.go
    quiz_attempts.go
    self_mark_mastered.go
    reset_user_unit_progress.go
    mapper.go
  application/service/
    record_learning_interactions_batch.go
    record_quiz_attempt.go
    record_self_mark_mastered.go
    reset_user_unit_progress.go
```

调用链：

```text
POST /api/learning-interactions:batch
  -> extract trusted principal
  -> decode and validate API request
  -> api application service
  -> analytics.RecordLearningInteractionsBatch
  -> normalizer.NormalizeLearningInteractionsByIDs
  -> return raw accepted response

POST /api/quiz-attempts
  -> extract trusted principal
  -> decode and validate API request
  -> api application service
  -> analytics.RecordQuizAttempt
  -> normalizer.NormalizeQuizAttemptByID
  -> return raw accepted response

POST /api/learning-units:mark-mastered
  -> extract trusted principal
  -> decode and validate API request
  -> api application service
  -> analytics.RecordSelfMarkMastered
  -> normalizer.NormalizeSelfMarkMasteredByID
  -> return raw accepted response

POST /api/learning-units:reset-unlearned
  -> extract trusted principal
  -> decode and validate API request
  -> api application service
  -> reducer.ResetUserUnitProgress
  -> return reducer event accepted response
```

API response 不返回 `progress_quality`、`reducer_effect`、`learning.user_unit_states` 或 Recommendation 排序内部字段。

## 18. 测试要求

新增 endpoint 必须至少覆盖：

- request decode 成功；
- required 字段缺失；
- 字段类型错误；
- trusted principal 缺失；
- principal user id 正确传给业务 usecase；
- 业务 usecase validation error 映射；
- 业务 usecase internal error 映射；
- 成功 response 不泄露内部字段；
- 幂等重复响应语义，如该 endpoint 是写 API。

如果 endpoint 编排多个业务模块，还必须有 API application service 单测覆盖：

- 第一步成功、第二步失败时的成功边界；
- 是否 fail-fast；
- 是否返回前端可重试响应；
- 是否记录补偿所需信息。

跨模块真实数据库闭环放在 `internal/test/e2e`。

## 19. 当前默认选择

- HTTP handler 放在 `internal/api/infrastructure/http/handler`。
- API 组合服务放在 `internal/api/application/service`。
- API 不新增 migration / SQLC / repository。
- API 不定义业务 domain。
- 默认 JSON 字段使用 snake_case。
- 默认 validation 失败整请求拒绝，不 partial success。
- 默认不启动 request 外 goroutine。
- 默认业务状态不在 API 层持久化；需要状态时回到对应 owner 模块设计。
