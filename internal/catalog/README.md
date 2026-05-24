# Catalog

`catalog` 负责内容资产、内容索引层，以及用户对视频的消费状态投影。

当前模块边界包括：

- 切片视频内容资产主记录
- transcript 元数据、时间定位与索引模型
- Recall-ready 的视频级 coarse unit 索引
- 单视频入库审计
- 用户对视频的互动状态投影
- 观看进度上报命令 `RecordVideoWatchProgress`
- 视频点赞/收藏 set-unset 命令 `SetVideoLike` / `SetVideoFavorite`
- Word Favorite 状态、set/unset 和分页列表能力
- Feed facade 使用的批量读取能力：视频列表 preview 字段、unit label
- Video Detail 使用的单视频详情读取能力：播放资源、transcript 元数据、互动统计、当前用户状态
- Video Favorites / Video History 使用的只读分页列表能力：当前收藏、当前观看历史 preview
- End Quiz 使用的批量取题能力：视频上下文题优先，通用 unit 题 fallback

当前已落地结构：

```text
internal/catalog/
  README.md
  doc.go
  application/
    dto/
    repository/
    service/
    usecase/
  domain/
    model/
  infrastructure/
    migration/
    persistence/
      mapper/
      query/
      repository/
      schema/
      sqlcgen/
      sqlc.yaml
  test/
    fixture/
    unit/
    integration/
```

当前已实现 `RecordVideoWatchProgress`。该用例维护一次视频观看 session 的低频摘要和 Catalog 消费状态投影：

```text
POST /api/video-watch-progress
  -> internal/api
  -> catalog.RecordVideoWatchProgress
  -> analytics.video_watch_events
  -> catalog.video_user_states
  -> catalog.video_engagement_stats
```

`analytics.video_watch_events` 仍属于 Analytics schema，但在 watch-progress 命令中作为 session ledger 与 Catalog 投影同事务维护。Catalog repository 可以在这个命令内写入该表；这是为了保证 `watch_count`、`completed_count` 和 `total_watch_ms` 的去重依据与投影更新保持原子一致，不表示 Catalog 泛化拥有 Analytics raw fact 表。

watch-progress 写入路径使用数据库内条件 upsert，不在 application 层 pre-read `analytics.video_watch_events`。`watch_session_id` 是前端生成的 session correlation key；同一 `user_id + watch_session_id` 首次并发上报时，repository 只做一次内部重试，让第二次 SQL 语句读取已存在 session 后继续由数据库侧计算 delta；同一用户同一 `watch_session_id` 绑定不同视频时返回 conflict；不同用户偶然复用同一个 `watch_session_id` 不冲突；不存在的视频返回 not found。普通观看进度不写 Learning Engine，也不写 Recommendation serving state。

Video Interactions 写入路径维护当前用户对视频的点赞和收藏状态：

```text
PUT/DELETE /api/videos/{video_id}/like
PUT/DELETE /api/videos/{video_id}/favorite
  -> internal/api
  -> catalog.SetVideoLike / catalog.SetVideoFavorite
  -> catalog.video_user_states
  -> catalog.video_engagement_stats
```

这两个命令都是幂等 set / unset，不做 toggle。调用方必须提供客户端动作时间 `occurred_at`。Repository 在同一事务内更新 `catalog.video_user_states.has_liked` / `has_bookmarked`、对应的 `like_state_updated_at` / `favorite_state_updated_at` 水位，以及 `catalog.video_engagement_stats.like_count` / `favorite_count`。重复 set 不重复增加计数，重复 unset 不重复减少计数；旧 `occurred_at` 请求不会覆盖更新状态，也不会改动计数；unset 没有状态行时不创建空的 user state 行。MVP 不新增点赞/收藏审计表，不写 Analytics，不写 Learning Engine，也不写 Recommendation。

Word Favorite 维护当前用户对词 / 字幕 token 的收藏投影：

```text
POST /api/word-favorites/status
PUT /api/word-favorites
DELETE /api/word-favorites
GET /api/word-favorites
  -> internal/api
  -> catalog.WordFavorite usecases
  -> catalog.word_favorites
```

Word Favorite 使用 canonical key，而不是把前端点击文本作为持久化身份。`word_list + coarse_unit_id` 和 `video_transcript + coarse_unit_id` 都按 `user_id + coarse_unit_id` 收藏；`video_transcript + coarse_unit_id=null` 按 `user_id + video_id + sentence_index + token_index` 收藏。带 `coarse_unit_id` 的 transcript 请求会保存来源 token 字段用于列表展示，但不校验 token 是否实际映射到该 coarse unit。`PUT` / `DELETE` 都必须提供客户端动作时间 `occurred_at`；Repository 用 `state_updated_at` 做状态水位，旧请求 stale no-op，并且 stale `PUT` 与同一 `occurred_at` 的已生效 PUT 重试都会在目标存在性 / 可展示性校验之前被丢弃。`PUT` 生效时写 `is_favorited=true`，从 tombstone 恢复时用本次 `occurred_at` 作为 `favorited_at`；已收藏状态下较新同状态 PUT 只推进水位，不刷新列表排序时间。`DELETE` 不物理删除，而是写 `is_favorited=false`、`favorited_at=null` 的 tombstone，目标内容不存在时仍可返回成功。`catalog.word_favorites` 是用户状态投影，不对 `video_id` / `coarse_unit_id` 建内容 FK，保证 tombstone 能独立于内容生命周期挡住旧请求。该能力不写 Analytics、Learning Engine、Recommendation 或 User profile。

Feed lookup 是只读能力，服务 `POST /api/feed` 的 facade 组装：

```text
internal/api FeedService
  -> catalog.FeedVideoLookupUsecase
  -> catalog.VideoPresentationReader.ListFeedVideosByIDs
  -> catalog.videos + catalog.video_engagement_stats

internal/api FeedService
  -> catalog.UnitLabelLookupUsecase
  -> catalog.UnitLabelReader.ListUnitLabelsByIDs
  -> semantic.coarse_unit
```

`ListFeedVideosByIDs` 只返回可展示视频：`catalog.videos.status = active`、`visibility_status = public`、且 `publish_at` 为空或已发布。它只返回 Feed preview 需要的 `title`、`thumbnail_url` 和 `view_count`；互动统计缺行时 `view_count` 返回 `0`。

Video Detail lookup 是只读能力，服务 `GET /api/videos/{video_id}`：

```text
internal/api VideoDetailService
  -> catalog.GetVideoDetailUsecase
  -> catalog.VideoPresentationReader.GetVideoDetailByID
  -> catalog.videos + catalog.video_transcripts + catalog.video_engagement_stats + catalog.video_user_states
```

`GetVideoDetailByID` 使用同一套可展示视频 predicate。Transcript 元数据缺行时 `transcript_object_path` 返回空；互动统计缺行时 `view_count`、`like_count`、`favorite_count` 返回 `0`；当前用户没有 `catalog.video_user_states` 行时 `has_liked`、`has_favorited` 返回 `false`。

Video Library lookup 是只读分页能力，服务 `GET /api/video-favorites` 与 `GET /api/video-history`：

```text
internal/api VideoLibraryService.ListFavorites
  -> catalog.ListVideoFavoritesUsecase
  -> catalog.VideoLibraryReader.ListVideoFavorites
  -> catalog.video_user_states + catalog.videos + catalog.video_engagement_stats

internal/api VideoLibraryService.ListHistory
  -> catalog.ListVideoHistoryUsecase
  -> catalog.VideoLibraryReader.ListVideoHistory
  -> catalog.video_user_states + catalog.videos + catalog.video_engagement_stats
```

两个列表都使用 active/public/已发布 predicate，并使用 keyset cursor 分页。Favorites 读取 `has_bookmarked=true` 且 `bookmarked_at is not null` 的当前投影，按 `bookmarked_at desc, video_id asc` 排序；History 读取 `has_watched=true` 且 `last_watched_at is not null` 的当前投影，按 `last_watched_at desc, video_id asc` 排序。列表只返回 preview、`view_count` 和对应 metadata，不返回播放资源、transcript、description、like/favorite count 或当前用户互动状态。History 不直接读取 `analytics.video_watch_events` 热路径列表。

`ListUnitLabelsByIDs` 只补 `semantic.coarse_unit.status = active` 的 `label`。Catalog 在这里提供 lightweight read capability，是为了让 API facade 批量补齐展示文本；Catalog 不理解 Recommendation 的 role、rank、score，也不参与 quiz 选择。

End Quiz lookup 是只读能力，服务 `POST /api/videos/end-quiz`：

```text
internal/api endquiz.Handler
  -> catalog.EndQuizQuestionLookupUsecase
  -> catalog.EndQuizQuestionReader.HasVisibleVideoForEndQuiz
  -> catalog.videos

internal/api endquiz.Handler
  -> catalog.EndQuizQuestionLookupUsecase
  -> catalog.EndQuizQuestionReader.ListVideoUnitQuizQuestionCandidates
  -> catalog.questions

internal/api endquiz.Handler
  -> catalog.EndQuizQuestionLookupUsecase
  -> catalog.EndQuizQuestionReader.ListUnitQuizQuestionCandidates
  -> catalog.questions
```

`EndQuizQuestionLookupUsecase` 先校验视频是 active/public/已发布，再按请求中的 `coarse_unit_ids` 首次出现顺序去重。每个 unit 优先使用 `scope_type = 'video_unit'` 且匹配 `video_id` 的 active 题；没有合法视频上下文题时 fallback 到 `scope_type = 'unit'`、`video_id is null` 的 active 通用题。`content_payload` 必须包含非空 `question`、非空 options、每个 option 的 `id/text`，并且至少有一个 `id = correct`；坏候选会被跳过，最终无题的 unit 进入 `missing_coarse_unit_ids`。

End Quiz lookup 不写 quiz session、delivery、Analytics 或 Learning Engine。答题结果仍由 API 层的 `POST /api/quiz-attempts` 写入 Analytics，再由 normalizer 推进 Learning Engine。
