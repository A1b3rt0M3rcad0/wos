package woscli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestVersionNeedsNoWorkspaceCredentialsOrService(t *testing.T) {
	var output, errors bytes.Buffer
	code := Run(context.Background(), []string{"version", "--output", "json", "--workspace", filepath.Join(t.TempDir(), "absent")}, &output, &errors)
	if code != 0 || errors.Len() != 0 {
		t.Fatal("version touched workspace or service", code, errors.String())
	}
	var value struct {
		Operation string `json:"operation"`
		Data      struct {
			Version  string `json:"version"`
			Commit   string `json:"commit"`
			Protocol string `json:"work_protocol"`
		}
	}
	if err := json.Unmarshal(output.Bytes(), &value); err != nil || value.Operation != "version" || value.Data.Version != Version || value.Data.Commit != Commit || value.Data.Protocol != "contracts_v1" {
		t.Fatal("invalid release identity", err, output.String())
	}
	output.Reset()
	if Run(context.Background(), []string{"--version"}, &output, &errors) != 0 {
		t.Fatal("standalone version alias failed")
	}
}
