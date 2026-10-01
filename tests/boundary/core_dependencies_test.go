package boundary_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestCoreDoesNotImportForbiddenLayers(t *testing.T) {
	root := repositoryRoot(t)
	coreRoot := filepath.Join(root, "core")

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
			assertAllowedImport(t, path, coreRoot, importPath)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk core imports: %v", err)
	}
}

func assertAllowedImport(t *testing.T, file, coreRoot, importPath string) {
	t.Helper()

	forbidden := []string{
		"net/http",
		"database/sql",
		"github.com/A1b3rt0M3rcad0/wos/internal/",
		"github.com/A1b3rt0M3rcad0/wos/storage/",
	}
	for _, prefix := range forbidden {
		if importPath == prefix || strings.HasPrefix(importPath, prefix) {
			t.Errorf("%s imports forbidden dependency %q", file, importPath)
		}
	}
	if strings.Contains(strings.ToLower(importPath), "woobe") && importPath != "github.com/A1b3rt0M3rcad0/wos/core/domain" {
		t.Errorf("%s imports runtime/product dependency %q", file, importPath)
	}

	rel, err := filepath.Rel(coreRoot, file)
	if err != nil {
		t.Fatalf("relative core path: %v", err)
	}
	if strings.HasPrefix(filepath.ToSlash(rel), "domain/") && strings.HasPrefix(importPath, "github.com/A1b3rt0M3rcad0/wos/") {
		t.Errorf("domain package must not depend on another WOS layer: %s imports %q", file, importPath)
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
