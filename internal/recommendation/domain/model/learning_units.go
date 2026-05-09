package model

type LearningRole string

const (
	LearningRoleHardReview LearningRole = "hard_review"
	LearningRoleNewNow     LearningRole = "new_now"
	LearningRoleSoftReview LearningRole = "soft_review"
	LearningRoleNearFuture LearningRole = "near_future"
)

type LearningUnitEvidence struct {
	SentenceIndex *int32 `json:"sentence_index,omitempty"`
	SpanIndex     *int32 `json:"span_index,omitempty"`
	StartMs       *int32 `json:"start_ms,omitempty"`
	EndMs         *int32 `json:"end_ms,omitempty"`
}

type ExpectedLearningUnit struct {
	CoarseUnitID int64                 `json:"coarse_unit_id"`
	Role         LearningRole          `json:"role"`
	IsPrimary    bool                  `json:"is_primary"`
	Evidence     *LearningUnitEvidence `json:"evidence,omitempty"`
}

func LearningUnitIDs(units []ExpectedLearningUnit) []int64 {
	seen := make(map[int64]struct{}, len(units))
	result := make([]int64, 0, len(units))
	for _, unit := range units {
		if _, ok := seen[unit.CoarseUnitID]; ok {
			continue
		}
		seen[unit.CoarseUnitID] = struct{}{}
		result = append(result, unit.CoarseUnitID)
	}
	return result
}

func LearningUnitIDsByRole(units []ExpectedLearningUnit, role LearningRole) []int64 {
	result := make([]int64, 0, len(units))
	for _, unit := range units {
		if unit.Role == role {
			result = append(result, unit.CoarseUnitID)
		}
	}
	return result
}

func LearningUnitIDsByRoles(units []ExpectedLearningUnit, roles ...LearningRole) []int64 {
	roleSet := make(map[LearningRole]struct{}, len(roles))
	for _, role := range roles {
		roleSet[role] = struct{}{}
	}

	seen := make(map[int64]struct{}, len(units))
	result := make([]int64, 0, len(units))
	for _, unit := range units {
		if _, ok := roleSet[unit.Role]; !ok {
			continue
		}
		if _, ok := seen[unit.CoarseUnitID]; ok {
			continue
		}
		seen[unit.CoarseUnitID] = struct{}{}
		result = append(result, unit.CoarseUnitID)
	}
	return result
}

func PrimaryLearningUnitIDs(units []ExpectedLearningUnit) []int64 {
	seen := make(map[int64]struct{}, len(units))
	result := make([]int64, 0, len(units))
	for _, unit := range units {
		if !unit.IsPrimary {
			continue
		}
		if _, ok := seen[unit.CoarseUnitID]; ok {
			continue
		}
		seen[unit.CoarseUnitID] = struct{}{}
		result = append(result, unit.CoarseUnitID)
	}
	return result
}

func CountLearningUnitsByRole(units []ExpectedLearningUnit, role LearningRole) int {
	return len(LearningUnitIDsByRole(units, role))
}

func HasLearningRole(units []ExpectedLearningUnit, role LearningRole) bool {
	for _, unit := range units {
		if unit.Role == role {
			return true
		}
	}
	return false
}

func IsCoreLearningRole(role LearningRole) bool {
	return role == LearningRoleHardReview || role == LearningRoleNewNow
}

func IsFutureLikeLearningRole(role LearningRole) bool {
	return role == LearningRoleSoftReview || role == LearningRoleNearFuture
}

func PrimaryLearningUnitEvidence(units []ExpectedLearningUnit) *LearningUnitEvidence {
	for _, unit := range units {
		if unit.IsPrimary && unit.Evidence != nil {
			return unit.Evidence
		}
	}
	for _, unit := range units {
		if unit.Evidence != nil {
			return unit.Evidence
		}
	}
	return nil
}
