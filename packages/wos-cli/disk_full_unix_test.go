//go:build !windows

package woscli

import (
	"errors"
	"syscall"
)

func diskFullFixtureError(e error) bool { return errors.Is(e, syscall.ENOSPC) }
