//go:build integration

package infrastructure_test

import (
	"os"
	"testing"

	"learning-video-recommendation-system/internal/analytics/test/fixture"
)

var suite *fixture.Suite

func TestMain(m *testing.M) {
	var err error
	suite, err = fixture.OpenSuite()
	if err != nil {
		panic(err)
	}
	code := m.Run()
	if err := suite.Close(); err != nil {
		panic(err)
	}
	os.Exit(code)
}

func testDB(t *testing.T) *fixture.TestDatabase {
	t.Helper()
	return suite.CreateTestDatabase(t)
}
