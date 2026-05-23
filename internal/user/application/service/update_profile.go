package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"

	"learning-video-recommendation-system/internal/user/application/dto"
	"learning-video-recommendation-system/internal/user/application/repository"
	"learning-video-recommendation-system/internal/user/domain/model"
)

var (
	minBirthDate = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)

	allowedGenders = map[string]struct{}{
		"male":              {},
		"female":            {},
		"other":             {},
		"prefer_not_to_say": {},
	}

	allowedEducationStages = map[string]struct{}{
		"primary_school": {},
		"middle_school":  {},
		"high_school":    {},
		"undergraduate":  {},
		"graduate":       {},
		"phd":            {},
		"working":        {},
		"other":          {},
	}
)

type UpdateMeProfileUsecase struct {
	profiles repository.ProfileRepository
	now      func() time.Time
}

type UpdateMeProfileOption func(*UpdateMeProfileUsecase)

func WithUpdateMeProfileNow(now func() time.Time) UpdateMeProfileOption {
	return func(u *UpdateMeProfileUsecase) {
		if now != nil {
			u.now = now
		}
	}
}

func NewUpdateMeProfileUsecase(profiles repository.ProfileRepository, options ...UpdateMeProfileOption) *UpdateMeProfileUsecase {
	usecase := &UpdateMeProfileUsecase{
		profiles: profiles,
		now:      func() time.Time { return time.Now().UTC() },
	}
	for _, option := range options {
		option(usecase)
	}
	return usecase
}

func (u *UpdateMeProfileUsecase) Execute(ctx context.Context, request dto.UpdateMeProfileRequest) (dto.UpdateMeProfileResponse, error) {
	if err := u.validateBase(request); err != nil {
		return dto.UpdateMeProfileResponse{}, err
	}

	patch, err := u.validatePatch(request)
	if err != nil {
		return dto.UpdateMeProfileResponse{}, err
	}

	if _, err := u.getOrRepairProfile(ctx, request.UserID); err != nil {
		return dto.UpdateMeProfileResponse{}, err
	}

	profile, err := u.profiles.UpdateProfile(ctx, patch)
	if err != nil {
		return dto.UpdateMeProfileResponse{}, err
	}

	return mapProfileResponse(profile), nil
}

func (u *UpdateMeProfileUsecase) validateBase(request dto.UpdateMeProfileRequest) error {
	if request.UserID == "" {
		return ValidationError("user_id is required")
	}
	if u.profiles == nil {
		return errors.New("profile repository is required")
	}
	if !request.SetDisplayName &&
		!request.SetBirthDate &&
		!request.SetGender &&
		!request.SetEducationStage &&
		!request.SetTimezone {
		return ValidationError("profile patch must contain at least one field")
	}
	return nil
}

func (u *UpdateMeProfileUsecase) validatePatch(request dto.UpdateMeProfileRequest) (model.UserProfilePatch, error) {
	patch := model.UserProfilePatch{
		UserID:            request.UserID,
		SetDisplayName:    request.SetDisplayName,
		SetBirthDate:      request.SetBirthDate,
		SetGender:         request.SetGender,
		SetEducationStage: request.SetEducationStage,
		SetTimezone:       request.SetTimezone,
	}

	if request.SetDisplayName {
		displayName, err := normalizeDisplayName(request.DisplayName)
		if err != nil {
			return model.UserProfilePatch{}, err
		}
		patch.DisplayName = displayName
	}
	if request.SetBirthDate && request.BirthDate != nil {
		birthDate, err := u.parseBirthDate(*request.BirthDate)
		if err != nil {
			return model.UserProfilePatch{}, err
		}
		patch.BirthDate = &birthDate
	}
	if request.SetGender && request.Gender != nil {
		if _, ok := allowedGenders[*request.Gender]; !ok {
			return model.UserProfilePatch{}, ValidationError("gender is unsupported")
		}
		patch.Gender = request.Gender
	}
	if request.SetEducationStage && request.EducationStage != nil {
		if _, ok := allowedEducationStages[*request.EducationStage]; !ok {
			return model.UserProfilePatch{}, ValidationError("education_stage is unsupported")
		}
		patch.EducationStage = request.EducationStage
	}
	if request.SetTimezone && request.Timezone != nil {
		timezone, _, ok := validTimezone(*request.Timezone)
		if !ok {
			return model.UserProfilePatch{}, ValidationError("timezone must be a valid IANA timezone")
		}
		patch.Timezone = &timezone
	}

	return patch, nil
}

func (u *UpdateMeProfileUsecase) parseBirthDate(value string) (time.Time, error) {
	if len(value) != len("2006-01-02") {
		return time.Time{}, ValidationError("birth_date must be YYYY-MM-DD")
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, time.UTC)
	if err != nil {
		return time.Time{}, ValidationError("birth_date must be YYYY-MM-DD")
	}
	today := dateOnly(u.now().UTC())
	if parsed.Before(minBirthDate) || parsed.After(today) {
		return time.Time{}, ValidationError("birth_date must be between 1900-01-01 and today")
	}
	return parsed, nil
}

func (u *UpdateMeProfileUsecase) getOrRepairProfile(ctx context.Context, userID string) (model.UserProfile, error) {
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

func normalizeDisplayName(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	length := 0
	for _, char := range trimmed {
		length++
		if char == '_' {
			continue
		}
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			continue
		}
		return "", ValidationError("display_name may only contain letters, numbers, and underscore")
	}
	if length < 2 || length > 20 {
		return "", ValidationError("display_name must be 2-20 characters")
	}
	return trimmed, nil
}

func mapProfileResponse(profile model.UserProfile) dto.UpdateMeProfileResponse {
	return dto.UpdateMeProfileResponse{
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
	}
}
