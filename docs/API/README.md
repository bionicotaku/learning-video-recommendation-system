# API 设计文档索引

本目录统一收口 API 设计、已实现契约和业务语义文档。

## 入口文档

- [API契约归总.md](API契约归总.md)：当前已实现 26 个业务 endpoint 的 method/path、认证、请求字段、响应字段、状态码、错误、validation、幂等和副作用。查看具体 API 契约优先读这里。
- [API模块总体设计规范.md](API模块总体设计规范.md)：`internal/api` 的统一入口规范，包括 principal、JSON、错误 envelope、handler 边界和测试要求。

## 业务文档

| 文档 | 覆盖范围 |
|---|---|
| [Feed-API-MVP设计.md](Feed-API-MVP设计.md) | `POST /api/feed` 的推荐流展示契约和 API facade 边界。 |
| [Video-Detail-API-MVP设计.md](Video-Detail-API-MVP设计.md) | `GET /api/videos/{video_id}` 的播放页详情读取契约。 |
| [Video-Favorites-API-MVP设计.md](Video-Favorites-API-MVP设计.md) | `GET /api/video-favorites` 的收藏列表分页契约。 |
| [Video-History-API-MVP设计.md](Video-History-API-MVP设计.md) | `GET /api/video-history` 的观看历史分页契约。 |
| [End-Quiz-批量取题API-MVP设计.md](End-Quiz-批量取题API-MVP设计.md) | `POST /api/videos/end-quiz` 的视频末尾批量取题契约。 |
| [Catalog-观看进度上报MVP设计.md](Catalog-观看进度上报MVP设计.md) | `POST /api/video-watch-progress` 的观看进度上报契约。 |
| [Video-Interactions-API-MVP设计.md](Video-Interactions-API-MVP设计.md) | 视频 like / favorite 四个幂等互动接口。 |
| [Word-Favorite-API-MVP设计.md](Word-Favorite-API-MVP设计.md) | word favorite 状态、收藏、取消收藏和分页列表四个接口。 |
| [学习事件上报API设计.md](学习事件上报API设计.md) | learning interactions batch、quiz attempt、self mark mastered、reset unlearned 四个学习事件写入口。 |
| [Unit-Progress-API-MVP设计.md](Unit-Progress-API-MVP设计.md) | mastered / unmastered 两个学习进度分页读取接口。 |
| [Unit-Collections-API-MVP设计.md](Unit-Collections-API-MVP设计.md) | `GET /api/unit-collections` 和 `PUT /api/learning-targets/active-collection` 的词书读取 / 激活语义。 |
| [Active-Learning-Targets-API-MVP设计.md](Active-Learning-Targets-API-MVP设计.md) | `GET /api/learning-targets/active-coarse-unit-ids` 的 active target id 读取契约。 |
| [Me-API-MVP设计.md](Me-API-MVP设计.md) | `GET /api/me` 的 profile、stats、activity calendar 读取契约。 |
| [Me-Profile-Update-API-MVP设计.md](Me-Profile-Update-API-MVP设计.md) | `PATCH /api/me/profile` 的 profile 编辑契约。 |
| [User-Feedback-API-MVP设计.md](User-Feedback-API-MVP设计.md) | `POST /api/feedback` 的 multipart 反馈上传契约。 |

## 维护规则

- 新增或修改已实现 endpoint 时，同步更新 [API契约归总.md](API契约归总.md)。
- 业务背景、owner 边界、非目标范围和前端样例写在对应业务文档；不要在本 README 重复 endpoint 详情。
- 新增 API 设计文档统一放在本目录，并在上表加入链接。
- 当前未维护完整 OpenAPI / 客户端 SDK；如后续落地，也从本目录链接。
