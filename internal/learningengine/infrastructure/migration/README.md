# Learning Engine Migrations

这个目录是 Learning engine 唯一合法的 migration 根。

最终应只定义：

- Learning engine 自己的 schema
- Learning engine 自己的表
- Learning engine 自己的索引

不负责：

- Recommendation 的表
- Recommendation 的索引
- 任何旧 `scheduler` 兼容 migration
