//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"learning-video-recommendation-system/internal/test/e2e/testutil"
)

var sharedHarness *testutil.Harness

func TestMain(m *testing.M) {
	baseDir, err := os.MkdirTemp("", "learning-recommendation-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create e2e temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(baseDir)

	sharedHarness, err = testutil.OpenHarness(filepath.Join(baseDir, "embedded-postgres"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "open e2e harness: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := sharedHarness.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "close e2e harness: %v\n", err)
		}
	}()

	if err := sharedHarness.ApplySchemaForMain(); err != nil {
		fmt.Fprintf(os.Stderr, "apply e2e schema: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func harness(t *testing.T) *testutil.Harness {
	t.Helper()
	if sharedHarness == nil {
		t.Fatal("shared e2e harness is not initialized")
	}
	return sharedHarness
}
