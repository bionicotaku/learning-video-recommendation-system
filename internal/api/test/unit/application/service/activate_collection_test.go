package service_test

import (
	"context"
	"errors"
	"testing"

	apirepo "learning-video-recommendation-system/internal/api/application/repository"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	learningrepo "learning-video-recommendation-system/internal/learningengine/reducer/application/repository"
	learningservice "learning-video-recommendation-system/internal/learningengine/reducer/application/service"
	learningmodel "learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
	userrepo "learning-video-recommendation-system/internal/user/application/repository"
	usermodel "learning-video-recommendation-system/internal/user/domain/model"
)

const userID = "11111111-1111-4111-8111-111111111111"

func TestActivateLearningCollectionUpdatesTargetAndOnboardingInOneTx(t *testing.T) {
	targets := &fakeTargetCommandRepository{
		activated: learningmodel.ActivatedUnitCollectionTarget{
			CollectionID:   "11111111-1111-4111-8111-111111111111",
			CollectionSlug: "toefl-core",
			TargetCount:    1000,
		},
	}
	profiles := &fakeProfileRepository{
		profile: &usermodel.UserProfile{UserID: userID, OnboardingStatus: usermodel.OnboardingStatusNew},
	}
	txManager := &fakeActivateCollectionTxManager{repos: fakeActivateCollectionRepositories{
		targets:  targets,
		profiles: profiles,
	}}
	service := apiservice.NewActivateLearningCollectionService(txManager)

	response, err := service.Execute(context.Background(), learningdto.ActivateUnitCollectionTargetRequest{
		UserID:         userID,
		CollectionSlug: "toefl-core",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !txManager.committed || txManager.userID != userID {
		t.Fatalf("transaction state committed=%v user=%q", txManager.committed, txManager.userID)
	}
	if targets.userID != userID || targets.slug != "toefl-core" {
		t.Fatalf("target activation args user=%q slug=%q", targets.userID, targets.slug)
	}
	if profiles.updatedStatus != usermodel.OnboardingStatusCollectionSelected {
		t.Fatalf("updated onboarding status = %q", profiles.updatedStatus)
	}
	if response.CollectionSlug != "toefl-core" || response.TargetCount != 1000 {
		t.Fatalf("response = %+v", response)
	}
}

func TestActivateLearningCollectionRollsBackWhenOnboardingFails(t *testing.T) {
	expectedErr := errors.New("profile update failed")
	txManager := &fakeActivateCollectionTxManager{repos: fakeActivateCollectionRepositories{
		targets: &fakeTargetCommandRepository{activated: learningmodel.ActivatedUnitCollectionTarget{
			CollectionID:   "11111111-1111-4111-8111-111111111111",
			CollectionSlug: "toefl-core",
			TargetCount:    1000,
		}},
		profiles: &fakeProfileRepository{
			profile:   &usermodel.UserProfile{UserID: userID},
			updateErr: expectedErr,
		},
	}}
	service := apiservice.NewActivateLearningCollectionService(txManager)

	_, err := service.Execute(context.Background(), learningdto.ActivateUnitCollectionTargetRequest{
		UserID:         userID,
		CollectionSlug: "toefl-core",
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("Execute() error = %v, want profile update error", err)
	}
	if txManager.committed {
		t.Fatalf("transaction should not be committed")
	}
}

func TestActivateLearningCollectionMapsMissingCollection(t *testing.T) {
	txManager := &fakeActivateCollectionTxManager{repos: fakeActivateCollectionRepositories{
		targets:  &fakeTargetCommandRepository{err: learningrepo.ErrUnitCollectionNotFound},
		profiles: &fakeProfileRepository{},
	}}
	service := apiservice.NewActivateLearningCollectionService(txManager)

	_, err := service.Execute(context.Background(), learningdto.ActivateUnitCollectionTargetRequest{
		UserID:         userID,
		CollectionSlug: "missing-book",
	})
	if !errors.Is(err, learningservice.ErrUnitCollectionNotFound) {
		t.Fatalf("Execute() error = %v, want ErrUnitCollectionNotFound", err)
	}
}

type fakeActivateCollectionTxManager struct {
	repos     fakeActivateCollectionRepositories
	userID    string
	committed bool
}

func (f *fakeActivateCollectionTxManager) WithinUserTx(ctx context.Context, userID string, fn func(ctx context.Context, repos apirepo.ActivateCollectionRepositories) error) error {
	f.userID = userID
	if err := fn(ctx, f.repos); err != nil {
		return err
	}
	f.committed = true
	return nil
}

type fakeActivateCollectionRepositories struct {
	targets  *fakeTargetCommandRepository
	profiles *fakeProfileRepository
}

func (f fakeActivateCollectionRepositories) TargetCommands() learningrepo.TargetStateCommandRepository {
	return f.targets
}

func (f fakeActivateCollectionRepositories) UserProfiles() userrepo.ProfileRepository {
	return f.profiles
}

type fakeTargetCommandRepository struct {
	userID    string
	slug      string
	activated learningmodel.ActivatedUnitCollectionTarget
	err       error
}

func (f *fakeTargetCommandRepository) EnsureTargetUnits(_ context.Context, _ string, _ []learningmodel.TargetUnitSpec) error {
	return nil
}

func (f *fakeTargetCommandRepository) ActivateUnitCollectionTarget(_ context.Context, userID string, collectionSlug string) (learningmodel.ActivatedUnitCollectionTarget, error) {
	f.userID = userID
	f.slug = collectionSlug
	return f.activated, f.err
}

func (f *fakeTargetCommandRepository) SetTargetInactive(_ context.Context, _ string, _ int64) error {
	return nil
}

type fakeProfileRepository struct {
	profile       *usermodel.UserProfile
	updateErr     error
	updatedStatus string
}

func (f *fakeProfileRepository) GetProfile(_ context.Context, _ string) (usermodel.UserProfile, bool, error) {
	if f.profile == nil {
		return usermodel.UserProfile{}, false, nil
	}
	return *f.profile, true, nil
}

func (f *fakeProfileRepository) RepairProfile(_ context.Context, userID string) (usermodel.UserProfile, error) {
	profile := usermodel.UserProfile{UserID: userID, OnboardingStatus: usermodel.OnboardingStatusNew}
	f.profile = &profile
	return profile, nil
}

func (f *fakeProfileRepository) UpdateTimezone(_ context.Context, _ string, _ string) error {
	return nil
}

func (f *fakeProfileRepository) UpdateOnboardingStatus(_ context.Context, _ string, status string) error {
	f.updatedStatus = status
	return f.updateErr
}

var _ learningrepo.TargetStateCommandRepository = (*fakeTargetCommandRepository)(nil)
var _ userrepo.ProfileRepository = (*fakeProfileRepository)(nil)
