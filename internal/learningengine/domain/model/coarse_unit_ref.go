// 作用：定义 coarse unit 的轻量引用模型，用于在 Learning engine 内表达 unit 的展示性元数据。
// 输入/输出：输入通常来自 semantic.coarse_unit；输出是 CoarseUnitRef 结构。
// 谁调用它：当前主要由领域层和测试辅助使用，未来可供读模型或解释层扩展。
// 它调用谁/传给谁：不主动调用其他文件；实例会在需要附带 unit 元信息的链路中向外传递。
package model

import "learning-video-recommendation-system/internal/learningengine/domain/enum"

// CoarseUnitRef is a lightweight reference to a semantic.coarse_unit record.
type CoarseUnitRef struct {
	CoarseUnitID int64
	Kind         enum.UnitKind
	Label        string
	Pos          string
	EnglishDef   string
	ChineseDef   string
}
