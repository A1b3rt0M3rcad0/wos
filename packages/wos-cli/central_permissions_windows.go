package woscli

import (
	"fmt"
	"os"
)

func validateInstallationDirectory(path string) error {
	info, e := os.Lstat(path)
	if e != nil {
		return e
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("installation data directory must be a real directory")
	}
	// Windows access control follows the user's LOCALAPPDATA ACL. Document
	// access still uses the existing protected regular-file/reparse checks.
	return nil
}
