# Legal Documents API MVP 设计

## 0. 文档信息

文档状态：MVP 已实现，作为当前后端实现和前端联调契约。
目标读者：前端、后端 API、User 模块、数据库维护者、GCP API Gateway 配置维护者。
当前范围：定义法律文档公开读取 API、无用户 JWT 鉴权的 Gateway 配置语义、`app_user.legal_documents` 存储模型、错误语义和测试要求。
当前明确不做：不做用户同意记录写入，不做 `POST /api/legal-documents:accept`，不做法律文档版本历史，不做后台编辑系统，不做 Markdown 转 HTML，不把法律文档内容塞进 `/api/me`。

关联文档：

- [API模块总体设计规范.md](API模块总体设计规范.md)
- [Me-API-MVP设计.md](Me-API-MVP设计.md)
- [User-Feedback-API-MVP设计.md](User-Feedback-API-MVP设计.md)
- [../User模块设计.md](../User模块设计.md)

## 1. API 定位

`GET /api/legal-documents/{type}` 按文档类型读取当前法律文档内容。法律文档用于未登录、登录、注册、设置页和 App Store 审核等入口展示，因此本读取 API 不要求用户登录。

法律文档不属于 `/api/me` 用户资料响应。Me 页只提供入口；文档内容由独立 API 按文档类型拉取，避免把用户资料、学习统计和 legal content 混在同一个契约里。

MVP 把法律文档存储归到 User 模块：

```text
internal/api legaldocuments.Handler
  -> user.GetLegalDocumentUsecase
  -> app_user.legal_documents
```

这个归属表示 legal documents 是用户面向的账号/合规/支持边界数据，和 `app_user.feedback_*` 同属低频用户支持能力。不表示它属于 `/api/me`、profile、onboarding 或 activity stats。User 模块可以拥有这张表，但现有 profile 与统计读取路径不得读取 legal documents。

## 2. Endpoint

```http
GET /api/legal-documents/{type}
Accept: application/json
```

认证：不要求用户 JWT 鉴权。

当前支持的 `type`：

| type | 含义 |
|---|---|
| `privacy-policy` | 隐私政策 |
| `user-agreement` | 用户协议 |

前端 request descriptor 使用相对 path，由 `requestJson` 基于 `EXPO_PUBLIC_API_BASE_URL` 拼接。当前约定 `EXPO_PUBLIC_API_BASE_URL` 已包含 `/api` 前缀，因此 descriptor 不重复写 `/api`：

```ts
{
  method: 'GET',
  path: '/legal-documents/privacy-policy',
  auth: 'none',
}
```

前端规则：

- 不传 `user_id`。
- 不要求登录态。
- 页面根据 route `type` 判断是否支持；非法 type 不发请求。
- 如果客户端已登录，也不需要为了读取文档附带 token。

后端规则：

- Handler 不调用 `auth.RequirePrincipal(...)`。
- 如果请求经过 Gateway 时带有用户信息 header，handler 也不依赖 principal 生成响应。
- `type` 只允许 `privacy-policy` 和 `user-agreement`。

## 3. GCP API Gateway 配置语义

当前后端生产入口默认部署在 GCP API Gateway 后面，其他业务 API 由 Gateway 校验客户端 JWT，并向后端注入 `X-Apigateway-Api-Userinfo`。

Legal Documents GET 是例外：它必须在 Gateway operation 层覆盖全局 `security`，使该 operation 不要求客户端 JWT。

OpenAPI 2.0 示例：

```yaml
securityDefinitions:
  firebase:
    authorizationUrl: ""
    flow: "implicit"
    type: "oauth2"
    x-google-issuer: "https://securetoken.google.com/PROJECT_ID"
    x-google-jwks_uri: "https://www.googleapis.com/service_accounts/v1/jwk/securetoken@system.gserviceaccount.com"
    x-google-audiences: "PROJECT_ID"

security:
  - firebase: []

paths:
  /api/legal-documents/{type}:
    get:
      security: []
      parameters:
        - name: type
          in: path
          required: true
          type: string
```

`security: []` 的语义是“该 operation 不做用户 JWT 鉴权”。它不是“JWT 可选”。如果未来需要“带 token 时识别用户，没 token 也允许”，不要依赖 Gateway optional OAuth/JWT security 表达，应改成后端 public endpoint 自行做可选上下文处理。

Gateway 到 Cloud Run 的服务间鉴权仍应保留。也就是说，公开的是用户 JWT 要求，不是把 Cloud Run backend 直接暴露给公网。Cloud Run 可以继续只允许 API Gateway runtime service account 调用。

## 4. Response

成功响应使用 `200 OK`。

后端 DTO：

```ts
type LegalDocumentResponseDto = {
  type: 'privacy-policy' | 'user-agreement';
  title: string;
  markdown: string;
  updated_at: string | null;
  version: string | null;
};
```

示例：

```json
{
  "type": "privacy-policy",
  "title": "隐私政策",
  "markdown": "# 隐私政策\n\n...",
  "updated_at": "2026-05-22T00:00:00Z",
  "version": "2026-05-22"
}
```

字段说明：

| 字段 | 类型 | 说明 |
|---|---|---|
| `type` | string | 文档类型。只返回请求支持的 type。 |
| `title` | string | 页面标题。 |
| `markdown` | string | Markdown 原文，不返回 HTML。 |
| `updated_at` | string \| null | 文档更新时间。存在时必须是 RFC3339 UTC 时间字符串。 |
| `version` | string \| null | 文档版本标识。MVP 可使用日期版本，例如 `2026-05-22`。 |

`markdown` 返回 Markdown 原文。当前前端只承诺普通 Markdown 渲染：标题、段落、加粗、斜体、列表、链接、引用和代码块。

## 5. Storage Model

MVP 使用 `app_user.legal_documents` 保存当前生效的法律文档内容：

```sql
create table app_user.legal_documents (
  document_type text primary key
    check (document_type in ('privacy-policy', 'user-agreement')),
  title text not null,
  markdown text not null,
  version text,
  updated_at timestamptz
);
```

初始 migration 应 seed 两行：

```sql
insert into app_user.legal_documents (
  document_type,
  title,
  markdown,
  version,
  updated_at
) values
  ('privacy-policy', '隐私政策', '...', '2026-05-22', now()),
  ('user-agreement', '用户协议', '...', '2026-05-22', now())
on conflict (document_type) do update
set
  title = excluded.title,
  markdown = excluded.markdown,
  version = excluded.version,
  updated_at = excluded.updated_at;
```

MVP 不保留历史版本表。每个 `document_type` 只有一份当前内容。后续如果需要用户接受记录或版本追溯，再扩展：

```text
app_user.legal_document_versions
app_user.user_legal_document_acceptances
```

在实现接受记录前，不需要新增这些表。

## 6. Owner 和调用边界

### 6.1 User 模块负责

- `app_user.legal_documents` migration。
- legal document SQLC query。
- `GetLegalDocument` usecase。
- 校验支持的 document type。
- 将 DB row 映射成 User application DTO。

### 6.2 API 模块负责

- 注册 `GET /api/legal-documents/{type}`。
- 解析 path 中的 `type`。
- 不要求 principal。
- 调用 User `GetLegalDocument` usecase。
- 映射错误到统一 API error envelope。
- 返回前端 DTO。

### 6.3 不允许的做法

- 不在 API handler 中硬编码整篇 Markdown。
- 不从 `/api/me` 返回法律文档内容。
- 不把 legal documents 放到 `catalog`、`semantic`、`analytics`、`learning` 或 `recommendation`。
- 不为当前两篇低频文档新建独立 `legal` module/schema。
- 不在 GET 请求中产生用户接受记录或其他副作用。

## 7. Error Semantics

错误响应使用 API 模块统一 JSON error envelope。

| HTTP | code | 场景 |
|---|---|---|
| `200 OK` | - | 文档读取成功。 |
| `400 Bad Request` | `invalid_request` | `type` 不在 `privacy-policy` / `user-agreement` 中。 |
| `500 Internal Server Error` | `internal_error` | 支持的 `type` 在数据库中缺失，或数据库/未知服务端错误。 |

支持的 `type` 缺失属于后端配置错误，不应返回 `404` 让前端理解成用户请求了不存在资源。实现和测试中应把它视为 backend consistency failure。

示例错误：

```json
{
  "error": {
    "code": "invalid_request",
    "message": "unsupported legal document type",
    "details": [],
    "request_id": "req_..."
  }
}
```

## 8. Implementation Status

当前后端实现已落地：

1. User module migration：`internal/user/infrastructure/migration/000003_create_legal_documents.up.sql` / `.down.sql`。
2. User persistence query：按 `document_type` 读取 legal document。
3. User application：DTO、repository port 和 `GetLegalDocument` usecase。
4. API handler：`legaldocuments` endpoint group。
5. Server wiring：`cmd/server/wiring.go` 装配 User repository/usecase 和 handler。
6. Router：`internal/api/infrastructure/http/router/router.go` 注册 endpoint group。

部署侧 OpenAPI config 仍必须为 `GET /api/legal-documents/{type}` 设置 operation-level `security: []`。如果部署配置不在本仓库，发布时必须同步对应部署仓库，否则生产无 token 请求会被 Gateway 拦截，后端 handler 不会收到请求。

## 9. Test Requirements

后端实现必须覆盖：

- User usecase 单测：支持的 type 返回 DTO；非法 type 返回 validation error。
- User repository integration：migration 后能读取 seed 的 `privacy-policy` 和 `user-agreement`。
- API integration：无 principal 请求 `GET /api/legal-documents/privacy-policy` 返回 `200 OK`。
- API integration：无 principal 请求非法 type 返回 `400 invalid_request`。
- API integration：handler 不要求 `X-Apigateway-Api-Userinfo`。

验收命令：

```text
make quick-check
make check
```

如果只写本文档、不改代码，则不需要执行 `make check`。
