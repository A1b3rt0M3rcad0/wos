package wosskill

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeSkillInstallInspectUpdateAndRemove(t *testing.T) {
	root := t.TempDir()
	opts := Options{Agent: "codex", Project: root}
	preview, err := Operate("install", Options{Agent: "codex", Project: root, DryRun: true})
	if err != nil || len(preview) != 5 || preview[0].Action != "install" {
		t.Fatalf("unexpected dry-run: %+v, %v", preview, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".agents")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run wrote files: %v", err)
	}
	installed, err := Operate("install", opts)
	if err != nil || len(installed) != 5 {
		t.Fatalf("install: %+v %v", installed, err)
	}
	for _, action := range installed {
		if action.Action != "install" {
			t.Fatalf("unexpected action: %+v", action)
		}
		if _, err := os.Stat(filepath.Join(action.Target, "SKILL.md")); err != nil {
			t.Fatal(err)
		}
		for _, filename := range legalFiles {
			expected, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(filename)))
			if err != nil {
				t.Fatal(err)
			}
			actual, err := os.ReadFile(filepath.Join(action.Target, filepath.FromSlash(filename)))
			if err != nil || !bytes.Equal(actual, expected) {
				t.Fatalf("installed legal file differs from source: %s, %v", filename, err)
			}
		}
	}
	check, err := Operate("doctor", opts)
	if err != nil || check[0].Action != "installed" {
		t.Fatalf("doctor: %+v %v", check, err)
	}
	repeated, err := Operate("install", opts)
	if err != nil || repeated[0].Action != "unchanged" {
		t.Fatalf("repeated: %+v %v", repeated, err)
	}
	marker := filepath.Join(installed[0].Target, markerFile)
	var m manifest
	content, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, &m); err != nil {
		t.Fatal(err)
	}
	m.Version = "0.0.1"
	content, _ = json.Marshal(m)
	if err := os.WriteFile(marker, content, 0o644); err != nil {
		t.Fatal(err)
	}
	updated, err := Operate("update", opts)
	if err != nil || updated[0].Action != "update" {
		t.Fatalf("update: %+v %v", updated, err)
	}
	for _, action := range updated[1:] {
		if action.Action != "unchanged" {
			t.Fatalf("unexpected update: %+v", action)
		}
	}
	removed, err := Operate("uninstall", opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range removed {
		if action.Action != "remove" {
			t.Fatalf("expected removal: %+v", action)
		}
		if _, err := os.Stat(action.Target); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("not removed: %s %v", action.Target, err)
		}
	}
}

func TestNativeInstallerUpgradesManagedBundlesWithoutLegalFiles(t *testing.T) {
	opts := Options{Project: t.TempDir(), Agent: "codex"}
	installed, err := Operate("install", opts)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate an intact older installer bundle: its ownership manifest has no
	// legal files. Updating it must add notices without treating it as edited.
	for _, item := range installed {
		marker := filepath.Join(item.Target, markerFile)
		content, err := os.ReadFile(marker)
		if err != nil {
			t.Fatal(err)
		}
		var m manifest
		if err := json.Unmarshal(content, &m); err != nil {
			t.Fatal(err)
		}
		for _, filename := range legalFiles {
			if err := os.Remove(filepath.Join(item.Target, filepath.FromSlash(filename))); err != nil {
				t.Fatal(err)
			}
			delete(m.Files, filename)
		}
		content, err = json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(marker, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	updated, err := Operate("update", opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range updated {
		if item.Action != "update" {
			t.Fatalf("missing legal files were not restored: %+v", item)
		}
	}
	for _, filename := range legalFiles {
		if _, err := os.Stat(filepath.Join(updated[0].Target, filepath.FromSlash(filename))); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNativeInstallerPreservesModifiedLegalFiles(t *testing.T) {
	opts := Options{Project: t.TempDir(), Agent: "codex"}
	installed, err := Operate("install", opts)
	if err != nil {
		t.Fatal(err)
	}
	notice := filepath.Join(installed[0].Target, "NOTICE.md")
	if err := os.WriteFile(notice, []byte("local notice"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"update", "uninstall"} {
		if _, err := Operate(command, opts); err == nil || !strings.Contains(err.Error(), "locally modified") {
			t.Fatalf("%s replaced edited legal material: %v", command, err)
		}
	}
	if data, err := os.ReadFile(notice); err != nil || string(data) != "local notice" {
		t.Fatalf("modified notice lost: %q %v", data, err)
	}
}

func TestNativeInstallerGuardsAllDestinationsBeforeWrites(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "skills", "wos-review")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Operate("install", Options{Agent: "all", Project: root}); err == nil || !strings.Contains(err.Error(), "Unmanaged") && !strings.Contains(err.Error(), "unmanaged") {
		t.Fatalf("expected unmanaged collision: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "skills", skillNames[0])); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial installation: %v", err)
	}
}

func TestNativeInstallerPreservesEditsAndUnexpectedFiles(t *testing.T) {
	root := t.TempDir()
	opts := Options{Project: root, Agent: "codex"}
	p, err := Operate("install", opts)
	if err != nil {
		t.Fatal(err)
	}
	target := p[0].Target
	if err := os.WriteFile(filepath.Join(target, "personal.txt"), []byte("preserve"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"update", "uninstall"} {
		_, err := Operate(command, opts)
		if err == nil || !strings.Contains(err.Error(), "locally modified") {
			t.Fatalf("%s should reject modified skill: %v", command, err)
		}
	}
	if data, err := os.ReadFile(filepath.Join(target, "personal.txt")); err != nil || string(data) != "preserve" {
		t.Fatalf("user file lost: %q %v", data, err)
	}
}

func TestNativeInstallerRejectsSymlinkParents(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	link := filepath.Join(root, ".agents")
	if err := os.Symlink(other, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, err := Operate("install", Options{Project: root})
	if err == nil || !strings.Contains(err.Error(), "unsafe directory") {
		t.Fatalf("expected unsafe path rejection: %v", err)
	}
}

func TestNativeSkillCLISubcommand(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Run([]string{"install", "--agent", "codex", "--project", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("CLI install failed: %d %s", code, stderr.String())
	}
	var results []Result
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil || len(results) != 5 {
		t.Fatalf("bad JSON: %v %+v", err, results)
	}
	if !strings.HasPrefix(results[0].Target, filepath.Join(root, ".agents", "skills")) {
		t.Fatalf("incorrect target: %s", results[0].Target)
	}
	stdout.Reset()
	stderr.Reset()
	if c := Run([]string{"version"}, &stdout, &stderr); c != 0 || strings.TrimSpace(stdout.String()) == "" {
		t.Fatalf("version: %d %s", c, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if c := Run([]string{"install", "--agent", "not-an-agent"}, &stdout, &stderr); c == 0 {
		t.Fatal("invalid agent accepted")
	}
	stdout.Reset()
	stderr.Reset()
	if c := Run([]string{"--help"}, &stdout, &stderr); c != 0 || !strings.Contains(stdout.String(), "No Node.js") {
		t.Fatalf("help: %d %s", c, stdout.String())
	}
}
