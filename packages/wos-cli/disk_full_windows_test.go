package woscli

import (
	"errors"
	"golang.org/x/sys/windows"
)

func diskFullFixtureError(e error) bool {
	return errors.Is(e, windows.ERROR_DISK_FULL) || errors.Is(e, windows.ERROR_HANDLE_DISK_FULL)
}
