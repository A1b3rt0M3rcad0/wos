package boundary_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/A1b3rt0M3rcad0/wos"

func TestCoreDoesNotImportForbiddenLayers(t *testing.T) {
	root := repositoryRoot(t)
	coreRoot := filepath.Join(root, "packages", "wos-core")

	err := filepath.WalkDir(coreRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			assertAllowedCoreImport(t, path, coreRoot, importPath)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk core imports: %v", err)
	}
}

func assertAllowedCoreImport(t *testing.T, file, coreRoot, importPath string) {
	t.Helper()

	for _, prefix := range []string{
		"net/http",
		modulePath + "/packages/wos-api",
	} {
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			t.Errorf("%s imports forbidden dependency %q", file, importPath)
		}
	}

	rel, err := filepath.Rel(coreRoot, file)
	if err != nil {
		t.Fatalf("relative core path: %v", err)
	}
	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, "storage/") && importPath == "database/sql" {
		t.Errorf("%s imports database/sql outside storage adapters", file)
	}
	if strings.HasPrefix(rel, "domain/") && strings.HasPrefix(importPath, modulePath+"/") {
		t.Errorf("domain package must not depend on another WOS layer: %s imports %q", file, importPath)
	}
	if strings.HasPrefix(rel, "application/") && strings.HasPrefix(importPath, modulePath+"/packages/wos-core/storage/") {
		t.Errorf("application must depend on ports, not storage adapters: %s imports %q", file, importPath)
	}
}

func TestLegacyProductRootsAreRemoved(t *testing.T) {
	root := repositoryRoot(t)
	for _, legacy := range []string{"core", "api", "storage", "internal", "cmd"} {
		_, err := os.Stat(filepath.Join(root, legacy))
		if err == nil {
			t.Errorf("legacy product root %q still exists", legacy)
			continue
		}
		if !os.IsNotExist(err) {
			t.Fatalf("stat legacy product root %q: %v", legacy, err)
		}
	}
}

func TestStandaloneServerPackageIsRemoved(t *testing.T) {
	root := repositoryRoot(t)
	_, err := os.Stat(filepath.Join(root, "packages", "wos-server"))
	if err == nil {
		t.Fatal("packages/wos-server must not exist; standalone runtime belongs to wos-api")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("stat packages/wos-server: %v", err)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve boundary test path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
