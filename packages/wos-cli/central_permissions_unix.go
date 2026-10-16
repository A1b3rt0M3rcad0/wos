//go:build !windows

package woscli

import (
	"fmt"
	"os"
	"syscall"
)

func validateInstallationDirectory(path string) error {
	info, e := os.Lstat(path)
	if e != nil {
		return e
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 || int(owner.Uid) != os.Geteuid() {
		return fmt.Errorf("installation data directory must be owned by the current user and deny group/world access")
	}
	return nil
}
