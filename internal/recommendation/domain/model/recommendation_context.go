package model

import "time"

type RecommendationContext struct {
	Request                RecommendationRequest
	PreferredDurationSec   [2]int
	Now                    time.Time
	ActiveUnitStates       []LearningStateSnapshot
	UnitInventory          []UnitVideoInventory
	UnitServingStates      []UserUnitServingState
	VideoServingStates     []UserVideoServingState
	VideoUserStates        []VideoUserState
	RecommendableVideoUnit []RecommendableVideoUnit
	RecallScope            RecallScopeSummary
}
