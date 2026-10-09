//go:build !windows

package woscli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

func acquireRuntimeLockV2(ctx context.Context, identity string) (func(), error) {
	cache, e := os.UserCacheDir()
	if e != nil {
		return nil, e
	}
	directory := filepath.Join(cache, "wos-runtime-locks-v2")
	if e = os.MkdirAll(directory, 0700); e != nil {
		return nil, e
	}
	canonical, e := filepath.EvalSymlinks(directory)
	if e != nil || canonical != directory {
		return nil, fmt.Errorf("runtime lock directory must be canonical")
	}
	info, e := os.Lstat(directory)
	if e != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("runtime lock directory must deny group/world access")
	}
	if st, ok := info.Sys().(*syscall.Stat_t); !ok || int(st.Uid) != os.Geteuid() {
		return nil, fmt.Errorf("runtime lock directory owner differs")
	}
	fd, e := syscall.Open(filepath.Join(directory, identity+".lock"), syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if e != nil {
		return nil, e
	}
	file := os.NewFile(uintptr(fd), "wos runtime lock")
	if e = validateSecretFile(file); e != nil {
		file.Close()
		return nil, e
	}
	st, e := file.Stat()
	if e != nil {
		file.Close()
		return nil, e
	}
	if owner, ok := st.Sys().(*syscall.Stat_t); !ok || int(owner.Uid) != os.Geteuid() {
		file.Close()
		return nil, fmt.Errorf("runtime lock owner differs")
	}
	for {
		if e = ctx.Err(); e != nil {
			file.Close()
			return nil, e
		}
		e = syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
		if e == nil {
			var once sync.Once
			return func() { once.Do(func() { syscall.Flock(fd, syscall.LOCK_UN); file.Close() }) }, nil
		}
		if e != syscall.EWOULDBLOCK && e != syscall.EAGAIN {
			file.Close()
			return nil, e
		}
		timer := time.NewTimer(20 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			file.Close()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
