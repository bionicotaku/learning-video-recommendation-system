package policy

const QuizSpeedThresholdMS int32 = 5000

func QuizProgressQuality(isFirstTryCorrect bool, totalElapsedMS int32) int16 {
	if isFirstTryCorrect {
		if totalElapsedMS <= QuizSpeedThresholdMS {
			return 5
		}
		return 4
	}
	if totalElapsedMS <= QuizSpeedThresholdMS {
		return 2
	}
	return 1
}
