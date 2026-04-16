package planner

import "learning-video-recommendation-system/internal/recommendation/domain/model"

type DemandPlanner interface {
	Plan(context model.RecommendationContext) (model.DemandBundle, error)
}
