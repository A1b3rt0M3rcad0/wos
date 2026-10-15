package woscli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReturnResolutionCleanupRecoversRealFullDisk(t *testing.T) {
	fixture := os.Getenv("WOS_TEST_FULL_DISK_DIRECTORY")
	if fixture == "" {
		t.Skip("requires an explicitly isolated capacity-limited filesystem")
	}
	for _, stage := range []string{"proof_persistence", "draft_archived", "archive_persisted", "draft_unfrozen"} {
		t.Run(stage, func(t *testing.T) {
			root, e := os.MkdirTemp(fixture, "resolution-full-disk-")
			if e != nil {
				t.Fatal(e)
			}
			defer os.RemoveAll(root)
			w, profile, token, intent, doc := resolutionLocalFixtureAt(t, root)
			fillerPath := filepath.Join(root, "owned-capacity-filler")
			fill := func() {
				t.Helper()
				f, e := os.Create(fillerPath)
				if e != nil {
					t.Fatal(e)
				}
				var full error
				block := make([]byte, 65536)
				for i := 0; i < 1024; i++ {
					if _, full = f.Write(block); full != nil {
						break
					}
				}
				if e = f.Close(); e != nil {
					t.Fatal(e)
				}
				if !diskFullFixtureError(full) {
					t.Fatalf("fixture must be a disposable volume smaller than 64 MiB: %v", full)
				}
			}
			if stage == "proof_persistence" {
				fill()
				e = updateResolutionStateV2(context.Background(), w, profile, token, intent, "abandoned_confirmed", &doc)
			} else {
				if e = updateResolutionStateV2(context.Background(), w, profile, token, intent, "abandoned_confirmed", &doc); e != nil {
					t.Fatal(e)
				}
				e = cleanupResolutionV2(context.Background(), w, profile, token, intent, doc, func(at string) error {
					if at == stage {
						fill()
					}
					return nil
				})
			}
			if !diskFullFixtureError(e) {
				t.Fatalf("publication did not encounter real full disk at %s: %v", stage, e)
			}
			current, e := w.LoadProfileV2(profile.Name)
			if e != nil {
				t.Fatal(e)
			}
			if len(current.Local.PendingOperations) != 1 || current.Local.PendingOperations[0].Payload != intent.Payload {
				t.Fatal("disk full lost original signature/journal")
			}
			file, _, _, _, e := loadContractV2(w, profile, *intent.ContractID)
			if e != nil {
				t.Fatal(e)
			}
			if file.Execution == nil {
				t.Fatal("disk full removed draft")
			}
			if e = os.Remove(fillerPath); e != nil {
				t.Fatal(e)
			}
			if stage == "proof_persistence" {
				if e = updateResolutionStateV2(context.Background(), w, profile, token, intent, "abandoned_confirmed", &doc); e != nil {
					t.Fatal(e)
				}
			}
			if e = cleanupResolutionV2(context.Background(), w, profile, token, intent, doc, nil); e != nil {
				t.Fatal(e)
			}
			current, e = w.LoadProfileV2(profile.Name)
			if e != nil {
				t.Fatal(e)
			}
			if len(current.Local.PendingOperations) != 0 {
				t.Fatal("full disk recovery remained pending")
			}
		})
	}
}
