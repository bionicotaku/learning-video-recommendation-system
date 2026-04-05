package policy

import "testing"

func TestDefaultLearningPolicyUsesDocumentedDefaults(t *testing.T) {
	got := DefaultLearningPolicy()

	if got.MasteredIntervalDays != DefaultMasteredIntervalDays {
		t.Fatalf("MasteredIntervalDays = %v, want %v", got.MasteredIntervalDays, DefaultMasteredIntervalDays)
	}
	if got.MinEaseFactor != DefaultMinEaseFactor {
		t.Fatalf("MinEaseFactor = %v, want %v", got.MinEaseFactor, DefaultMinEaseFactor)
	}

	wantIntervals := []float64{1, 3, 6}
	if len(got.InitialIntervals) != len(wantIntervals) {
		t.Fatalf("len(InitialIntervals) = %d, want %d", len(got.InitialIntervals), len(wantIntervals))
	}
	for index, want := range wantIntervals {
		if got.InitialIntervals[index] != want {
			t.Fatalf("InitialIntervals[%d] = %v, want %v", index, got.InitialIntervals[index], want)
		}
	}
}

func TestDefaultLearningPolicyReturnsCopiedIntervals(t *testing.T) {
	first := DefaultLearningPolicy()
	first.InitialIntervals[0] = 99

	second := DefaultLearningPolicy()
	if second.InitialIntervals[0] != 1 {
		t.Fatalf("InitialIntervals[0] = %v, want 1 after prior mutation", second.InitialIntervals[0])
	}
}
