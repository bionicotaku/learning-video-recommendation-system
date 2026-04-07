# Catalog Migrations

这个目录是 Catalog 唯一合法的 migration 根。

最终应只定义：

- Catalog 自己的 schema
- Catalog 自己的表
- Catalog 自己的索引

不负责：

- Learning engine 的表
- Recommendation 的表
- 任何旧 `videos` 流水线兼容 migration

执行顺序：

- 在整库初始化场景下，Catalog 应最先执行
- 建议统一通过仓库根的 `make migrate-up` / `make migrate-down` 编排
