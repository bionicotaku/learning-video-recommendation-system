# Recommendation Migrations

这个目录是 Recommendation 唯一合法的 migration 根。

最终应只定义：

- Recommendation 自己的 schema
- Recommendation 自己的表
- Recommendation 自己的索引

不负责：

- Learning engine 的表
- Learning engine 的索引
- 任何旧 `scheduler` 兼容 migration

执行顺序：

- 整库初始化时建议在 Catalog、Learning engine 之后执行
- 建议统一通过仓库根的 `make migrate-up` / `make migrate-down` 编排
