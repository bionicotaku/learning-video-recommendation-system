//go:build integration

package unitcollections_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	apiservice "learning-video-recommendation-system/internal/api/application/service"
	apitx "learning-video-recommendation-system/internal/api/infrastructure/persistence/tx"
	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	"learning-video-recommendation-system/internal/platform/postgres/pgtest"
	usermodel "learning-video-recommendation-system/internal/user/domain/model"
)

func TestActivateLearningCollectionCommitsTargetAndOnboardingTogether(t *testing.T) {
	db := openActivationTestDatabase(t)
	seedActivationUser(t, db, userID, "student@example.com")
	seedActivationCollection(t, db, "22222222-2222-4222-8222-222222222222", "toefl-core", []int64{101, 102})

	service := apiservice.NewActivateLearningCollectionService(apitx.NewActivateCollectionManager(db.Pool))
	response, err := service.Execute(context.Background(), learningdto.ActivateUnitCollectionTargetRequest{
		UserID:         userID,
		CollectionSlug: "toefl-core",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.CollectionSlug != "toefl-core" || response.TargetCount != 2 {
		t.Fatalf("response = %+v", response)
	}

	var activeSlug string
	var onboardingStatus string
	var targetCount int
	if err := db.Pool.QueryRow(context.Background(), `select active_collection_slug from learning.user_learning_profiles where user_id = $1`, userID).Scan(&activeSlug); err != nil {
		t.Fatalf("read active collection: %v", err)
	}
	if err := db.Pool.QueryRow(context.Background(), `select onboarding_status from app_user.user_profiles where user_id = $1`, userID).Scan(&onboardingStatus); err != nil {
		t.Fatalf("read onboarding status: %v", err)
	}
	if err := db.Pool.QueryRow(context.Background(), `select count(*) from learning.user_unit_states where user_id = $1 and is_target = true`, userID).Scan(&targetCount); err != nil {
		t.Fatalf("read target count: %v", err)
	}
	if activeSlug != "toefl-core" || onboardingStatus != usermodel.OnboardingStatusCollectionSelected || targetCount != 2 {
		t.Fatalf("active_slug=%q onboarding=%q target_count=%d", activeSlug, onboardingStatus, targetCount)
	}
}

func TestActivateLearningCollectionRollsBackTargetWhenOnboardingCannotUpdate(t *testing.T) {
	db := openActivationTestDatabase(t)
	seedActivationUser(t, db, userID, "student@example.com")
	seedActivationCollection(t, db, "33333333-3333-4333-8333-333333333333", "ielts-core", []int64{201})
	failProfileUpdate(t, db)

	service := apiservice.NewActivateLearningCollectionService(apitx.NewActivateCollectionManager(db.Pool))
	_, err := service.Execute(context.Background(), learningdto.ActivateUnitCollectionTargetRequest{
		UserID:         userID,
		CollectionSlug: "ielts-core",
	})
	if err == nil {
		t.Fatalf("Execute() error = nil, want onboarding update failure")
	}

	var profileCount int
	var targetCount int
	if err := db.Pool.QueryRow(context.Background(), `select count(*) from learning.user_learning_profiles where user_id = $1`, userID).Scan(&profileCount); err != nil {
		t.Fatalf("read learning profile count: %v", err)
	}
	if err := db.Pool.QueryRow(context.Background(), `select count(*) from learning.user_unit_states where user_id = $1`, userID).Scan(&targetCount); err != nil {
		t.Fatalf("read target count: %v", err)
	}
	if profileCount != 0 || targetCount != 0 {
		t.Fatalf("learning writes should roll back, profile_count=%d target_count=%d", profileCount, targetCount)
	}
}

func openActivationTestDatabase(t *testing.T) *pgtest.Database {
	t.Helper()
	suite, err := pgtest.OpenSuite(pgtest.Options{
		TempDirPrefix:        "api-unitcollections-*",
		TemplateDatabaseName: "api_unitcollections_template",
		DatabaseNamePrefix:   "api_unitcollections_test",
		SchemaPlan: pgtest.NewSchemaPlan(
			pgtest.SQLFile(pgtest.RepoPath(
				"internal",
				"learningengine",
				"reducer",
				"infrastructure",
				"persistence",
				"schema",
				"000000_external_refs.sql",
			)),
			pgtest.MigrationDir(pgtest.RepoPath("internal", "user", "infrastructure", "migration")),
			pgtest.MigrationDir(pgtest.RepoPath("internal", "learningengine", "reducer", "infrastructure", "migration")),
		),
	})
	if err != nil {
		t.Fatalf("open pgtest suite: %v", err)
	}
	t.Cleanup(func() {
		if err := suite.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "close api unitcollections suite: %v\n", err)
		}
	})
	return suite.CreateTestDatabase(t)
}

func seedActivationUser(t *testing.T, db *pgtest.Database, userID string, email string) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `insert into auth.users (id, email, email_confirmed_at) values ($1, $2, now())`, userID, email); err != nil {
		t.Fatalf("seed auth user: %v", err)
	}
}

func seedActivationCollection(t *testing.T, db *pgtest.Database, collectionID string, slug string, unitIDs []int64) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		insert into semantic.unit_collections (collection_id, slug, name, status, coarse_unit_count)
		values ($1::uuid, $2, $3, 'active', $4)
	`, collectionID, slug, slug, len(unitIDs)); err != nil {
		t.Fatalf("seed collection: %v", err)
	}
	for index, unitID := range unitIDs {
		if _, err := db.Pool.Exec(context.Background(), `
			insert into semantic.coarse_unit (
				id,
				kind,
				label,
				lang,
				status,
				version,
				fine_unit_ids,
				original_defs
		) values (
			$1::bigint,
			'word',
			'unit-' || $1::bigint::text,
			'en',
				'active',
				1,
				'{}'::bigint[],
				'{}'::text[]
			)
		`, unitID); err != nil {
			t.Fatalf("seed coarse unit %d: %v", unitID, err)
		}
		if _, err := db.Pool.Exec(context.Background(), `
			insert into semantic.unit_collection_members (collection_id, coarse_unit_id, sort_order, target_priority)
			values ($1::uuid, $2, $3, 0)
		`, collectionID, unitID, index+1); err != nil {
			t.Fatalf("seed collection member %d: %v", unitID, err)
		}
	}
}

func failProfileUpdate(t *testing.T, db *pgtest.Database) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		create or replace function app_user.fail_profile_update()
		returns trigger
		language plpgsql
		as $$
		begin
			raise exception 'profile update failed';
		end;
		$$;
		create trigger fail_profile_update
		before update on app_user.user_profiles
		for each row execute function app_user.fail_profile_update();
	`); err != nil {
		t.Fatalf("install profile update failure trigger: %v", err)
	}
}
