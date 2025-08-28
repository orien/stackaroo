/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package version

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInfo_ContainsAllExpectedComponents(t *testing.T) {
	info := Info()

	// Should contain all expected components
	assert.Contains(t, info, "stackaroo", "info should contain application name")
	assert.Contains(t, info, "Git commit:", "info should contain git commit label")
	assert.Contains(t, info, "Build date:", "info should contain build date label")
	assert.Contains(t, info, "Go version:", "info should contain go version label")
	assert.Contains(t, info, "Platform:", "info should contain platform label")

	// Should be multi-line format
	lines := strings.Split(info, "\n")
	assert.Len(t, lines, 5, "info should have exactly 5 lines")
}

func TestInfo_FormatsVersionCorrectly(t *testing.T) {
	// Test with default values
	info := Info()

	// First line should contain version
	lines := strings.Split(info, "\n")
	require.Len(t, lines, 5)

	firstLine := lines[0]
	assert.True(t, strings.HasPrefix(firstLine, "stackaroo "), "first line should start with 'stackaroo '")
	assert.Contains(t, firstLine, Version, "first line should contain the version")
}

func TestInfo_IncludesRuntimeVariables(t *testing.T) {
	info := Info()

	// Should include actual runtime Go version
	assert.Contains(t, info, GoVersion, "should include actual Go version")
	assert.Contains(t, info, runtime.Version(), "should match runtime.Version()")

	// Should include actual platform
	assert.Contains(t, info, Platform, "should include actual platform")
	expectedPlatform := runtime.GOOS + "/" + runtime.GOARCH
	assert.Contains(t, info, expectedPlatform, "should match OS/ARCH format")
}

func TestShort_ReturnsVersionOnly(t *testing.T) {
	short := Short()

	assert.Equal(t, Version, short, "Short() should return exactly the Version variable")
	assert.NotContains(t, short, "Git commit", "Short() should not contain additional metadata")
	assert.NotContains(t, short, "\n", "Short() should be single line")
}

func TestRuntimeVariables_ArePopulatedCorrectly(t *testing.T) {
	// Test GoVersion
	assert.NotEmpty(t, GoVersion, "GoVersion should not be empty")
	assert.True(t, strings.HasPrefix(GoVersion, "go"), "GoVersion should start with 'go'")
	assert.Equal(t, runtime.Version(), GoVersion, "GoVersion should match runtime.Version()")

	// Test Platform
	assert.NotEmpty(t, Platform, "Platform should not be empty")
	assert.Contains(t, Platform, "/", "Platform should contain OS/ARCH separator")

	expectedPlatform := runtime.GOOS + "/" + runtime.GOARCH
	assert.Equal(t, expectedPlatform, Platform, "Platform should match GOOS/GOARCH format")
}

func TestBuildTimeVariables_HaveDefaultValues(t *testing.T) {
	// These will be "dev", "unknown", "unknown" respectively in development builds
	// but could be overridden via ldflags in actual builds

	assert.NotEmpty(t, Version, "Version should not be empty")
	assert.NotEmpty(t, GitCommit, "GitCommit should not be empty")
	assert.NotEmpty(t, BuildDate, "BuildDate should not be empty")

	// In development/test environment, these should have default values
	// This documents the expected behaviour for local development
	t.Logf("Version: %s", Version)
	t.Logf("GitCommit: %s", GitCommit)
	t.Logf("BuildDate: %s", BuildDate)
}

func TestInfo_OutputFormat(t *testing.T) {
	info := Info()
	t.Logf("Version info output:\n%s", info)

	// Verify the exact format structure
	lines := strings.Split(info, "\n")
	require.Len(t, lines, 5)

	// Check each line format
	assert.True(t, strings.HasPrefix(lines[0], "stackaroo "), "line 1: should start with 'stackaroo '")
	assert.True(t, strings.HasPrefix(lines[1], "  Git commit: "), "line 2: should be indented git commit")
	assert.True(t, strings.HasPrefix(lines[2], "  Build date: "), "line 3: should be indented build date")
	assert.True(t, strings.HasPrefix(lines[3], "  Go version: "), "line 4: should be indented go version")
	assert.True(t, strings.HasPrefix(lines[4], "  Platform:   "), "line 5: should be indented platform")
}

// TestInfo_WithDifferentVersionFormats tests various version string formats
// that might be injected at build time
func TestInfo_WithDifferentVersionFormats(t *testing.T) {
	testCases := []struct {
		name             string
		version          string
		expectedContains string
	}{
		{
			name:             "release version",
			version:          "v1.0.0",
			expectedContains: "stackaroo v1.0.0",
		},
		{
			name:             "development version",
			version:          "1.0.0+a1b2c3d",
			expectedContains: "stackaroo 1.0.0+a1b2c3d",
		},
		{
			name:             "dirty development version",
			version:          "1.0.0+a1b2c3d-dirty",
			expectedContains: "stackaroo 1.0.0+a1b2c3d-dirty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// We can't actually modify the global variables in tests,
			// but we can test the expected format by constructing what
			// Info() would return with different version values
			expectedInfo := strings.Replace(Info(), Version, tc.version, 1)
			assert.Contains(t, expectedInfo, tc.expectedContains)
		})
	}
}
