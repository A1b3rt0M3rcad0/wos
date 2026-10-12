package main

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/internal/server"
)

func captureCommand(t *testing.T, args ...string) (int, []byte) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	out, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	os.Stdout, os.Stderr = writer, writer
	defer func() { os.Stdout, os.Stderr = oldOut, oldErr }()
	code := run(args)
	writer.Close()
	data, err := io.ReadAll(out)
	if err != nil {
		t.Fatal(err)
	}
	return code, data
}

func TestUnifiedWOSClientUsesServiceReleaseIdentity(t *testing.T) {
	oldVersion, oldCommit, oldBuilt := server.Version, server.Commit, server.BuiltAt
	defer func() { server.Version, server.Commit, server.BuiltAt = oldVersion, oldCommit, oldBuilt }()
	server.Version, server.Commit, server.BuiltAt = "0.4.0", "0123456789abcdef0123456789abcdef01234567", "2026-10-12T12:00:00Z"
	code, output := captureCommand(t, "version", "--output", "json")
	if code != 0 {
		t.Fatalf("version failed: %s", output)
	}
	var value struct {
		Data struct {
			Version string `json:"version"`
			Commit  string `json:"commit"`
			BuiltAt string `json:"built_at"`
		}
	}
	if err := json.Unmarshal(output, &value); err != nil {
		t.Fatal(err)
	}
	if value.Data.Version != server.Version || value.Data.Commit != server.Commit || value.Data.BuiltAt != server.BuiltAt {
		t.Fatalf("unified version drift: %s", output)
	}
}

func TestUnifiedWOSCreatesAndReadsNativeProject(t *testing.T) {
	root := t.TempDir()
	code, output := captureCommand(t, "project", "init", "--workspace", root, "--server", "http://127.0.0.1:18099", "--server-id", "01a12288-0000-7000-8000-000000000001", "--namespace", "01a12288-0000-7000-8000-000000000002", "--outcome", "01a12288-0000-7000-8000-000000000003", "--output", "json")
	if code != 0 {
		t.Fatalf("project init failed: %s", output)
	}
	if _, err := os.Stat(root + "/.wos/project.yaml"); err != nil {
		t.Fatal(err)
	}
	code, output = captureCommand(t, "work", "checkout", "not-an-id", "--workspace", root, "--profile", "missing", "--output", "json")
	if code == 0 {
		t.Fatal("unconfigured profile gained authority")
	}
	if len(output) == 0 {
		t.Fatal("client error was not surfaced")
	}
}
