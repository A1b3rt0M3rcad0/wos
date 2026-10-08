package woscli

import (
	"os"
	"os/exec"
	"testing"
)

func TestLockChildHelper(t *testing.T) {
	root := os.Getenv("WOS_CLI_LOCK_FIXTURE")
	if root == "" {
		return
	}
	w, err := OpenWorkspace(root)
	if err != nil {
		os.Exit(2)
	}
	defer w.Close()
	release, err := w.Lock(".wos/process-lock")
	if err != nil {
		os.Exit(9)
	}
	release()
	os.Exit(0)
}
func TestWorkspaceLockExcludesAnotherNativeProcess(t *testing.T) {
	root := t.TempDir()
	w, err := OpenWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	release, err := w.Lock(".wos/process-lock")
	if err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	child := func() error {
		cmd := exec.Command(exe, "-test.run=^TestLockChildHelper$")
		cmd.Env = append(os.Environ(), "WOS_CLI_LOCK_FIXTURE="+root)
		return cmd.Run()
	}
	err = child()
	failure, ok := err.(*exec.ExitError)
	if !ok || failure.ExitCode() != 9 {
		t.Fatalf("another process obtained lock: %v", err)
	}
	release()
	if err = child(); err != nil {
		t.Fatalf("lock remained after owner released: %v", err)
	}
}
