package boundary_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	mainGo := `package main

import (
    "time"

    "github.com/A1b3rt0M3rcad0/wos/core/domain"
    "github.com/A1b3rt0M3rcad0/wos/core/ports"
)

type clock struct{}
func (clock) Now() time.Time { return time.Now().UTC() }

type ids struct{}
func (ids) NewID() (domain.ID, error) {
    return domain.ParseID("0199e100-0000-7000-8000-000000000001")
}

var _ ports.Clock = clock{}
var _ ports.IDGenerator = ids{}

func main() {}
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatalf("write external go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0o600); err != nil {
		t.Fatalf("write external main.go: %v", err)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("external public Core compile failed: %v\n%s", err, output)
	}
}
