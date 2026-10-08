//go:build !windows

package woscli

import (
	"fmt"
	"os"
	"syscall"
)

func validateRegularV2(file *os.File) error {
	info, e := file.Stat()
	if e != nil {
		return e
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("local v2 file must be regular")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Nlink != 1 {
		return fmt.Errorf("hardlinks or unverifiable file identity are forbidden")
	}
	return nil
}
func validateSecretFile(file *os.File) error {
	if e := validateRegularV2(file); e != nil {
		return e
	}
	info, e := file.Stat()
	if e != nil {
		return e
	}
	if info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("mounted secret must deny group/world access")
	}
	return nil
}
