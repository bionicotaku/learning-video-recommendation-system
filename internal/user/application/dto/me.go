package dto

type MeRequest struct {
	UserID         string
	ClientTimezone string
}

type MeResponse struct {
	UserID           string  `json:"user_id"`
	Email            *string `json:"email"`
	EmailConfirmed   bool    `json:"email_confirmed"`
	DisplayName      *string `json:"display_name"`
	AvatarURL        *string `json:"avatar_url"`
	Locale           string  `json:"locale"`
	Timezone         *string `json:"timezone"`
	OnboardingStatus string  `json:"onboarding_status"`
	Stats            MeStats `json:"stats"`
}

type MeStats struct {
	TotalWatchSeconds int64 `json:"total_watch_seconds"`
	QuizAttemptCount  int64 `json:"quiz_attempt_count"`
	StartedUnitCount  int64 `json:"started_unit_count"`
}

type ActivityCalendarRequest struct {
	UserID         string
	ClientTimezone string
}

type ActivityCalendarResponse struct {
	Timezone string        `json:"timezone"`
	Today    string        `json:"today"`
	Days     []ActivityDay `json:"days"`
}

type ActivityDay struct {
	LocalDate                string `json:"local_date"`
	WatchSeconds             int64  `json:"watch_seconds"`
	QuizAttemptCount         int64  `json:"quiz_attempt_count"`
	LearningInteractionCount int64  `json:"learning_interaction_count"`
	IsActive                 bool   `json:"is_active"`
}

type UpdateOnboardingStatusRequest struct {
	UserID string
	Status string
}

type UpdateOnboardingStatusResponse struct{}
