//go:build integration

package service_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	userdto "learning-video-recommendation-system/internal/user/application/dto"
	userservice "learning-video-recommendation-system/internal/user/application/service"
	userrepo "learning-video-recommendation-system/internal/user/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/user/test/fixture"
)

var suite *fixture.Suite

func TestMain(m *testing.M) {
	var err error
	suite, err = fixture.OpenSuite()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "open user integration suite: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	if err := suite.Close(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "close user integration suite: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

func TestAuthTriggersCreateProfileAndSyncEmailOnly(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	userID := "11111111-1111-4111-8111-111111111111"
	db.SeedAuthUser(t, userID, "alice@example.com")

	var displayName string
	var email string
	var locale string
	var birthDateIsNull bool
	var genderIsNull bool
	var educationStageIsNull bool
	var ipRegionIsNull bool
	if err := db.Pool.QueryRow(context.Background(), `
		select email, display_name, locale, birth_date is null, gender is null, education_stage is null, ip_region is null
		from app_user.user_profiles
		where user_id = $1`, userID).Scan(&email, &displayName, &locale, &birthDateIsNull, &genderIsNull, &educationStageIsNull, &ipRegionIsNull); err != nil {
		t.Fatalf("read profile: %v", err)
	}
	if email != "alice@example.com" || displayName != "alice" || locale != "zh-CN" {
		t.Fatalf("unexpected profile defaults: email=%q display=%q locale=%q", email, displayName, locale)
	}
	if !birthDateIsNull || !genderIsNull || !educationStageIsNull || !ipRegionIsNull {
		t.Fatalf("new profile fields should default to null")
	}

	fallbackUserID := "11111111-1111-4111-8111-111111111112"
	if _, err := db.Pool.Exec(context.Background(), `insert into auth.users (id, email, email_confirmed_at) values ($1, null, now())`, fallbackUserID); err != nil {
		t.Fatalf("seed auth user without email: %v", err)
	}
	if err := db.Pool.QueryRow(context.Background(), `
		select display_name
		from app_user.user_profiles
		where user_id = $1`, fallbackUserID).Scan(&displayName); err != nil {
		t.Fatalf("read fallback profile: %v", err)
	}
	if displayName != "user" {
		t.Fatalf("display_name fallback = %q, want user", displayName)
	}

	if _, err := db.Pool.Exec(context.Background(), `update app_user.user_profiles set display_name = 'custom' where user_id = $1`, userID); err != nil {
		t.Fatalf("update display: %v", err)
	}
	if _, err := db.Pool.Exec(context.Background(), `update auth.users set email = 'new@example.com' where id = $1`, userID); err != nil {
		t.Fatalf("update auth email: %v", err)
	}
	if err := db.Pool.QueryRow(context.Background(), `
		select email, display_name
		from app_user.user_profiles
		where user_id = $1`, userID).Scan(&email, &displayName); err != nil {
		t.Fatalf("read updated profile: %v", err)
	}
	if email != "new@example.com" || displayName != "custom" {
		t.Fatalf("email sync should not override display: email=%q display=%q", email, displayName)
	}
}

func TestGetMeRepairsProfileAndUpdatesTimezone(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	userID := "22222222-2222-4222-8222-222222222222"
	db.SeedAuthUser(t, userID, "bob@example.com")
	if _, err := db.Pool.Exec(context.Background(), `delete from app_user.user_profiles where user_id = $1`, userID); err != nil {
		t.Fatalf("delete profile: %v", err)
	}

	repository := userrepo.NewRepository(db.Pool)
	usecase := userservice.NewGetMeUsecase(repository, repository)
	response, err := usecase.Execute(context.Background(), userdtoMeRequest(userID, "Asia/Shanghai"))
	if err != nil {
		t.Fatalf("GetMe Execute: %v", err)
	}
	if response.DisplayName != "bob" || response.Timezone == nil || *response.Timezone != "Asia/Shanghai" {
		t.Fatalf("unexpected response: %+v", response)
	}
	if response.BirthDate != nil || response.Gender != nil || response.EducationStage != nil || response.IPRegion != nil {
		t.Fatalf("new profile fields should default to nil: %+v", response)
	}
	if response.Stats.TotalWatchSeconds != 0 || response.Stats.QuizAttemptCount != 0 || response.Stats.StartedUnitCount != 0 {
		t.Fatalf("stats should default to zero: %+v", response.Stats)
	}
}

func TestGetMeRepairsProfileWithDisplayNameFallback(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	userID := "22222222-2222-4222-8222-222222222223"
	if _, err := db.Pool.Exec(context.Background(), `insert into auth.users (id, email, email_confirmed_at) values ($1, null, now())`, userID); err != nil {
		t.Fatalf("seed auth user without email: %v", err)
	}
	if _, err := db.Pool.Exec(context.Background(), `delete from app_user.user_profiles where user_id = $1`, userID); err != nil {
		t.Fatalf("delete profile: %v", err)
	}

	repository := userrepo.NewRepository(db.Pool)
	usecase := userservice.NewGetMeUsecase(repository, repository)
	response, err := usecase.Execute(context.Background(), userdtoMeRequest(userID, ""))
	if err != nil {
		t.Fatalf("GetMe Execute: %v", err)
	}
	if response.DisplayName != "user" {
		t.Fatalf("display_name fallback = %q, want user", response.DisplayName)
	}
}

func TestActivityStatsRecorderAndCalendar(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	userID := "33333333-3333-4333-8333-333333333333"
	db.SeedAuthUser(t, userID, "carol@example.com")
	repository := userrepo.NewRepository(db.Pool)
	if err := repository.UpdateTimezone(context.Background(), userID, "America/Los_Angeles"); err != nil {
		t.Fatalf("update timezone: %v", err)
	}

	at := time.Now().UTC()
	if err := repository.AddWatchDuration(context.Background(), userID, at, 1500); err != nil {
		t.Fatalf("add watch: %v", err)
	}
	if err := repository.IncrementQuizAttempt(context.Background(), userID, at); err != nil {
		t.Fatalf("increment quiz: %v", err)
	}
	if err := repository.IncrementLearningInteraction(context.Background(), userID, at); err != nil {
		t.Fatalf("increment interaction: %v", err)
	}
	if err := repository.IncrementStartedUnit(context.Background(), userID); err != nil {
		t.Fatalf("increment started: %v", err)
	}

	stats, found, err := repository.GetActivityStats(context.Background(), userID)
	if err != nil || !found {
		t.Fatalf("GetActivityStats found=%v err=%v", found, err)
	}
	if stats.TotalWatchMS != 1500 || stats.QuizAttemptCount != 1 || stats.StartedUnitCount != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	getMe := userservice.NewGetMeUsecase(repository, repository)
	response, err := getMe.Execute(context.Background(), userdtoMeRequest(userID, "America/Los_Angeles"))
	if err != nil {
		t.Fatalf("GetMe Execute: %v", err)
	}
	if len(response.ActivityCalendar.Days) != 7 {
		t.Fatalf("days len = %d, want 7", len(response.ActivityCalendar.Days))
	}
	location, _ := time.LoadLocation("America/Los_Angeles")
	activeLocalDate := at.In(location).Format("2006-01-02")
	var activeDayFound bool
	for _, day := range response.ActivityCalendar.Days {
		if day.LocalDate == activeLocalDate {
			activeDayFound = day.WatchSeconds == 1 && day.QuizAttemptCount == 1 && day.LearningInteractionCount == 1
		}
	}
	if !activeDayFound {
		t.Fatalf("expected active local day in response: %+v", response)
	}
}

func TestActivityCalendarReturnsCurrentStreakFromYesterdayWhenTodayInactive(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	userID := "44444444-4444-4444-8444-444444444444"
	db.SeedAuthUser(t, userID, "dora@example.com")
	repository := userrepo.NewRepository(db.Pool)
	if err := repository.UpdateTimezone(context.Background(), userID, "UTC"); err != nil {
		t.Fatalf("update timezone: %v", err)
	}

	today := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	for _, offset := range []int{-1, -2, -3} {
		if err := repository.IncrementLearningInteraction(context.Background(), userID, today.AddDate(0, 0, offset)); err != nil {
			t.Fatalf("increment interaction day %d: %v", offset, err)
		}
	}

	getMe := userservice.NewGetMeUsecase(
		repository,
		repository,
		userservice.WithMeNow(func() time.Time { return today }),
	)
	response, err := getMe.Execute(context.Background(), userdtoMeRequest(userID, "UTC"))
	if err != nil {
		t.Fatalf("GetMe Execute: %v", err)
	}
	if response.ActivityCalendar.CurrentStreakDays != 3 {
		t.Fatalf("current streak = %d, want 3: %+v", response.ActivityCalendar.CurrentStreakDays, response)
	}
}

func TestActivityCalendarReturnsZeroStreakWhenTodayAndYesterdayInactive(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	userID := "55555555-5555-4555-8555-555555555555"
	db.SeedAuthUser(t, userID, "erin@example.com")
	repository := userrepo.NewRepository(db.Pool)

	today := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	if err := repository.IncrementLearningInteraction(context.Background(), userID, today.AddDate(0, 0, -2)); err != nil {
		t.Fatalf("increment interaction: %v", err)
	}

	getMe := userservice.NewGetMeUsecase(
		repository,
		repository,
		userservice.WithMeNow(func() time.Time { return today }),
	)
	response, err := getMe.Execute(context.Background(), userdtoMeRequest(userID, "UTC"))
	if err != nil {
		t.Fatalf("GetMe Execute: %v", err)
	}
	if response.ActivityCalendar.CurrentStreakDays != 0 {
		t.Fatalf("current streak = %d, want 0: %+v", response.ActivityCalendar.CurrentStreakDays, response)
	}
}

func userdtoMeRequest(userID string, timezone string) userdto.MeRequest {
	return userdto.MeRequest{UserID: userID, ClientTimezone: timezone}
}
