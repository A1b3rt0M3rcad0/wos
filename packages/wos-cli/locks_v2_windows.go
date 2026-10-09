package woscli

import (
	"context"
	"fmt"
	"golang.org/x/sys/windows"
	"runtime"
	"sync"
)

func acquireRuntimeLockV2(ctx context.Context, identity string) (func(), error) {
	// Agents running as a service and an interactive user can share a workspace
	// across Windows sessions. Failure to create a global mutex is fail-closed;
	// silently choosing a session-local namespace would weaken exclusion.
	name, e := windows.UTF16PtrFromString(`Global\WOS-v2-` + identity)
	if e != nil {
		return nil, e
	}
	// Windows mutex ownership is thread-bound. Keep acquisition and release on
	// the same locked thread; callers must release from the acquiring goroutine.
	runtime.LockOSThread()
	handle, e := windows.CreateMutex(nil, false, name)
	if e != nil {
		runtime.UnlockOSThread()
		return nil, e
	}
	for {
		if e = ctx.Err(); e != nil {
			windows.CloseHandle(handle)
			runtime.UnlockOSThread()
			return nil, e
		}
		status, e := windows.WaitForSingleObject(handle, 20)
		if e != nil {
			windows.CloseHandle(handle)
			runtime.UnlockOSThread()
			return nil, e
		}
		switch status {
		case windows.WAIT_OBJECT_0, windows.WAIT_ABANDONED:
			var once sync.Once
			return func() {
				once.Do(func() { windows.ReleaseMutex(handle); windows.CloseHandle(handle); runtime.UnlockOSThread() })
			}, nil
		case uint32(windows.WAIT_TIMEOUT):
		default:
			windows.CloseHandle(handle)
			runtime.UnlockOSThread()
			return nil, fmt.Errorf("runtime mutex unavailable")
		}
	}
}
