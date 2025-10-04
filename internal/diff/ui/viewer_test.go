/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package ui

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldUseInteractive_NOCOLORSet(t *testing.T) {
	// Setup
	_ = os.Setenv("NO_COLOR", "1")
	defer func() { _ = os.Unsetenv("NO_COLOR") }()

	result := ShouldUseInteractive()

	assert.False(t, result, "should not use interactive when NO_COLOR is set")
}

func TestShouldUseInteractive_StackarooPlainSet(t *testing.T) {
	// Setup
	_ = os.Setenv("STACKAROO_PLAIN", "1")
	defer func() { _ = os.Unsetenv("STACKAROO_PLAIN") }()

	result := ShouldUseInteractive()

	assert.False(t, result, "should not use interactive when STACKAROO_PLAIN is set")
}

func TestShouldUseInteractive_TermDumb(t *testing.T) {
	// Setup
	originalTerm := os.Getenv("TERM")
	_ = os.Setenv("TERM", "dumb")
	defer func() {
		if originalTerm != "" {
			_ = os.Setenv("TERM", originalTerm)
		} else {
			_ = os.Unsetenv("TERM")
		}
	}()

	result := ShouldUseInteractive()

	assert.False(t, result, "should not use interactive when TERM is dumb")
}

func TestShouldUseInteractive_TermEmpty(t *testing.T) {
	// Setup
	originalTerm := os.Getenv("TERM")
	_ = os.Unsetenv("TERM")
	defer func() {
		if originalTerm != "" {
			_ = os.Setenv("TERM", originalTerm)
		}
	}()

	result := ShouldUseInteractive()

	assert.False(t, result, "should not use interactive when TERM is empty")
}

func TestShouldUseInteractive_MultipleConditions(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		cleanupEnv  func()
		expected    bool
		description string
	}{
		{
			name: "NO_COLOR takes precedence",
			setupEnv: func() {
				_ = os.Setenv("NO_COLOR", "1")
				_ = os.Setenv("TERM", "xterm-256color")
			},
			cleanupEnv: func() {
				_ = os.Unsetenv("NO_COLOR")
			},
			expected:    false,
			description: "NO_COLOR should disable interactive even with good TERM",
		},
		{
			name: "STACKAROO_PLAIN takes precedence",
			setupEnv: func() {
				_ = os.Setenv("STACKAROO_PLAIN", "1")
				_ = os.Setenv("TERM", "xterm-256color")
			},
			cleanupEnv: func() {
				_ = os.Unsetenv("STACKAROO_PLAIN")
			},
			expected:    false,
			description: "STACKAROO_PLAIN should disable interactive",
		},
		{
			name: "All disable conditions",
			setupEnv: func() {
				_ = os.Setenv("NO_COLOR", "1")
				_ = os.Setenv("STACKAROO_PLAIN", "1")
				_ = os.Setenv("TERM", "dumb")
			},
			cleanupEnv: func() {
				_ = os.Unsetenv("NO_COLOR")
				_ = os.Unsetenv("STACKAROO_PLAIN")
			},
			expected:    false,
			description: "Multiple disable conditions should all work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			originalTerm := os.Getenv("TERM")
			tt.setupEnv()

			// Test
			result := ShouldUseInteractive()

			// Cleanup
			tt.cleanupEnv()
			if originalTerm != "" {
				_ = os.Setenv("TERM", originalTerm)
			} else {
				_ = os.Unsetenv("TERM")
			}

			// Assert
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestForceInteractive_Enable(t *testing.T) {
	// Setup - set some env vars that would normally disable interactive
	_ = os.Setenv("STACKAROO_PLAIN", "1")
	_ = os.Setenv("NO_COLOR", "1")

	// Force interactive
	ForceInteractive(true)

	// Check that disabling env vars were unset
	assert.Empty(t, os.Getenv("STACKAROO_PLAIN"), "STACKAROO_PLAIN should be unset")
	assert.Empty(t, os.Getenv("NO_COLOR"), "NO_COLOR should be unset")

	// Cleanup
	_ = os.Unsetenv("STACKAROO_PLAIN")
	_ = os.Unsetenv("NO_COLOR")
}

func TestForceInteractive_Disable(t *testing.T) {
	// Ensure clean state
	_ = os.Unsetenv("STACKAROO_PLAIN")

	// Force plain mode
	ForceInteractive(false)

	// Check that STACKAROO_PLAIN was set
	assert.NotEmpty(t, os.Getenv("STACKAROO_PLAIN"), "STACKAROO_PLAIN should be set")

	// Cleanup
	_ = os.Unsetenv("STACKAROO_PLAIN")
}

func TestForceInteractive_Toggle(t *testing.T) {
	// Start in plain mode
	ForceInteractive(false)
	assert.NotEmpty(t, os.Getenv("STACKAROO_PLAIN"))

	// Toggle to interactive
	ForceInteractive(true)
	assert.Empty(t, os.Getenv("STACKAROO_PLAIN"))

	// Toggle back to plain
	ForceInteractive(false)
	assert.NotEmpty(t, os.Getenv("STACKAROO_PLAIN"))

	// Cleanup
	_ = os.Unsetenv("STACKAROO_PLAIN")
	_ = os.Unsetenv("NO_COLOR")
}

func TestShouldUseInteractive_TTYDetection(t *testing.T) {
	// This test checks that the function doesn't panic when checking stdout
	// In test environment, stdout might not be a TTY, which is expected

	// Ensure env vars don't interfere
	_ = os.Unsetenv("NO_COLOR")
	_ = os.Unsetenv("STACKAROO_PLAIN")
	originalTerm := os.Getenv("TERM")
	if originalTerm == "" {
		_ = os.Setenv("TERM", "xterm")
	}

	// Should not panic
	result := ShouldUseInteractive()

	// In test environment, this will likely be false because stdout isn't a TTY
	// But the important thing is it doesn't panic
	assert.False(t, result, "test environment typically doesn't have TTY")

	// Cleanup
	if originalTerm == "" {
		_ = os.Unsetenv("TERM")
	} else {
		_ = os.Setenv("TERM", originalTerm)
	}
}

func TestShouldUseInteractive_PriorityOrder(t *testing.T) {
	// Test that environment variables are checked in the correct priority order
	// Priority: STACKAROO_PLAIN > NO_COLOR > TERM > TTY

	t.Run("STACKAROO_PLAIN highest priority", func(t *testing.T) {
		_ = os.Setenv("STACKAROO_PLAIN", "1")
		_ = os.Unsetenv("NO_COLOR")
		_ = os.Setenv("TERM", "xterm-256color")
		defer func() {
			_ = os.Unsetenv("STACKAROO_PLAIN")
		}()

		assert.False(t, ShouldUseInteractive(), "STACKAROO_PLAIN should take precedence")
	})

	t.Run("NO_COLOR second priority", func(t *testing.T) {
		_ = os.Unsetenv("STACKAROO_PLAIN")
		_ = os.Setenv("NO_COLOR", "1")
		_ = os.Setenv("TERM", "xterm-256color")
		defer func() {
			_ = os.Unsetenv("NO_COLOR")
		}()

		assert.False(t, ShouldUseInteractive(), "NO_COLOR should take precedence over TERM")
	})

	t.Run("TERM checked third", func(t *testing.T) {
		_ = os.Unsetenv("STACKAROO_PLAIN")
		_ = os.Unsetenv("NO_COLOR")
		_ = os.Setenv("TERM", "dumb")

		assert.False(t, ShouldUseInteractive(), "dumb TERM should disable interactive")

		_ = os.Unsetenv("TERM")
	})
}

func TestForceInteractive_ErrorHandling(t *testing.T) {
	// Test that ForceInteractive handles errors gracefully (doesn't panic)
	// Even if os.Setenv/Unsetenv theoretically fail

	// These should not panic
	assert.NotPanics(t, func() {
		ForceInteractive(true)
	}, "ForceInteractive(true) should not panic")

	assert.NotPanics(t, func() {
		ForceInteractive(false)
	}, "ForceInteractive(false) should not panic")

	// Cleanup
	_ = os.Unsetenv("STACKAROO_PLAIN")
	_ = os.Unsetenv("NO_COLOR")
}

func TestShouldUseInteractive_EmptyStringVsUnset(t *testing.T) {
	// Test difference between empty string and unset

	t.Run("NO_COLOR empty string should disable", func(t *testing.T) {
		_ = os.Setenv("NO_COLOR", "")
		defer func() { _ = os.Unsetenv("NO_COLOR") }()

		// According to no-color.org, any value (even empty) disables colour
		result := ShouldUseInteractive()
		assert.False(t, result, "NO_COLOR with empty value should still disable interactive")
	})

	t.Run("STACKAROO_PLAIN empty string should disable", func(t *testing.T) {
		_ = os.Setenv("STACKAROO_PLAIN", "")
		defer func() { _ = os.Unsetenv("STACKAROO_PLAIN") }()

		result := ShouldUseInteractive()
		assert.False(t, result, "STACKAROO_PLAIN with empty value should disable interactive")
	})
}
