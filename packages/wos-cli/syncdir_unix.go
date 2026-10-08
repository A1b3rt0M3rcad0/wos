//go:build !windows

package woscli

import "os"

func syncDirectory(f *os.File) error { return f.Sync() }
