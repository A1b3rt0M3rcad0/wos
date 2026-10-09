package woscli

import (
	"bufio"
	"context"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestV2RuntimeLockChild(t *testing.T) {
	root := os.Getenv("WOS_V2_LOCK_FIXTURE")
	if root == "" {
		return
	}
	w, e := OpenWorkspace(root)
	if e != nil {
		os.Exit(2)
	}
	defer w.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	release, e := w.LockV2(ctx, os.Getenv("WOS_V2_LOCK_PROFILE"), "")
	if e != nil {
		os.Exit(9)
	}
	if os.Getenv("WOS_V2_LOCK_HOLD") == "true" {
		fmt.Println("locked")
		select {}
	}
	release()
	os.Exit(0)
}
func TestV2StableLockAcrossProcessesProfilesAndOwnerDeath(t *testing.T) {
	// Native execution is required; the Windows build of this test is not evidence.
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	root := t.TempDir()
	w, e := OpenWorkspace(root)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	executable, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	child := func(profile string, hold bool) *exec.Cmd {
		command := exec.Command(executable, "-test.run=^TestV2RuntimeLockChild$")
		command.Env = append(os.Environ(), "WOS_V2_LOCK_FIXTURE="+root, "WOS_V2_LOCK_PROFILE="+profile, fmt.Sprintf("WOS_V2_LOCK_HOLD=%t", hold))
		return command
	}
	release, e := w.LockV2(context.Background(), "executor_a", "")
	if e != nil {
		t.Fatal(e)
	}
	defer release()
	// Replacing a YAML document cannot replace the locked runtime identity.
	project, _, _ := profileFixtureV2(t, "executor_a")
	if e = w.CreateV2(".wos/project.yaml", project); e != nil {
		t.Fatal(e)
	}
	before, e := w.ReadV2(".wos/project.yaml")
	if e != nil {
		t.Fatal(e)
	}
	project.Name = "Replacement"
	if e = w.WriteV2(".wos/project.yaml", project, signing.Digest(before)); e != nil {
		t.Fatal(e)
	}
	e = child("executor_a", false).Run()
	failure, ok := e.(*exec.ExitError)
	if !ok || failure.ExitCode() != 9 {
		t.Fatalf("another process acquired held profile after rename: %v", e)
	}
	if collision := child("Executor_A", false).Run(); collision == nil {
		t.Fatal("case-colliding reservation bypassed stable lock")
	}
	if e = child("reviewer_b", false).Run(); e != nil {
		t.Fatal("independent profile contended", e)
	}
	release()
	if e = child("executor_a", false).Run(); e != nil {
		t.Fatal("released lock remained", e)
	}
	owner := child("executor_a", true)
	stdout, e := owner.StdoutPipe()
	if e != nil {
		t.Fatal(e)
	}
	if e = owner.Start(); e != nil {
		t.Fatal(e)
	}
	defer func() { owner.Process.Kill(); owner.Wait() }()
	ready := make(chan bool, 1)
	go func() { scanner := bufio.NewScanner(stdout); ready <- scanner.Scan() && scanner.Text() == "locked" }()
	select {
	case ok := <-ready:
		if !ok {
			t.Fatal("owner never acquired")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("owner readiness deadline")
	}
	if e = owner.Process.Kill(); e != nil {
		t.Fatal(e)
	}
	owner.Wait()
	if e = child("executor_a", false).Run(); e != nil {
		t.Fatal("dead owner left permanent lock", e)
	}
	entries, e := os.ReadDir(root)
	if e != nil {
		t.Fatal(e)
	}
	if len(entries) != 1 || entries[0].Name() != ".wos" {
		t.Fatal("runtime lock leaked operational workspace files")
	}
}
