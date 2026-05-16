//go:build integration

package fixture

import (
	"context"
	"testing"
)

func (db *TestDatabase) SeedUser(t *testing.T, userID string) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `insert into auth.users (id) values ($1)`, userID); err != nil {
		t.Fatalf("seed auth.users: %v", err)
	}
}

func (db *TestDatabase) SeedCoarseUnit(t *testing.T, unitID int64) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `insert into semantic.coarse_unit (id) values ($1)`, unitID); err != nil {
		t.Fatalf("seed semantic.coarse_unit: %v", err)
	}
}

func (db *TestDatabase) SeedQuestion(t *testing.T, questionID string) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `insert into catalog.questions (question_id) values ($1)`, questionID); err != nil {
		t.Fatalf("seed catalog.questions: %v", err)
	}
}
