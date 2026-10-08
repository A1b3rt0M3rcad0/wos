package woscli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type KeyringBackend interface {
	Get(string, string) (string, error)
	Set(string, string, string) error
	Delete(string, string) error
}
type SecretResolver struct {
	WorkspaceRoot string
	Keyring       KeyringBackend
	Environment   func(string) string
}

func validSecretReference(ref string) bool {
	if regexp.MustCompile(`^env:[A-Za-z_][A-Za-z0-9_]*$`).MatchString(ref) {
		return true
	}
	if strings.HasPrefix(ref, "keyring:wos/") {
		parts := strings.Split(strings.TrimPrefix(ref, "keyring:wos/"), "/")
		return len(parts) == 2 && validProfileName(parts[0]) && (parts[1] == "api" || parts[1] == "signing")
	}
	if strings.HasPrefix(ref, "mounted:") {
		path := strings.TrimPrefix(ref, "mounted:")
		return filepath.IsAbs(path) && filepath.Clean(path) == path
	}
	return false
}
func (r SecretResolver) Read(ref string) ([]byte, error) {
	if !validSecretReference(ref) {
		return nil, fmt.Errorf("invalid secret reference")
	}
	var value []byte
	switch {
	case strings.HasPrefix(ref, "env:"):
		env := r.Environment
		if env == nil {
			env = os.Getenv
		}
		value = []byte(env(strings.TrimPrefix(ref, "env:")))
	case strings.HasPrefix(ref, "keyring:"):
		if r.Keyring == nil {
			return nil, fmt.Errorf("OS keyring unavailable; use an explicit env or protected mounted reference")
		}
		raw, e := r.Keyring.Get("wos", strings.TrimPrefix(ref, "keyring:wos/"))
		if e != nil {
			return nil, fmt.Errorf("OS keyring secret unavailable")
		}
		value = []byte(raw)
	case strings.HasPrefix(ref, "mounted:"):
		path := strings.TrimPrefix(ref, "mounted:")
		canonical, e := filepath.EvalSymlinks(path)
		if e != nil || canonical != path {
			return nil, fmt.Errorf("mounted secret must use a canonical nonsymlink path")
		}
		root, e := filepath.EvalSymlinks(r.WorkspaceRoot)
		if e != nil {
			return nil, fmt.Errorf("workspace root unavailable")
		}
		relative, e := filepath.Rel(root, path)
		if e != nil && strings.EqualFold(filepath.VolumeName(root), filepath.VolumeName(path)) || e == nil && filepath.IsLocal(relative) {
			return nil, fmt.Errorf("mounted secret must be outside the repository workspace")
		}
		file, e := os.Open(path)
		if e != nil {
			return nil, fmt.Errorf("mounted secret unavailable")
		}
		defer file.Close()
		if e = validateSecretFile(file); e != nil {
			return nil, e
		}
		value, e = io.ReadAll(io.LimitReader(file, 8193))
		if e != nil {
			return nil, fmt.Errorf("mounted secret read failed")
		}
	}
	if len(value) == 0 || len(value) > 8192 {
		clear(value)
		return nil, fmt.Errorf("secret unavailable or exceeds bounded secret size")
	}
	// Permit the single terminal newline used by explicitly mounted secrets.
	value = bytes.TrimSuffix(bytes.TrimSuffix(value, []byte("\n")), []byte("\r"))
	if len(value) == 0 {
		return nil, fmt.Errorf("secret unavailable")
	}
	return value, nil
}
func (r SecretResolver) Store(ref string, secret []byte) error {
	if !strings.HasPrefix(ref, "keyring:wos/") || !validSecretReference(ref) {
		return fmt.Errorf("only OS keyring references can be provisioned; env/mounted secrets are supplied externally")
	}
	if r.Keyring == nil {
		return fmt.Errorf("OS keyring unavailable; no plaintext fallback is permitted")
	}
	if len(secret) == 0 || len(secret) > 8192 {
		return fmt.Errorf("invalid secret size")
	}
	if e := r.Keyring.Set("wos", strings.TrimPrefix(ref, "keyring:wos/"), string(secret)); e != nil {
		return fmt.Errorf("OS keyring provisioning failed")
	}
	return nil
}
