package dto

type MeRequest struct {
	UserID         string
	ClientTimezone string
}

type MeResponse struct {
	UserID           string           `json:"user_id"`
	Email            *string          `json:"email"`
	EmailConfirmed   bool             `json:"email_confirmed"`
	DisplayName      string           `json:"display_name"`
	AvatarURL        *string          `json:"avatar_url"`
	Locale           string           `json:"locale"`
	Timezone         *string          `json:"timezone"`
	OnboardingStatus string           `json:"onboarding_status"`
	BirthDate        *string          `json:"birth_date"`
	Gender           *string          `json:"gender"`
	EducationStage   *string          `json:"education_stage"`
	IPRegion         *string          `json:"ip_region"`
	Stats            MeStats          `json:"stats"`
	ActivityCalendar ActivityCalendar `json:"activity_calendar"`
}

type MeStats struct {
	TotalWatchSeconds int64 `json:"total_watch_seconds"`
	QuizAttemptCount  int64 `json:"quiz_attempt_count"`
	StartedUnitCount  int64 `json:"started_unit_count"`
}

type ActivityCalendar struct {
	Timezone          string        `json:"timezone"`
	Today             string        `json:"today"`
	CurrentStreakDays int64         `json:"current_streak_days"`
	Days              []ActivityDay `json:"days"`
}

type ActivityDay struct {
	LocalDate                string `json:"local_date"`
	WatchSeconds             int64  `json:"watch_seconds"`
	QuizAttemptCount         int64  `json:"quiz_attempt_count"`
	LearningInteractionCount int64  `json:"learning_interaction_count"`
}

type UpdateOnboardingStatusRequest struct {
	UserID string
	Status string
}

type UpdateOnboardingStatusResponse struct{}

type UpdateMeProfileRequest struct {
	UserID string

	SetDisplayName bool
	DisplayName    string

	SetBirthDate bool
	BirthDate    *string

	SetGender bool
	Gender    *string

	SetEducationStage bool
	EducationStage    *string

	SetTimezone bool
	Timezone    *string
}

type UpdateMeProfileResponse struct {
	UserID           string  `json:"user_id"`
	Email            *string `json:"email"`
	EmailConfirmed   bool    `json:"email_confirmed"`
	DisplayName      string  `json:"display_name"`
	AvatarURL        *string `json:"avatar_url"`
	Locale           string  `json:"locale"`
	Timezone         *string `json:"timezone"`
	OnboardingStatus string  `json:"onboarding_status"`
	BirthDate        *string `json:"birth_date"`
	Gender           *string `json:"gender"`
	EducationStage   *string `json:"education_stage"`
	IPRegion         *string `json:"ip_region"`
}
