// Package version provides version information for the application.
package version

import (
	"runtime/debug"
)

// Version is the current version of the application.
// This is intended to be set at build time using ldflags:
// -ldflags "-X noctl/internal/version.Version=v1.0.0"
var Version = "dev"

// Get returns the version of the application.
// It prioritizes the Version variable set via ldflags,
// and falls back to build info if available.
func Get() string {
	if Version != "dev" && Version != "" {
		return Version
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}

	return Version
}
