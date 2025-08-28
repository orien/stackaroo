/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package version

import (
	"fmt"
	"runtime"
)

// Build-time variables (populated via -ldflags during build)
var (
	// Version is the semantic version of stackaroo (e.g., "v1.0.0" or "1.0.0+a1b2c3d")
	Version = "dev"

	// GitCommit is the short git commit hash (e.g., "a1b2c3d")
	GitCommit = "unknown"

	// BuildDate is when the binary was built (e.g., "2025-01-27 14:30:45 UTC")
	BuildDate = "unknown"
)

// Runtime variables (determined at runtime)
var (
	// GoVersion is the Go compiler version used to build the binary
	GoVersion = runtime.Version()

	// Platform is the operating system and architecture (e.g., "linux/amd64")
	Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

// Info returns formatted version information for display to users
func Info() string {
	return fmt.Sprintf(`stackaroo %s
  Git commit: %s
  Build date: %s
  Go version: %s
  Platform:   %s`, Version, GitCommit, BuildDate, GoVersion, Platform)
}

// Short returns just the version string without additional metadata
func Short() string {
	return Version
}
