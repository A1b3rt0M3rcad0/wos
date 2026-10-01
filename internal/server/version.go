package server

import "fmt"

// Build metadata is intended to be overridden with -ldflags during release
// builds. Development binaries remain explicit instead of inventing a version.
var (
	Version = "dev"
	Commit  = "unknown"
	BuiltAt = "unknown"
)

type VersionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	BuiltAt string `json:"built_at"`
}

func CurrentVersion() VersionInfo {
	return VersionInfo{Version: Version, Commit: Commit, BuiltAt: BuiltAt}
}

func (v VersionInfo) String() string {
	return fmt.Sprintf("wos %s (commit %s, built %s)", v.Version, v.Commit, v.BuiltAt)
}
