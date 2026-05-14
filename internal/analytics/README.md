# Analytics

`analytics` 负责保存前端和产品交互产生的原始事实。

当前模块边界包括：

- 习题 / 练习答题原始事实

当前已落地结构：

```text
internal/analytics/
  README.md
  doc.go
  infrastructure/
    migration/
```

当前阶段 `analytics` 只先落 migration。
后续如果补充上报 API、normalizer 读取或查询实现，再继续按统一结构扩展到 `application / domain / infrastructure / test`。
