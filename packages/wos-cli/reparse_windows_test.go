//go:build windows

package woscli

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestWorkspaceRejectsWindowsJunction(t *testing.T) {
	root, target := t.TempDir(), t.TempDir()
	junction := filepath.Join(root, "junction")
	if output, err := exec.Command("cmd", "/c", "mklink", "/J", junction, target).CombinedOutput(); err != nil {
		t.Fatalf("required actual Windows junction fixture failed: %v %s", err, output)
	}
	w, err := OpenWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err = w.AtomicWrite("junction/escape.json", []byte("bad")); err == nil {
		t.Fatal("Windows junction escaped workspace")
	}
}
