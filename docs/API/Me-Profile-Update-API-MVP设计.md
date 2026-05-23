# Me Profile Update API MVP 设计

## 0. 文档信息

文档状态：MVP 已实现，作为当前 API 契约维护。
目标读者：前端、后端 API、User、Supabase Auth 维护者。
当前范围：定义当前用户 profile 可编辑字段、请求校验、响应契约和模块边界。
当前明确不做：不修改邮箱、不上传头像、不修改 locale、不写 `ip_region`、不更新活动统计。

关联文档：

- [Me-API-MVP设计.md](Me-API-MVP设计.md)
- [../User模块设计.md](../User模块设计.md)
- [API模块总体设计规范.md](API模块总体设计规范.md)

## 1. API 定位

`PATCH /api/me/profile` 用于修改当前登录用户的产品资料字段。它只更新 `app_user.user_profiles` 中允许前端维护的 profile 字段，不读取或写入学习、推荐、视频、反馈等业务表。

本接口从 trusted principal 获取 `user_id`，不接受 body/query/path 中的用户 ID。

## 2. Endpoint

```http
PATCH /api/me/profile
Content-Type: application/json
```

## 3. Request

```ts
type UpdateMeProfileRequest = {
  display_name?: string;
  birth_date?: string | null;
  gender?: "male" | "female" | "other" | "prefer_not_to_say" | null;
  education_stage?:
    | "primary_school"
    | "middle_school"
    | "high_school"
    | "undergraduate"
    | "graduate"
    | "phd"
    | "working"
    | "other"
    | null;
  timezone?: string | null;
};
```

字段省略表示不修改。空 JSON object `{}` 返回 `400 invalid_request`。

## 4. 字段规则

| 字段 | 规则 |
|---|---|
| `display_name` | 不允许 `null`。trim 后必须是 2-20 个 Unicode 字符，只允许 Unicode 字母、Unicode 数字、下划线。 |
| `birth_date` | 格式必须是 `YYYY-MM-DD`，范围 `1900-01-01 <= birth_date <= today`；`null` 表示清空。 |
| `gender` | 只能是 `male`、`female`、`other`、`prefer_not_to_say`；`null` 表示清空。 |
| `education_stage` | 只能是 `primary_school`、`middle_school`、`high_school`、`undergraduate`、`graduate`、`phd`、`working`、`other`；`null` 表示清空。 |
| `timezone` | 必须是合法 IANA timezone name；`null` 表示清空。 |

`display_name` 的 Unicode 字母/数字规则包含中文、日语、韩语、法语等自然语言字符；不允许空格、emoji、标点和连字符。

与 `GET /api/me` 对非法 `X-Client-Timezone` 的容忍不同，`PATCH /api/me/profile` 中非法 `timezone` 必须返回 `400 invalid_request`，因为这是用户明确提交的设置。

## 5. Response

响应返回更新后的 profile 子集，不返回 stats 或 activity calendar。

```ts
type UpdateMeProfileResponse = {
  user_id: string;
  email: string | null;
  email_confirmed: boolean;
  display_name: string;
  avatar_url: string | null;
  locale: string;
  timezone: string | null;
  onboarding_status: "new" | "collection_selected" | "completed";
  birth_date: string | null;
  gender: "male" | "female" | "other" | "prefer_not_to_say" | null;
  education_stage:
    | "primary_school"
    | "middle_school"
    | "high_school"
    | "undergraduate"
    | "graduate"
    | "phd"
    | "working"
    | "other"
    | null;
  ip_region: string | null;
};
```

示例：

```json
{
  "user_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  "email": "alice@example.com",
  "email_confirmed": true,
  "display_name": "Alice_01",
  "avatar_url": null,
  "locale": "zh-CN",
  "timezone": "Asia/Shanghai",
  "onboarding_status": "collection_selected",
  "birth_date": "2001-09-01",
  "gender": "prefer_not_to_say",
  "education_stage": "primary_school",
  "ip_region": null
}
```

## 6. 不允许修改字段

本接口不接受以下字段：

```text
email
email_confirmed_at
avatar_url
locale
onboarding_status
ip_region
stats
activity_calendar
```

`email` 必须通过 Supabase Auth 修改，`app_user.user_profiles.email` 只是 Auth 派生缓存。`avatar_url` 后续通过头像上传 API 设计。`ip_region` MVP 只预留数据库字段，暂不由前端提交，也不由本接口写入。

## 7. 处理流程

```text
1. API auth middleware 从 trusted principal 解析 user_id。
2. Handler 校验 Content-Type 和 JSON body。
3. Handler 拒绝空 patch 和不允许修改字段。
4. Handler / User usecase 校验字段格式和值域。
5. User usecase 读取或 lazy repair profile。
6. User repository 更新允许字段，并返回更新后的 profile。
7. Handler 返回 UpdateMeProfileResponse。
```

如果 profile 缺失，行为应与 `GET /api/me` 一致：先从 `auth.users` lazy repair，再应用本次 patch。这样刚注册但 trigger 尚未补齐的用户也可以正常保存资料。

## 8. 错误

| HTTP | code | 场景 |
|---|---|---|
| `200 OK` | 无 | 更新成功，返回更新后的 profile 子集。 |
| `400 Bad Request` | `invalid_request` | JSON 非 object、空 patch、字段类型错误、字段值非法、包含不允许修改字段、非法 timezone。 |
| `401 Unauthorized` | `unauthorized` | trusted principal 缺失、无法解析，或 principal 指向不存在的 Auth user。 |
| `500 Internal Server Error` | `internal_error` | 数据库错误或未知服务端错误。 |

## 9. 数据库访问

目标表：

```text
app_user.user_profiles
```

更新语义：

```sql
update app_user.user_profiles
set
  display_name = coalesce(sqlc.narg(display_name), display_name),
  birth_date = case when sqlc.arg(set_birth_date)::bool then sqlc.narg(birth_date)::date else birth_date end,
  gender = case when sqlc.arg(set_gender)::bool then sqlc.narg(gender)::text else gender end,
  education_stage = case when sqlc.arg(set_education_stage)::bool then sqlc.narg(education_stage)::text else education_stage end,
  timezone = case when sqlc.arg(set_timezone)::bool then sqlc.narg(timezone)::text else timezone end,
  updated_at = now()
where user_id = sqlc.arg(user_id)
returning ...;
```

实现时可以按仓库 sqlc 习惯拆成更清晰的 query。关键是要区分“字段省略”和“字段显式传 `null`”。

## 10. 测试计划

目标测试：

```bash
go test ./internal/user/...
go test ./internal/api/test/integration/me
make quick-check
```

覆盖场景：

- 缺 principal 返回 `401 unauthorized`。
- `{}` 返回 `400 invalid_request`。
- `display_name: null` 返回 `400 invalid_request`。
- `display_name` 支持中文、日语、韩语、法语字母和数字。
- `display_name` 拒绝空格、emoji、标点、连字符和长度不合法。
- `birth_date` 支持合法日期和 `null` 清空，拒绝未来日期和早于 `1900-01-01` 的日期。
- `gender` 和 `education_stage` 支持合法枚举和 `null` 清空，拒绝未知值。
- `timezone` 支持合法 IANA timezone 和 `null` 清空，拒绝非法 timezone。
- 不允许提交 `email`、`avatar_url`、`locale`、`onboarding_status`、`ip_region`、`stats`、`activity_calendar`。
- profile 缺失时先 lazy repair，再应用 patch。
