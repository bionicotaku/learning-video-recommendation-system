# Catalog

`catalog` 负责内容资产、内容索引层，以及用户对视频的消费状态投影。

当前模块边界包括：

- 切片视频内容资产主记录
- transcript 元数据、时间定位与索引模型
- Recall-ready 的视频级 coarse unit 索引
- 单视频入库审计
- 用户对视频的互动状态投影
- 观看进度上报命令 `RecordVideoWatchProgress`

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
