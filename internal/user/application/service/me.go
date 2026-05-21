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
}

func NewGetMeUsecase(profiles repository.ProfileRepository, stats repository.ActivityStatsRepository) *GetMeUsecase {
	return &GetMeUsecase{profiles: profiles, stats: stats}
}

func (u *GetMeUsecase) Execute(ctx context.Context, request dto.MeRequest) (dto.MeResponse, error) {
	if request.UserID == "" {
		return dto.MeResponse{}, ValidationError("user_id is required")
	}
	if u.profiles == nil {
		return dto.MeResponse{}, errors.New("profile repository is required")
	}
	if u.stats == nil {
		return dto.MeResponse{}, errors.New("activity stats repository is required")
	}

	profile, found, err := u.profiles.GetProfile(ctx, request.UserID)
	if err != nil {
		return dto.MeResponse{}, err
	}
	if !found {
		profile, err = u.profiles.RepairProfile(ctx, request.UserID)
		if err != nil {
			return dto.MeResponse{}, err
		}
	}

	if timezone, _, ok := validTimezone(request.ClientTimezone); ok {
		if profile.Timezone == nil || *profile.Timezone != timezone {
			if err := u.profiles.UpdateTimezone(ctx, request.UserID, timezone); err != nil {
				return dto.MeResponse{}, err
			}
			profile.Timezone = &timezone
		}
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

	return dto.MeResponse{
		UserID:           profile.UserID,
		Email:            profile.Email,
		EmailConfirmed:   profile.EmailConfirmedAt != nil,
		DisplayName:      profile.DisplayName,
		AvatarURL:        profile.AvatarURL,
		Locale:           profile.Locale,
		Timezone:         profile.Timezone,
		OnboardingStatus: profile.OnboardingStatus,
		Stats: dto.MeStats{
			TotalWatchSeconds: activityStats.TotalWatchMS / 1000,
			QuizAttemptCount:  activityStats.QuizAttemptCount,
			StartedUnitCount:  activityStats.StartedUnitCount,
		},
	}, nil
}

type GetActivityCalendarUsecase struct {
	profiles repository.ProfileRepository
	stats    repository.ActivityStatsRepository
	now      func() time.Time
}

func NewGetActivityCalendarUsecase(profiles repository.ProfileRepository, stats repository.ActivityStatsRepository) *GetActivityCalendarUsecase {
	return &GetActivityCalendarUsecase{
		profiles: profiles,
		stats:    stats,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

func (u *GetActivityCalendarUsecase) Execute(ctx context.Context, request dto.ActivityCalendarRequest) (dto.ActivityCalendarResponse, error) {
	if request.UserID == "" {
		return dto.ActivityCalendarResponse{}, ValidationError("user_id is required")
	}
	if u.profiles == nil {
		return dto.ActivityCalendarResponse{}, errors.New("profile repository is required")
	}
	if u.stats == nil {
		return dto.ActivityCalendarResponse{}, errors.New("activity stats repository is required")
	}

	profile, found, err := u.profiles.GetProfile(ctx, request.UserID)
	if err != nil {
		return dto.ActivityCalendarResponse{}, err
	}
	if !found {
		profile, err = u.profiles.RepairProfile(ctx, request.UserID)
		if err != nil {
			return dto.ActivityCalendarResponse{}, err
		}
	}

	timezone, location := resolveTimezone(request.ClientTimezone, profile.Timezone)
	today := dateOnly(u.now().In(location))
	from := today.AddDate(0, 0, -6)

	rows, err := u.stats.ListDailyActivityStats(ctx, request.UserID, from, today)
	if err != nil {
		return dto.ActivityCalendarResponse{}, err
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
		isActive := watchSeconds > 0 || row.QuizAttemptCount > 0 || row.LearningInteractionCount > 0
		days = append(days, dto.ActivityDay{
			LocalDate:                key,
			WatchSeconds:             watchSeconds,
			QuizAttemptCount:         row.QuizAttemptCount,
			LearningInteractionCount: row.LearningInteractionCount,
			IsActive:                 isActive,
		})
	}

	return dto.ActivityCalendarResponse{
		Timezone: timezone,
		Today:    dateString(today),
		Days:     days,
	}, nil
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func dateString(value time.Time) string {
	return value.Format("2006-01-02")
}
