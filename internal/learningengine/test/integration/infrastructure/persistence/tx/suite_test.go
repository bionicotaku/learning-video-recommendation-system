//go:build integration

package tx_test

import (
	"fmt"
	"os"
	"testing"

	"learning-video-recommendation-system/internal/learningengine/test/fixture"
)

var sharedSuite *fixture.Suite

func TestMain(m *testing.M) {
	suite, err := fixture.OpenSuite()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "open learningengine tx suite: %v\n", err)
		os.Exit(1)
	}
	sharedSuite = suite

	code := m.Run()
	if err := suite.Close(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "close learningengine tx suite: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

func testDB(t *testing.T) *fixture.TestDatabase {
	t.Helper()
	if sharedSuite == nil {
		t.Fatal("learningengine tx suite not initialized")
	}
	return sharedSuite.CreateTestDatabase(t)
}
