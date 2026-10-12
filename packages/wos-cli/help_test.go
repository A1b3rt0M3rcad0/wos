package woscli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpDiscoveryNeverRequiresWorkspaceCredentialsOrServer(t *testing.T) {
	root := filepath.Join(t.TempDir(), "absent-workspace")
	t.Setenv("WOS_WORKSPACE", root)
	t.Setenv("WOS_PROFILE", "missing-profile")
	t.Setenv("WOS_TOKEN", "credential-canary-must-never-appear")
	for _, args := range [][]string{{"--help"}, {"-h"}, {"help", "outcome"}, {"work", "checkout", "--help"}, {"--profile", "missing-profile", "review", "--help"}, {"catalog", "--output", "json"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var out, diagnostic bytes.Buffer
			if code := Run(context.Background(), args, &out, &diagnostic); code != 0 {
				t.Fatalf("offline discovery failed: %d %s", code, diagnostic.String())
			}
			if out.Len() == 0 || diagnostic.Len() != 0 || strings.Contains(out.String(), os.Getenv("WOS_TOKEN")) {
				t.Fatal("empty help, unexpected diagnostic or credential exposure")
			}
		})
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("local discovery touched the absent workspace", err)
	}
}

func TestHelpCatalogBindsVersionAndSeparatesClientFromServiceCommands(t *testing.T) {
	for _, service := range []bool{false, true} {
		var out, diagnostic bytes.Buffer
		handled, code := HandleHelp([]string{"catalog", "--output", "json"}, &out, &diagnostic, service)
		if !handled || code != 0 {
			t.Fatal("catalog failed", diagnostic.String())
		}
		var result struct {
			Data CommandCatalog `json:"data"`
		}
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Data.SchemaVersion != 1 || result.Data.Version != Version || result.Data.Commit != Commit {
			t.Fatal("catalog identity does not match the executable")
		}
		found := map[string]bool{}
		for _, group := range result.Data.Groups {
			found[group.Name] = true
		}
		if !found["work"] || !found["outcome"] || found["server"] != service || found["skill"] != service {
			t.Fatal("catalog advertises unavailable service operations or omits client operations")
		}
	}
}

func TestOutcomeContextErrorGuidesBothWorkspaceProtocolsBeforeAccess(t *testing.T) {
	var out, diagnostic bytes.Buffer
	code := Run(context.Background(), []string{"outcome", "context", "--workspace", filepath.Join(t.TempDir(), "absent"), "--output", "json"}, &out, &diagnostic)
	if code != 2 || !strings.Contains(diagnostic.String(), "outcome get") || !strings.Contains(diagnostic.String(), "work show") || !strings.Contains(diagnostic.String(), "wos help") {
		t.Fatal("unsupported context command did not explain supported alternatives", code, diagnostic.String())
	}
}

func TestUnknownHelpAndInvalidCatalogReturnUsageWithoutSideEffects(t *testing.T) {
	for _, args := range [][]string{{"help", "missing"}, {"catalog", "work"}, {"--help", "--output", "invalid"}} {
		var out, diagnostic bytes.Buffer
		if handled, code := HandleHelp(args, &out, &diagnostic, true); !handled || code != 2 || diagnostic.Len() == 0 {
			t.Fatal("invalid discovery did not return a useful usage error", args, code)
		}
	}
}
