# Catalog Migrations

这个目录只保留 Catalog 的最终 clean baseline。

它只定义：

- Catalog 自己的 schema
- Catalog 自己的表
- Catalog 自己的索引

它不定义：

- Learning engine 的表
- Recommendation 的表
- 任何历史兼容 patch migration
- 任何 archive 改造过程

要求：

- 从空库执行这里的 migration head，得到的必须是当前最终版 Catalog schema
- live DB 的 `catalog` schema 必须与这里的 head 一致
- 如果需要修正现网历史库，使用一次性临时 SQL；不要把中间过程写回正式 migration 历史
