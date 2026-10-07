package boundary_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicCoreCompilesFromExternalModule(t *testing.T) {
	root := repositoryRoot(t)
	dir := t.TempDir()

	goMod := fmt.Sprintf(`module example.com/wos-external-contract

go 1.27

require github.com/A1b3rt0M3rcad0/wos v0.0.0
replace github.com/A1b3rt0M3rcad0/wos => %s
`, filepath.ToSlash(root))
	mainGo, err := os.ReadFile(filepath.Join(root, "examples", "embedded", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatalf("write external go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), mainGo, 0o600); err != nil {
		t.Fatalf("write external main.go: %v", err)
	}

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("external public Core compile failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "lifecycle=achieved revision=5 certified=true") {
		t.Fatalf("external host did not certify and read the result: %s", output)
	}
}
