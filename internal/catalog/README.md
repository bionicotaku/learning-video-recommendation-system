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
- Feed facade 使用的批量读取能力：视频列表 preview 字段、unit label
- Video Detail 使用的单视频详情读取能力：播放资源、transcript 元数据、互动统计、当前用户状态
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

watch-progress 写入路径使用数据库内条件 upsert，不在 application 层 pre-read `analytics.video_watch_events`。同一 `watch_session_id` 首次并发上报时，repository 只做一次内部重试，让第二次 SQL 语句读取已存在 session 后继续由数据库侧计算 delta；同一 `watch_session_id` 绑定不同用户或视频时返回 conflict；不存在的视频返回 not found。普通观看进度不写 Learning Engine，也不写 Recommendation serving state。

Video Interactions 写入路径维护当前用户对视频的点赞和收藏状态：

```text
PUT/DELETE /api/videos/{video_id}/like
PUT/DELETE /api/videos/{video_id}/favorite
  -> internal/api
  -> catalog.SetVideoLike / catalog.SetVideoFavorite
  -> catalog.video_user_states
  -> catalog.video_engagement_stats
```

这两个命令都是幂等 set / unset，不做 toggle。Repository 在同一事务内更新 `catalog.video_user_states.has_liked` / `has_bookmarked` 与 `catalog.video_engagement_stats.like_count` / `favorite_count`。重复 set 不重复增加计数，重复 unset 不重复减少计数；unset 没有状态行时不创建空的 user state 行。MVP 不新增点赞/收藏审计表，不写 Analytics，不写 Learning Engine，也不写 Recommendation。

Feed lookup 是只读能力，服务 `POST /api/feed` 的 facade 组装：

```text
internal/api FeedService
  -> catalog.FeedVideoLookupUsecase
  -> catalog.FeedLookupReader.ListFeedVideosByIDs
  -> catalog.videos + catalog.video_engagement_stats

internal/api FeedService
  -> catalog.UnitLabelLookupUsecase
  -> catalog.FeedLookupReader.ListUnitLabelsByIDs
  -> semantic.coarse_unit
```

`ListFeedVideosByIDs` 只返回可展示视频：`catalog.videos.status = active`、`visibility_status = public`、且 `publish_at` 为空或已发布。它只返回 Feed preview 需要的 `title`、`thumbnail_url` 和 `view_count`；互动统计缺行时 `view_count` 返回 `0`。

Video Detail lookup 是只读能力，服务 `GET /api/videos/{video_id}`：

```text
internal/api VideoDetailService
  -> catalog.GetVideoDetailUsecase
  -> catalog.FeedLookupReader.GetVideoDetailByID
  -> catalog.videos + catalog.video_transcripts + catalog.video_engagement_stats + catalog.video_user_states
```

`GetVideoDetailByID` 使用同一套可展示视频 predicate。Transcript 元数据缺行时 `transcript_object_path` 返回空；互动统计缺行时 `view_count`、`like_count`、`favorite_count` 返回 `0`；当前用户没有 `catalog.video_user_states` 行时 `has_liked`、`has_favorited` 返回 `false`。

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
