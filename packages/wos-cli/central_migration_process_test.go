package woscli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var centralCopyFixture = []byte(`{"frozen":"complete source document, signatures and pending bytes remain identical"}`)

func TestCentralMigrationCopyChild(t *testing.T) {
	path := os.Getenv("WOS_CENTRAL_COPY_CHILD")
	if path == "" {
		return
	}
	w, e := OpenWorkspace(path)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	target := ".wos/contract.yaml"
	e = w.publishRawV2(target, migrationCopyStage(target, centralCopyFixture), centralCopyFixture, func(point string) error {
		if point == os.Getenv("WOS_CENTRAL_COPY_POINT") {
			os.Exit(27)
		}
		return nil
	})
	t.Fatal("child should have exited at publication fault", e)
}

func TestCentralMigrationCopyRecoversNativeProcessDeath(t *testing.T) {
	for _, point := range []string{"staged", "published", "unlinked"} {
		t.Run(point, func(t *testing.T) {
			root := t.TempDir()
			if e := os.Mkdir(filepath.Join(root, ".wos"), 0700); e != nil {
				t.Fatal(e)
			}
			// Exceed the ordinary reader's 112-entry alias-discovery window.
			// Migration already knows its exact stage and must not scan for it.
			for n := 0; n < 150; n++ {
				if e := os.WriteFile(filepath.Join(root, ".wos", fmt.Sprintf("existing-%03d.json", n)), []byte("{}"), 0600); e != nil {
					t.Fatal(e)
				}
			}
			child := exec.Command(os.Args[0], "-test.run=^TestCentralMigrationCopyChild$")
			child.Env = append(os.Environ(), "WOS_CENTRAL_COPY_CHILD="+root, "WOS_CENTRAL_COPY_POINT="+point)
			output, e := child.CombinedOutput()
			if exit, ok := e.(*exec.ExitError); !ok || exit.ExitCode() != 27 {
				t.Fatalf("expected simulated process death, got %v: %s", e, output)
			}
			w, e := OpenWorkspace(root)
			if e != nil {
				t.Fatal(e)
			}
			defer w.Close()
			target := ".wos/contract.yaml"
			if e = migrationCopyFrozenFile(w, target, centralCopyFixture); e != nil {
				t.Fatal("exact frozen copy failed recovery", e)
			}
			actual, e := migrationRead(w, target)
			if e != nil || !bytes.Equal(actual, centralCopyFixture) {
				t.Fatal("native crash changed copied bytes", e)
			}
			if _, e = os.Lstat(filepath.Join(root, migrationCopyStage(target, centralCopyFixture))); !os.IsNotExist(e) {
				t.Fatal("recognized technical alias survived confirmed copy", e)
			}
		})
	}
}

func TestCentralMigrationCopyRepairsOnlyFrozenConstructorPrefix(t *testing.T) {
	for _, edited := range []bool{false, true} {
		t.Run(fmt.Sprint(edited), func(t *testing.T) {
			w, e := OpenWorkspace(t.TempDir())
			if e != nil {
				t.Fatal(e)
			}
			defer w.Close()
			target := ".wos/contract.yaml"
			stage := migrationCopyStage(target, centralCopyFixture)
			partial := centralCopyFixture[:17]
			if edited {
				partial = []byte("different operator edit")
			}
			if e = w.writeRawV2(stage, partial, "", true); e != nil {
				t.Fatal(e)
			}
			e = migrationCopyFrozenFile(w, target, centralCopyFixture)
			if edited {
				if e == nil {
					t.Fatal("non-prefix staging edit overwritten")
				}
				actual, readErr := migrationRead(w, stage)
				if readErr != nil || !bytes.Equal(actual, partial) {
					t.Fatal("observed edit was lost", readErr)
				}
			} else if e != nil {
				t.Fatal("unchanged constructor prefix did not recover", e)
			}
		})
	}
}
