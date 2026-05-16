package mapper_test

import (
	"testing"
	"time"

	"learning-video-recommendation-system/internal/analytics/infrastructure/persistence/mapper"
)

func TestTimePointerToPGNormalizesToUTC(t *testing.T) {
	localTime := time.Date(2026, 5, 15, 10, 0, 0, 0, time.FixedZone("PDT", -7*60*60))

	got := mapper.TimePointerToPG(&localTime)

	if !got.Valid {
		t.Fatalf("TimePointerToPG() valid = false, want true")
	}
	if got.Time.Location() != time.UTC {
		t.Fatalf("TimePointerToPG() location = %v, want UTC", got.Time.Location())
	}
	if !got.Time.Equal(localTime) {
		t.Fatalf("TimePointerToPG() = %v, want same instant as %v", got.Time, localTime)
	}
}
