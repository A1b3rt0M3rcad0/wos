//go:build windows

package woscli

import "os"

// Windows replacement and crash behavior are validated on actual Windows CI.
// Directory handles do not expose POSIX fsync through os.File.
func syncDirectory(f *os.File) error { return nil }
