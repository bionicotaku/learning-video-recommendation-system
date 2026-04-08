// 文件作用：
//   - 集中封装 pgx/pgtype 与 Go 基础类型之间的互转逻辑
//   - 避免 repository 和业务层直接处理 UUID、Numeric、Timestamptz 的底层细节
//
// 输入/输出：
//   - 输入：pgtype 值或 Go 基础值
//   - 输出：对应的 Go 值、pgtype 值或转换错误
//
// 谁调用它：
//   - candidate_mapper.go
//   - scheduler_run_mapper.go
//   - user_unit_serving_state_mapper.go
//   - repository 层通过这些 mapper 间接依赖它
//
// 它调用谁/传给谁：
//   - 调用 pgtype 自身的 Scan / Float64Value 等方法
//   - 把转换结果传给 mapper 和 sqlc 参数构造流程
package mapper

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func requiredUUID(value pgtype.UUID, field string) (uuid.UUID, error) {
	if !value.Valid {
		return uuid.Nil, fmt.Errorf("%s is invalid", field)
	}

	return uuid.UUID(value.Bytes), nil
}

func optionalUUID(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}

	id := uuid.UUID(value.Bytes)
	return &id
}

func UUIDToPG(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: id != uuid.Nil}
}

func OptionalUUIDToPG(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}

	return pgtype.UUID{Bytes: [16]byte(*id), Valid: true}
}

func requiredTime(value pgtype.Timestamptz, field string) (time.Time, error) {
	if !value.Valid {
		return time.Time{}, fmt.Errorf("%s is invalid", field)
	}

	return value.Time, nil
}

func optionalTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	t := value.Time
	return &t
}

func TimeToPG(value time.Time) pgtype.Timestamptz {
	if value.IsZero() {
		return pgtype.Timestamptz{}
	}

	return pgtype.Timestamptz{Time: value, Valid: true}
}

func OptionalTimeToPG(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}

	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func requiredFloat(value pgtype.Numeric, field string) (float64, error) {
	if !value.Valid {
		return 0, fmt.Errorf("%s is invalid", field)
	}

	f, err := value.Float64Value()
	if err != nil {
		return 0, fmt.Errorf("%s float conversion: %w", field, err)
	}
	if !f.Valid {
		return 0, fmt.Errorf("%s float conversion is invalid", field)
	}

	return f.Float64, nil
}

func floatToPG(value float64) (pgtype.Numeric, error) {
	var numeric pgtype.Numeric
	if err := numeric.Scan(strconv.FormatFloat(value, 'f', -1, 64)); err != nil {
		return pgtype.Numeric{}, err
	}

	return numeric, nil
}

func optionalInt(value pgtype.Int2) *int {
	if !value.Valid {
		return nil
	}

	v := int(value.Int16)
	return &v
}

func textFromPG(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}

	return value.String
}

func intsFromPG(values []int16) []int {
	if len(values) == 0 {
		return []int{}
	}

	out := make([]int, 0, len(values))
	for _, value := range values {
		out = append(out, int(value))
	}

	return out
}

func boolsFromPG(values []bool) []bool {
	if len(values) == 0 {
		return []bool{}
	}

	return append([]bool(nil), values...)
}
