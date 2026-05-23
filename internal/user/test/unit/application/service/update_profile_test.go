package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	userdto "learning-video-recommendation-system/internal/user/application/dto"
	userrepo "learning-video-recommendation-system/internal/user/application/repository"
	userservice "learning-video-recommendation-system/internal/user/application/service"
	"learning-video-recommendation-system/internal/user/domain/model"
)

func TestUpdateMeProfileRejectsInvalidRequests(t *testing.T) {
	validName := "Alice_01"
	tests := []struct {
		name    string
		request userdto.UpdateMeProfileRequest
	}{
		{
			name:    "missing_user_id",
			request: userdto.UpdateMeProfileRequest{SetDisplayName: true, DisplayName: validName},
		},
		{
			name:    "empty_patch",
			request: userdto.UpdateMeProfileRequest{UserID: validUserID},
		},
		{
			name:    "display_name_too_short",
			request: userdto.UpdateMeProfileRequest{UserID: validUserID, SetDisplayName: true, DisplayName: "a"},
		},
		{
			name:    "display_name_too_long",
			request: userdto.UpdateMeProfileRequest{UserID: validUserID, SetDisplayName: true, DisplayName: "abcdefghijklmnopqrstu"},
		},
		{
			name:    "display_name_space",
			request: userdto.UpdateMeProfileRequest{UserID: validUserID, SetDisplayName: true, DisplayName: "Alice Bob"},
		},
		{
			name:    "display_name_emoji",
			request: userdto.UpdateMeProfileRequest{UserID: validUserID, SetDisplayName: true, DisplayName: "Alice😊"},
		},
		{
			name:    "display_name_hyphen",
			request: userdto.UpdateMeProfileRequest{UserID: validUserID, SetDisplayName: true, DisplayName: "Alice-Bob"},
		},
		{
			name:    "birth_date_bad_format",
			request: userdto.UpdateMeProfileRequest{UserID: validUserID, SetBirthDate: true, BirthDate: stringPtr("2001-9-1")},
		},
		{
			name:    "birth_date_too_old",
			request: userdto.UpdateMeProfileRequest{UserID: validUserID, SetBirthDate: true, BirthDate: stringPtr("1899-12-31")},
		},
		{
			name:    "birth_date_future",
			request: userdto.UpdateMeProfileRequest{UserID: validUserID, SetBirthDate: true, BirthDate: stringPtr("2026-05-23")},
		},
		{
			name:    "gender_unknown",
			request: userdto.UpdateMeProfileRequest{UserID: validUserID, SetGender: true, Gender: stringPtr("unknown")},
		},
		{
			name:    "education_stage_unknown",
			request: userdto.UpdateMeProfileRequest{UserID: validUserID, SetEducationStage: true, EducationStage: stringPtr("college")},
		},
		{
			name:    "timezone_unknown",
			request: userdto.UpdateMeProfileRequest{UserID: validUserID, SetTimezone: true, Timezone: stringPtr("Mars/Olympus")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeProfileRepository{}
			usecase := userservice.NewUpdateMeProfileUsecase(
				repository,
				userservice.WithUpdateMeProfileNow(func() time.Time {
					return time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
				}),
			)

			_, err := usecase.Execute(context.Background(), tt.request)

			if !userservice.IsValidationError(err) {
				t.Fatalf("err = %v, want validation error", err)
			}
			if repository.updated {
				t.Fatalf("repository should not be updated")
			}
		})
	}
}

func TestUpdateMeProfileAcceptsUnicodeNamesAndMapsPatch(t *testing.T) {
	tests := []string{"张三", "ユーザー1", "사용자1", "Élodie_2"}

	for _, displayName := range tests {
		t.Run(displayName, func(t *testing.T) {
			repository := &fakeProfileRepository{
				profile: model.UserProfile{
					UserID:           validUserID,
					DisplayName:      "old",
					Locale:           model.DefaultLocale,
					OnboardingStatus: model.OnboardingStatusNew,
				},
				found: true,
			}
			usecase := userservice.NewUpdateMeProfileUsecase(
				repository,
				userservice.WithUpdateMeProfileNow(func() time.Time {
					return time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
				}),
			)

			response, err := usecase.Execute(context.Background(), userdto.UpdateMeProfileRequest{
				UserID:            validUserID,
				SetDisplayName:    true,
				DisplayName:       "  " + displayName + "  ",
				SetBirthDate:      true,
				BirthDate:         stringPtr("2001-09-01"),
				SetGender:         true,
				Gender:            stringPtr("prefer_not_to_say"),
				SetEducationStage: true,
				EducationStage:    stringPtr("undergraduate"),
				SetTimezone:       true,
				Timezone:          stringPtr("Asia/Shanghai"),
			})
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if !repository.updated {
				t.Fatalf("repository was not updated")
			}
			if repository.patch.DisplayName != displayName {
				t.Fatalf("display_name patch = %q, want %q", repository.patch.DisplayName, displayName)
			}
			if !repository.patch.SetBirthDate || repository.patch.BirthDate == nil || repository.patch.BirthDate.Format("2006-01-02") != "2001-09-01" {
				t.Fatalf("birth_date patch not mapped: %+v", repository.patch)
			}
			if response.DisplayName != displayName || response.BirthDate == nil || *response.BirthDate != "2001-09-01" {
				t.Fatalf("unexpected response: %+v", response)
			}
		})
	}
}

func TestUpdateMeProfileAcceptsPrimarySchoolEducationStage(t *testing.T) {
	repository := &fakeProfileRepository{
		profile: model.UserProfile{
			UserID:           validUserID,
			DisplayName:      "old",
			Locale:           model.DefaultLocale,
			OnboardingStatus: model.OnboardingStatusNew,
		},
		found: true,
	}
	usecase := userservice.NewUpdateMeProfileUsecase(repository)

	response, err := usecase.Execute(context.Background(), userdto.UpdateMeProfileRequest{
		UserID:            validUserID,
		SetEducationStage: true,
		EducationStage:    stringPtr("primary_school"),
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !repository.patch.SetEducationStage || repository.patch.EducationStage == nil || *repository.patch.EducationStage != "primary_school" {
		t.Fatalf("education_stage patch not mapped: %+v", repository.patch)
	}
	if response.EducationStage == nil || *response.EducationStage != "primary_school" {
		t.Fatalf("education_stage response = %+v, want primary_school", response.EducationStage)
	}
}

func TestUpdateMeProfileClearsNullableFields(t *testing.T) {
	repository := &fakeProfileRepository{
		profile: model.UserProfile{
			UserID:           validUserID,
			DisplayName:      "alice",
			Locale:           model.DefaultLocale,
			OnboardingStatus: model.OnboardingStatusNew,
		},
		found: true,
	}
	usecase := userservice.NewUpdateMeProfileUsecase(repository)

	_, err := usecase.Execute(context.Background(), userdto.UpdateMeProfileRequest{
		UserID:            validUserID,
		SetBirthDate:      true,
		BirthDate:         nil,
		SetGender:         true,
		Gender:            nil,
		SetEducationStage: true,
		EducationStage:    nil,
		SetTimezone:       true,
		Timezone:          nil,
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !repository.patch.SetBirthDate || repository.patch.BirthDate != nil ||
		!repository.patch.SetGender || repository.patch.Gender != nil ||
		!repository.patch.SetEducationStage || repository.patch.EducationStage != nil ||
		!repository.patch.SetTimezone || repository.patch.Timezone != nil {
		t.Fatalf("clear patch not mapped: %+v", repository.patch)
	}
}

func TestUpdateMeProfileRepairsMissingProfileBeforeUpdate(t *testing.T) {
	repository := &fakeProfileRepository{
		profile: model.UserProfile{
			UserID:           validUserID,
			DisplayName:      "repaired",
			Locale:           model.DefaultLocale,
			OnboardingStatus: model.OnboardingStatusNew,
		},
		found: false,
	}
	usecase := userservice.NewUpdateMeProfileUsecase(repository)

	_, err := usecase.Execute(context.Background(), userdto.UpdateMeProfileRequest{
		UserID:         validUserID,
		SetDisplayName: true,
		DisplayName:    "alice",
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !repository.repaired || !repository.updated {
		t.Fatalf("repair/update not called: repaired=%v updated=%v", repository.repaired, repository.updated)
	}
}

func TestUpdateMeProfileReturnsAuthUserNotFoundWhenRepairFails(t *testing.T) {
	repository := &fakeProfileRepository{
		found:     false,
		repairErr: userrepo.ErrAuthUserNotFound,
	}
	usecase := userservice.NewUpdateMeProfileUsecase(repository)

	_, err := usecase.Execute(context.Background(), userdto.UpdateMeProfileRequest{
		UserID:         validUserID,
		SetDisplayName: true,
		DisplayName:    "alice",
	})

	if !errors.Is(err, userrepo.ErrAuthUserNotFound) {
		t.Fatalf("err = %v, want ErrAuthUserNotFound", err)
	}
}

const validUserID = "11111111-1111-4111-8111-111111111111"

type fakeProfileRepository struct {
	profile   model.UserProfile
	found     bool
	repaired  bool
	updated   bool
	patch     model.UserProfilePatch
	repairErr error
	updateErr error
}

func (f *fakeProfileRepository) GetProfile(_ context.Context, _ string) (model.UserProfile, bool, error) {
	return f.profile, f.found, nil
}

func (f *fakeProfileRepository) RepairProfile(_ context.Context, _ string) (model.UserProfile, error) {
	f.repaired = true
	if f.repairErr != nil {
		return model.UserProfile{}, f.repairErr
	}
	f.found = true
	return f.profile, nil
}

func (f *fakeProfileRepository) UpdateProfile(_ context.Context, patch model.UserProfilePatch) (model.UserProfile, error) {
	f.updated = true
	f.patch = patch
	if f.updateErr != nil {
		return model.UserProfile{}, f.updateErr
	}
	profile := f.profile
	if patch.SetDisplayName {
		profile.DisplayName = patch.DisplayName
	}
	if patch.SetBirthDate {
		profile.BirthDate = patch.BirthDate
	}
	if patch.SetGender {
		profile.Gender = patch.Gender
	}
	if patch.SetEducationStage {
		profile.EducationStage = patch.EducationStage
	}
	if patch.SetTimezone {
		profile.Timezone = patch.Timezone
	}
	f.profile = profile
	return profile, nil
}

func (f *fakeProfileRepository) UpdateTimezone(context.Context, string, string) error {
	return nil
}

func (f *fakeProfileRepository) UpdateOnboardingStatus(context.Context, string, string) error {
	return nil
}

func stringPtr(value string) *string {
	return &value
}
