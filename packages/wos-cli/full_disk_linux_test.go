package woscli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// Opt-in: the runner supplies a disposable 1 MiB tmpfs, never the host disk.
func TestReceiptCleanupRecoversRealFullDisk(t *testing.T) {
	fixture := os.Getenv("WOS_TEST_FULL_DISK_DIRECTORY")
	if fixture == "" {
		t.Skip("requires isolated capacity-limited filesystem")
	}
	root, err := os.MkdirTemp(fixture, "receipt-cleanup-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	project, profile, token, contract, intent, receipt := returnFixtureV2(t)
	w, err := OpenWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(w.CreateV2(".wos/project.yaml", project))
	must(w.CreateV2(profilePathV2(profile.Name), profile))
	path := contractPathV2(profile.Name, *intent.ContractID)
	must(w.CreateV2(path, contract))
	must(w.CreateV2("deliverable.yaml", map[string]string{"summary": "retain independent artifact"}))
	beforeContract, err := w.ReadV2(path)
	must(err)
	beforeProfile, err := w.ReadV2(profilePathV2(profile.Name))
	must(err)
	beforeArtifact, err := w.ReadV2("deliverable.yaml")
	must(err)
	fillerPath := filepath.Join(root, "owned-capacity-filler")
	filler, err := os.Create(fillerPath)
	must(err)
	block := make([]byte, 65536)
	var fillErr error
	for i := 0; i < 32; i++ {
		_, fillErr = filler.Write(block)
		if fillErr != nil {
			break
		}
	}
	must(filler.Close())
	if !errors.Is(fillErr, syscall.ENOSPC) {
		t.Fatalf("runner did not supply the required disposable 1 MiB filesystem: %v", fillErr)
	}
	err = cleanupReturnV2(context.Background(), w, profile, token, intent, receipt)
	if !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("receipt persistence did not encounter real ENOSPC: %v", err)
	}
	for name, expected := range map[string][]byte{path: beforeContract, profilePathV2(profile.Name): beforeProfile, "deliverable.yaml": beforeArtifact} {
		actual, err := w.ReadV2(name)
		must(err)
		if !bytes.Equal(actual, expected) {
			t.Fatalf("full disk changed or deleted retained %s", name)
		}
	}
	must(os.Remove(fillerPath))
	must(cleanupReturnV2(context.Background(), w, profile, token, intent, receipt))
	if _, err := w.ReadV2(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("confirmed cleanup did not remove its exact contract")
	}
	current, err := w.LoadProfileV2(profile.Name)
	must(err)
	if len(current.Local.PendingOperations) != 0 {
		t.Fatal("confirmed receipt recovery retained pending intent")
	}
	artifact, err := w.ReadV2("deliverable.yaml")
	must(err)
	if !bytes.Equal(artifact, beforeArtifact) {
		t.Fatal("recovery changed unrelated artifact")
	}
	t.Log("real ENOSPC preserved persisted acceptance and exact files; freeing owned capacity recovered cleanup without a new return")
}
