# User Feedback API MVP 设计

## 0. 文档信息

文档状态：MVP 已实现，作为当前 API 契约维护。
目标读者：前端、后端 API、User 模块、数据库维护者。
当前范围：定义用户反馈上传 API 的请求格式、5 MiB 总请求限制、图片校验、User 模块归属、`app_user.feedback_*` 存储模型、错误语义和测试要求。
当前明确不做：不做客服工单状态流、不做运营后台、不做图片对象存储、不做前端直传、不做反馈列表读取、不做反馈删除、不做反馈回复。

关联文档：

- [API模块总体设计规范.md](API模块总体设计规范.md)
- [Me-API-MVP设计.md](Me-API-MVP设计.md)
- [../User模块设计.md](../User模块设计.md)

## 1. API 定位

`POST /api/feedback` 接收当前登录用户的一次产品反馈。反馈包含一个前端自定义 JSON payload，以及最多 5 张 JPEG 图片。

MVP 把 feedback 归到 User 模块：

```text
internal/api feedback.Handler
  -> user.SubmitFeedbackUsecase
  -> app_user.feedback_submissions
  -> app_user.feedback_images
```

这个归属表示 feedback 是用户相关的支持/反馈数据，不表示它属于 `/api/me`、profile、onboarding 或 activity stats。User 模块可以拥有这组表，但现有 profile 与统计读取路径不得读取 feedback。

本 API 从 trusted principal 获取 `user_id`，不接受 body、query、path 或普通 header 中的用户身份。

## 2. Endpoint

```http
POST /api/feedback
Content-Type: multipart/form-data
```

认证：必需。

请求字段：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `payload` | string | 是 | JSON object 字符串。字段集合由前端自定义，后端只校验它是 object。 |
| `images` | file[] | 否 | 重复 multipart file field。最多 5 个，只接受 JPEG。 |
| `client_feedback_id` | string UUID | 否 | 前端生成的幂等 ID，用于失败重试去重。 |

`images` 使用同名字段重复传入：

```text
payload={"type":"bug","screen":"feed","message":"subtitle is wrong","app_version":"1.3.0"}
client_feedback_id=11111111-1111-1111-1111-111111111111
images=@screenshot-1.jpg
images=@screenshot-2.jpg
```

`payload` 必须是 JSON object。下面这些值都不是合法 payload：

```json
[]
"text"
123
null
```

空 object 合法：

```json
{}
```

## 3. Size Limit

本 API 的总请求大小硬限制为 `5 MiB`：

```text
5 * 1024 * 1024 = 5,242,880 bytes
```

限制口径是完整 HTTP request body，包括 multipart boundary、field header、`payload`、`client_feedback_id` 和所有图片文件内容。

超过 5 MiB 必须返回：

```text
413 Payload Too Large
code = payload_too_large
```

当前 `cmd/server` 已有全局 `BodyLimit(1 << 20)`。实现本 API 时必须调整为 route-aware body limit：

```text
/api/feedback: 5 MiB
其他现有 JSON endpoint: 保持 1 MiB 默认限制
```

不能只在 feedback handler 内部扩大限制，因为全局 1 MiB `MaxBytesReader` 会先截断请求。

MVP 不定义单张图片独立公网契约；单张图片大小由 5 MiB 总请求限制、最多 5 张图片和后端 JPEG 解析共同约束。前端应在上传前压缩截图，推荐最长边不超过 1280px，JPEG quality 使用 0.75 左右。

## 4. Image Validation

每个 `images` 文件必须满足：

- field name 是 `images`；
- 文件数量 `0 <= count <= 5`；
- multipart file header 的 `Content-Type` 是 `image/jpeg` 或 `image/jpg`；
- 文件内容 magic bytes 符合 JPEG，至少以 `FF D8 FF` 开头；
- Go `image/jpeg.DecodeConfig` 能解析宽高；
- 解码后的 width / height 均大于 0。

后端不能只信任文件名或 multipart `Content-Type`。校验通过后，数据库统一保存：

```text
content_type = image/jpeg
```

MVP 不接受 PNG、WebP、HEIC、GIF 或视频。前端如果拿到非 JPEG 截图，需要先转成 JPEG 再上传。

## 5. Response

成功响应使用 `200 OK`，和当前写入类 API 的风格保持一致。

```ts
type SubmitFeedbackResponse = {
  feedback_id: string;
  accepted: true;
  image_count: number;
  created_at: string;
};
```

示例：

```json
{
  "feedback_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  "accepted": true,
  "image_count": 2,
  "created_at": "2026-05-22T18:30:00Z"
}
```

如果请求携带 `client_feedback_id`，并且同一用户此前已成功提交过相同 `client_feedback_id`，重试请求返回已有 submission 的 `feedback_id`、`image_count` 和 `created_at`，仍使用 `200 OK`。重复请求不新增 submission，也不重复写入图片。

## 6. Storage Model

MVP 直接把图片二进制写入 Postgres `bytea`，不存 base64 原文。`payload` 与图片分表，避免把大二进制混进 JSON。

建议表结构：

```sql
create table app_user.feedback_submissions (
  id uuid primary key,
  user_id uuid not null references auth.users(id) on delete cascade,
  client_feedback_id uuid,
  payload jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),

  unique (user_id, client_feedback_id),
  check (jsonb_typeof(payload) = 'object')
);

create table app_user.feedback_images (
  id uuid primary key,
  submission_id uuid not null references app_user.feedback_submissions(id) on delete cascade,
  sort_order int not null,
  content_type text not null,
  size_bytes int not null,
  sha256 text not null,
  width int not null,
  height int not null,
  image_data bytea not null,
  created_at timestamptz not null default now(),

  unique (submission_id, sort_order),
  check (sort_order between 1 and 5),
  check (content_type = 'image/jpeg'),
  check (size_bytes > 0),
  check (width > 0),
  check (height > 0)
);

create index idx_feedback_submissions_user_created_at_desc
on app_user.feedback_submissions (user_id, created_at desc);
```

`sort_order` 按前端上传顺序从 1 开始。`sha256` 用于排障、重复图片识别和未来迁移对象存储时做校验。

`app_user.feedback_*` 是 MVP 归置。后续如果 feedback 演进为工单、运营审核、客服回复或高频图片存储，应拆成独立 `feedback` schema / module，并把图片迁移到 GCS 或 Supabase Storage，Postgres 只保留 object key 和 metadata。

## 7. Transaction Boundary

一次反馈提交必须是原子写入：

```text
1. 校验 request、payload 和所有图片。
2. 开启数据库事务。
3. 写入 app_user.feedback_submissions。
4. 按上传顺序写入 app_user.feedback_images。
5. 提交事务。
```

成功边界：

```text
200 OK 表示 submission 和所有合法图片已经写入成功。
```

失败边界：

```text
任一校验或数据库写入失败，submission 与 images 都不得留下半成品。
```

因为 MVP 不使用对象存储，所以暂不需要处理 DB 成功但对象上传失败的补偿清理。

## 8. Error Semantics

错误响应使用 API 模块统一 JSON error envelope。

| HTTP | code | 场景 |
|---|---|---|
| `200 OK` | - | 新反馈写入成功，或同一 `client_feedback_id` 的幂等重试返回已有结果。 |
| `400 Bad Request` | `invalid_request` | `payload` 缺失、不是合法 JSON、不是 JSON object；`client_feedback_id` 不是 UUID；图片数量超过 5；图片字段名错误；图片不是 JPEG；JPEG 无法解析宽高。 |
| `401 Unauthorized` | `unauthorized` | trusted principal 缺失。 |
| `413 Payload Too Large` | `payload_too_large` | 完整 multipart request body 超过 5 MiB。 |
| `500 Internal Server Error` | `internal_error` | 数据库或未知服务端错误。 |

示例错误：

```json
{
  "error": {
    "code": "invalid_request",
    "message": "payload must be a JSON object",
    "details": [],
    "request_id": "req_..."
  }
}
```

## 9. Frontend Calling Notes

前端应使用 `multipart/form-data`，不要手动设置 multipart boundary。浏览器 / React Native fetch 示例语义：

```ts
const form = new FormData();
form.append("payload", JSON.stringify({
  type: "bug",
  screen: "feed",
  message: "subtitle is wrong",
  app_version: "1.3.0"
}));
form.append("client_feedback_id", clientFeedbackId);
form.append("images", imageFile1);
form.append("images", imageFile2);

await fetch(`${API_BASE_URL}/feedback`, {
  method: "POST",
  headers: {
    Authorization: `Bearer ${token}`
  },
  body: form
});
```

前端上传前应尽量压缩图片，避免触发 5 MiB 限制。`payload` 字段由前端决定，后端不会解析其中业务字段，也不会把 `payload.user_id` 作为可信身份。

## 10. 不做事项

MVP 不做：

- `GET /api/feedback` 列表读取；
- `GET /api/feedback/{id}` 详情读取；
- 图片下载 API；
- 客服状态、处理人、回复、关闭、优先级；
- 前端直传对象存储；
- base64 JSON 上传；
- `public` schema 表；
- 从 request body 信任 `user_id`；
- 把 feedback 混入 `/api/me` response；
- 写 Analytics raw fact 或 Learning Engine event。

## 11. 当前实现入口

目标实现入口：

| 层 | 目标位置 |
|---|---|
| HTTP handler | `internal/api/infrastructure/http/handler/feedback` |
| Server wiring | `cmd/server/wiring.go` |
| API route registration | `internal/api/infrastructure/http/router/router.go` |
| User DTO | `internal/user/application/dto/feedback.go` |
| User usecase | `internal/user/application/service/submit_feedback.go` |
| Repository port | `internal/user/application/repository/feedback_writer.go` |
| SQL | `internal/user/infrastructure/persistence/query/feedback.sql` |
| Repository impl | `internal/user/infrastructure/persistence/repository/feedback_writer.go` |
| Migration | `internal/user/infrastructure/migration` |

实现已同步更新：

- `internal/user/README.md`：补充 User 模块拥有用户反馈提交数据。
- `internal/user/infrastructure/migration/README.md`：补充 `app_user.feedback_submissions` 和 `app_user.feedback_images`。
- `internal/api/README.md`：补充 `POST /api/feedback`。
- `.env.example`：不需要新增配置。

## 12. 测试要求

目标测试：

```bash
go test -tags=integration ./internal/api/test/integration/feedback
go test ./internal/user/test/unit/application/service -run Feedback
go test ./internal/user/test/integration/repository -tags=integration -run Feedback
```

验收点：

- valid `payload` + 0 张图片提交成功。
- valid `payload` + 5 张 JPEG 提交成功，`image_count = 5`。
- 第 6 张图片返回 `400 invalid_request`。
- 非 JSON payload 返回 `400 invalid_request`。
- JSON array / string / number / null payload 返回 `400 invalid_request`。
- 非 JPEG 文件返回 `400 invalid_request`。
- JPEG content type 正确但 magic bytes 错误返回 `400 invalid_request`。
- multipart request body 超过 5 MiB 返回 `413 payload_too_large`。
- missing principal 返回 `401 unauthorized`。
- 同一用户重复 `client_feedback_id` 返回同一个 `feedback_id`，不重复写图片。
- 不同用户可以使用相同 `client_feedback_id`。
- 任一图片校验失败时，不写 submission，也不写任何 image。
- repository integration 验证 `payload` 存为 JSON object，图片存为 `bytea`，`sort_order` 按上传顺序递增。
