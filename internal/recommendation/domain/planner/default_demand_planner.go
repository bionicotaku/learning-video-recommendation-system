package planner

import (
	"math"
	"sort"
	"time"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/domain/policy"
)

const (
	softReviewLookahead = 72 * time.Hour
)

type DefaultDemandPlanner struct{}

func NewDefaultDemandPlanner() *DefaultDemandPlanner {
	return &DefaultDemandPlanner{}
}

func (p *DefaultDemandPlanner) Plan(context model.RecommendationContext) (model.DemandBundle, error) {
	inventoryByUnit := make(map[int64]model.UnitVideoInventory, len(context.UnitInventory))
	for _, item := range context.UnitInventory {
		inventoryByUnit[item.CoarseUnitID] = item
	}

	bundle := model.DemandBundle{
		TargetVideoCount:     context.Request.TargetVideoCount,
		PreferredDurationSec: context.Request.PreferredDurationSec,
	}

	for _, state := range context.ActiveUnitStates {
		demandUnit := classifyDemandUnit(state, inventoryByUnit[state.CoarseUnitID], context.Now)
		switch demandUnit.Bucket {
		case string(policy.BucketHardReview):
			bundle.HardReview = append(bundle.HardReview, demandUnit)
		case string(policy.BucketNewNow):
			bundle.NewNow = append(bundle.NewNow, demandUnit)
		case string(policy.BucketSoftReview):
			bundle.SoftReview = append(bundle.SoftReview, demandUnit)
		case string(policy.BucketNearFuture):
			bundle.NearFuture = append(bundle.NearFuture, demandUnit)
		}
	}

	sortDemandUnits(bundle.HardReview)
	sortDemandUnits(bundle.NewNow)
	sortDemandUnits(bundle.SoftReview)
	sortDemandUnits(bundle.NearFuture)

	bundle.Flags = plannerFlags(bundle)
	bundle.SessionMode = plannerSessionMode(bundle)
	bundle.LaneBudget = plannerLaneBudget(bundle.Flags)
	bundle.MixQuota = plannerMixQuota(bundle.TargetVideoCount, bundle.Flags)

	return bundle, nil
}

func classifyDemandUnit(state model.LearningStateSnapshot, inventory model.UnitVideoInventory, now time.Time) model.DemandUnit {
	supplyGrade := inventory.SupplyGrade
	if supplyGrade == "" {
		supplyGrade = "none"
	}

	bucket := string(policy.BucketNearFuture)
	switch {
	case isHardReview(state, now):
		bucket = string(policy.BucketHardReview)
	case state.Status == "new":
		if supplyGrade == "none" {
			bucket = string(policy.BucketNearFuture)
		} else {
			bucket = string(policy.BucketNewNow)
		}
	case isSoftReview(state, now):
		bucket = string(policy.BucketSoftReview)
	default:
		bucket = string(policy.BucketNearFuture)
	}

	return model.DemandUnit{
		UnitID:       state.CoarseUnitID,
		Bucket:       bucket,
		Weight:       round4(bucketBaseWeight(bucket) + state.TargetPriority + supplyWeight(supplyGrade)),
		SupplyGrade:  supplyGrade,
		TargetWeight: state.TargetPriority,
	}
}

func isHardReview(state model.LearningStateSnapshot, now time.Time) bool {
	if state.LastQuality != nil && *state.LastQuality < 3 {
		return true
	}
	return state.NextReviewAt != nil && !state.NextReviewAt.After(now)
}

func isSoftReview(state model.LearningStateSnapshot, now time.Time) bool {
	if state.NextReviewAt != nil && !state.NextReviewAt.After(now.Add(softReviewLookahead)) {
		return true
	}
	if state.MasteryScore < 0.6 {
		return true
	}
	return state.LastQuality != nil && *state.LastQuality < 4
}

func plannerFlags(bundle model.DemandBundle) model.PlannerFlags {
	flags := model.PlannerFlags{}
	for _, unit := range bundle.HardReview {
		if unit.SupplyGrade == "weak" || unit.SupplyGrade == "none" {
			flags.HardReviewLowSupply = true
			break
		}
	}

	return flags
}

func plannerSessionMode(bundle model.DemandBundle) string {
	if len(bundle.HardReview) > 0 {
		return string(policy.SessionModeReviewHeavy)
	}
	if len(bundle.NewNow) > len(bundle.SoftReview)+len(bundle.NearFuture) {
		return string(policy.SessionModeExplore)
	}
	return string(policy.SessionModeBalanced)
}

func plannerLaneBudget(flags model.PlannerFlags) model.LaneBudget {
	if flags.HardReviewLowSupply {
		return model.LaneBudget{
			ExactCore:       0.35,
			Bundle:          0.35,
			SoftFuture:      0.20,
			QualityFallback: 0.10,
		}
	}

	return model.LaneBudget{
		ExactCore:       0.45,
		Bundle:          0.30,
		SoftFuture:      0.15,
		QualityFallback: 0.10,
	}
}

func plannerMixQuota(targetVideoCount int, flags model.PlannerFlags) model.MixQuota {
	if flags.HardReviewLowSupply {
		futureLikeMax := ceilFraction(targetVideoCount, 0.375)
		return model.MixQuota{
			CoreDominantMin:   ceilFraction(targetVideoCount, 0.375),
			FutureDominantMax: futureLikeMax,
			FutureLikeMax:     futureLikeMax,
			FallbackMax:       1,
			SameUnitMax:       2,
		}
	}

	return model.MixQuota{
		CoreDominantMin:   ceilFraction(targetVideoCount, 0.50),
		FutureDominantMax: floorFraction(targetVideoCount, 0.25),
		FutureLikeMax:     floorFraction(targetVideoCount, 0.25),
		FallbackMax:       1,
		SameUnitMax:       2,
	}
}

func sortDemandUnits(units []model.DemandUnit) {
	sort.Slice(units, func(i, j int) bool {
		if units[i].Weight == units[j].Weight {
			return units[i].UnitID < units[j].UnitID
		}
		return units[i].Weight > units[j].Weight
	})
}

func bucketBaseWeight(bucket string) float64 {
	switch bucket {
	case string(policy.BucketHardReview):
		return 1.00
	case string(policy.BucketNewNow):
		return 0.80
	case string(policy.BucketSoftReview):
		return 0.60
	default:
		return 0.40
	}
}

func supplyWeight(supplyGrade string) float64 {
	switch supplyGrade {
	case "strong":
		return 0.20
	case "ok":
		return 0.10
	case "weak":
		return 0.00
	default:
		return -0.10
	}
}

func ceilFraction(value int, fraction float64) int {
	return int(math.Ceil(float64(value) * fraction))
}

func floorFraction(value int, fraction float64) int {
	return int(math.Floor(float64(value) * fraction))
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}
