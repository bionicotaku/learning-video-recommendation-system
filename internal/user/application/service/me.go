package service

import (
	"context"
	"errors"
	"time"

	"learning-video-recommendation-system/internal/user/application/dto"
	"learning-video-recommendation-system/internal/user/application/repository"
	"learning-video-recommendation-system/internal/user/domain/model"
)

type GetMeUsecase struct {
	profiles repository.ProfileRepository
	stats    repository.ActivityStatsRepository
	now      func() time.Time
}

type MeOption func(*GetMeUsecase)

func WithMeNow(now func() time.Time) MeOption {
	return func(u *GetMeUsecase) {
		if now != nil {
			u.now = now
		}
	}
}

func NewGetMeUsecase(profiles repository.ProfileRepository, stats repository.ActivityStatsRepository, options ...MeOption) *GetMeUsecase {
	usecase := &GetMeUsecase{
		profiles: profiles,
		stats:    stats,
		now:      func() time.Time { return time.Now().UTC() },
	}
	for _, option := range options {
		option(usecase)
	}
	return usecase
}

func (u *GetMeUsecase) Execute(ctx context.Context, request dto.MeRequest) (dto.MeResponse, error) {
	if err := u.validate(request.UserID); err != nil {
		return dto.MeResponse{}, err
	}

	profile, err := u.getOrRepairProfile(ctx, request.UserID)
	if err != nil {
		return dto.MeResponse{}, err
	}
	if err := u.updateTimezoneIfNeeded(ctx, request.UserID, request.ClientTimezone, &profile); err != nil {
		return dto.MeResponse{}, err
	}

	if err := u.stats.EnsureActivityStats(ctx, request.UserID); err != nil {
		return dto.MeResponse{}, err
	}
	activityStats, found, err := u.stats.GetActivityStats(ctx, request.UserID)
	if err != nil {
		return dto.MeResponse{}, err
	}
	if !found {
		activityStats = model.ActivityStats{UserID: request.UserID}
	}

	calendar, err := u.buildActivityCalendar(ctx, request.UserID, profile, request.ClientTimezone)
	if err != nil {
		return dto.MeResponse{}, err
	}

	return dto.MeResponse{
		UserID:           profile.UserID,
		Email:            profile.Email,
		EmailConfirmed:   profile.EmailConfirmedAt != nil,
		DisplayName:      profile.DisplayName,
		AvatarURL:        profile.AvatarURL,
		Locale:           profile.Locale,
		Timezone:         profile.Timezone,
		OnboardingStatus: profile.OnboardingStatus,
		BirthDate:        dateStringPointer(profile.BirthDate),
		Gender:           profile.Gender,
		EducationStage:   profile.EducationStage,
		IPRegion:         profile.IPRegion,
		Stats: dto.MeStats{
			TotalWatchSeconds: activityStats.TotalWatchMS / 1000,
			QuizAttemptCount:  activityStats.QuizAttemptCount,
			StartedUnitCount:  activityStats.StartedUnitCount,
		},
		ActivityCalendar: calendar,
	}, nil
}

func (u *GetMeUsecase) validate(userID string) error {
	if userID == "" {
		return ValidationError("user_id is required")
	}
	if u.profiles == nil {
		return errors.New("profile repository is required")
	}
	if u.stats == nil {
		return errors.New("activity stats repository is required")
	}
	return nil
}

func (u *GetMeUsecase) getOrRepairProfile(ctx context.Context, userID string) (model.UserProfile, error) {
	profile, found, err := u.profiles.GetProfile(ctx, userID)
	if err != nil {
		return model.UserProfile{}, err
	}
	if !found {
		profile, err = u.profiles.RepairProfile(ctx, userID)
		if err != nil {
			return model.UserProfile{}, err
		}
	}
	return profile, nil
}

func (u *GetMeUsecase) updateTimezoneIfNeeded(ctx context.Context, userID string, clientTimezone string, profile *model.UserProfile) error {
	if timezone, _, ok := validTimezone(clientTimezone); ok {
		if profile.Timezone == nil || *profile.Timezone != timezone {
			if err := u.profiles.UpdateTimezone(ctx, userID, timezone); err != nil {
				return err
			}
			profile.Timezone = &timezone
		}
	}
	return nil
}

func (u *GetMeUsecase) buildActivityCalendar(ctx context.Context, userID string, profile model.UserProfile, clientTimezone string) (dto.ActivityCalendar, error) {
	timezone, location := resolveTimezone(clientTimezone, profile.Timezone)
	today := dateOnly(u.now().In(location))
	from := today.AddDate(0, 0, -6)

	rows, err := u.stats.ListDailyActivityStats(ctx, userID, from, today)
	if err != nil {
		return dto.ActivityCalendar{}, err
	}
	currentStreakDays, err := u.stats.GetCurrentActivityStreakDays(ctx, userID, today)
	if err != nil {
		return dto.ActivityCalendar{}, err
	}
	byDate := make(map[string]model.DailyActivityStats, len(rows))
	for _, row := range rows {
		byDate[dateString(row.LocalDate)] = row
	}

	days := make([]dto.ActivityDay, 0, 7)
	for day := from; !day.After(today); day = day.AddDate(0, 0, 1) {
		key := dateString(day)
		row := byDate[key]
		watchSeconds := row.WatchMS / 1000
		days = append(days, dto.ActivityDay{
			LocalDate:                key,
			WatchSeconds:             watchSeconds,
			QuizAttemptCount:         row.QuizAttemptCount,
			LearningInteractionCount: row.LearningInteractionCount,
		})
	}

	return dto.ActivityCalendar{
		Timezone:          timezone,
		Today:             dateString(today),
		CurrentStreakDays: currentStreakDays,
		Days:              days,
	}, nil
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func dateString(value time.Time) string {
	return value.Format("2006-01-02")
}

func dateStringPointer(value *time.Time) *string {
	if value == nil {
		return nil
	}
	result := dateString(*value)
	return &result
}
