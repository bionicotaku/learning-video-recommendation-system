// 作用：定义 coarse unit 的类型枚举，表达 word/phrase/grammar 三类学习单元。
// 输入/输出：输入无；输出是 UnitKind 类型和相关常量。
// 谁调用它：domain/model/coarse_unit_ref.go、测试夹具。
// 它调用谁/传给谁：不主动调用其他文件；值会传给 CoarseUnitRef 或测试数据构造。
package enum

type UnitKind string

const (
	UnitKindWord    UnitKind = "word"
	UnitKindPhrase  UnitKind = "phrase"
	UnitKindGrammar UnitKind = "grammar"
)
