//go:build !windows

package woscli

import (
	"os"
	"testing"
)

func protectSecretFixtureV2(t *testing.T, path string) {
	t.Helper()
	if e := os.Chmod(path, 0600); e != nil {
		t.Fatal(e)
	}
}
