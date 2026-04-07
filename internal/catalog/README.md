# Catalog

`catalog` 负责内容资产与内容索引层。

当前模块边界包括：

- 切片视频内容资产主记录
- transcript 标准化读模型
- Recall-ready 的视频级 coarse unit 索引
- 单视频入库审计
- 用户对视频的互动状态投影

当前已落地结构：

```text
internal/catalog/
  README.md
  infrastructure/
    migration/
```

当前阶段 `catalog` 只先落 migration。
后续如果补充导入流程、repository 或查询实现，再继续按统一结构扩展到 `application / domain / infrastructure / test`。
