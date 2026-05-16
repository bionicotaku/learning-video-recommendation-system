//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"testing"

	"learning-video-recommendation-system/internal/test/e2e/testutil"
)

var sharedHarness *testutil.Harness

func TestMain(m *testing.M) {
	var err error
	sharedHarness, err = testutil.OpenHarness()
	if err != nil {
		fmt.Fprintf(os.Stderr, "open e2e harness: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := sharedHarness.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "close e2e harness: %v\n", err)
		}
	}()

	os.Exit(m.Run())
}

func harness(t *testing.T) *testutil.Harness {
	t.Helper()
	if sharedHarness == nil {
		t.Fatal("shared e2e harness is not initialized")
	}
	return sharedHarness
}
