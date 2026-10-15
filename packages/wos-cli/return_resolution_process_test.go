package woscli

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

func TestV2ResolutionCleanupChild(t *testing.T) {
	root := os.Getenv("WOS_RESOLUTION_CLEANUP_FIXTURE")
	if root == "" {
		return
	}
	_, fixture, token := profileFixtureV2(t, "executor_a")
	w, e := OpenWorkspace(root)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	profile, e := w.LoadProfileV2(fixture.Name)
	if e != nil {
		t.Fatal(e)
	}
	if e = profile.VerifyBinding(token); e != nil {
		t.Fatal(e)
	}
	if len(profile.Local.PendingOperations) != 1 {
		t.Fatal("missing original journal")
	}
	intent := profile.Local.PendingOperations[0]
	raw, e := base64.StdEncoding.Strict().DecodeString(intent.Resolution.Response)
	if e != nil {
		t.Fatal(e)
	}
	var doc signing.Document
	if e = signing.DecodeStrict(raw, &doc, 32<<10); e != nil {
		t.Fatal(e)
	}
	e = cleanupResolutionV2(context.Background(), w, profile, token, intent, doc, func(at string) error {
		if at == os.Getenv("WOS_RESOLUTION_CLEANUP_POINT") {
			os.Exit(23)
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
}

func TestReturnResolutionRecoversNativeProcessDeath(t *testing.T) {
	for _, stage := range []string{"draft_archived", "archive_persisted", "draft_unfrozen"} {
		t.Run(stage, func(t *testing.T) {
			w, profile, token, intent, doc := resolutionLocalFixture(t)
			if e := updateResolutionStateV2(context.Background(), w, profile, token, intent, "abandoned_confirmed", &doc); e != nil {
				t.Fatal(e)
			}
			child := exec.Command(os.Args[0], "-test.run=^TestV2ResolutionCleanupChild$")
			child.Env = append(os.Environ(), "WOS_RESOLUTION_CLEANUP_FIXTURE="+w.canonicalPath, "WOS_RESOLUTION_CLEANUP_POINT="+stage)
			output, e := child.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(e, &exit) || exit.ExitCode() != 23 {
				t.Fatalf("child did not die at %s: %v %s", stage, e, output)
			}
			current, e := w.LoadProfileV2(profile.Name)
			if e != nil {
				t.Fatal(e)
			}
			if len(current.Local.PendingOperations) != 1 {
				t.Fatal("death discarded durable proof")
			}
			if _, confirmed, e := recoverReturnV2(context.Background(), w, current, nil, token, current.Local.PendingOperations[0], false); e != nil || !confirmed {
				t.Fatalf("offline proof recovery failed after real process death: %v", e)
			}
			file, _, _, _, e := loadContractV2(w, profile, *intent.ContractID)
			if e != nil {
				t.Fatal(e)
			}
			if file.Local.Pending != nil || file.Execution == nil {
				t.Fatal("recovery lost editable contract")
			}
		})
	}
}
