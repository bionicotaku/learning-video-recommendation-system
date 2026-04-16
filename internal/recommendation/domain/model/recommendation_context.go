package model

import "time"

type RecommendationContext struct {
	Request                RecommendationRequest
	Now                    time.Time
	ActiveUnitStates       []LearningStateSnapshot
	UnitInventory          []UnitVideoInventory
	UnitServingStates      []UserUnitServingState
	VideoServingStates     []UserVideoServingState
	VideoUserStates        []VideoUserState
	RecommendableVideoUnit []RecommendableVideoUnit
}
