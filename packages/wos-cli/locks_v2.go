package woscli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"path/filepath"
	"runtime"
	"strings"
)

// LockV2 protects a stable runtime identity, never the replaceable YAML inode.
// Callers acquire profile first and then one contract; network operations must
// not hold a profile lock except replay of an already frozen profile intention.
func (w *Workspace) LockV2(ctx context.Context, profile, contract string) (func(), error) {
	if !validProfileName(profile) {
		return nil, fmt.Errorf("invalid lock profile")
	}
	if contract != "" && !validContractLockIDV2(contract) {
		return nil, fmt.Errorf("invalid lock contract")
	}
	root := filepath.Clean(w.canonicalPath)
	if runtime.GOOS == "windows" {
		root = strings.ToLower(root)
	}
	// Even on a case-sensitive filesystem, case-colliding profile reservations
	// share one lock. Selection still requires the exact existing directory name.
	raw, _ := json.Marshal([]string{"WOS stable workspace lock v2", root, strings.ToLower(profile), contract})
	hash := sha256.Sum256(raw)
	return acquireRuntimeLockV2(ctx, hex.EncodeToString(hash[:]))
}

func validContractLockIDV2(value string) bool { return d.ID(value).Validate() == nil }
