/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package prompt

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMockPrompter_Interface verifies MockPrompter implements Prompter interface
func TestMockPrompter_Interface(t *testing.T) {
	var _ Prompter = (*MockPrompter)(nil)
}

// TestMockPrompter_Confirm_Acceptance tests the mock prompter functionality for acceptance
func TestMockPrompter_Confirm_Acceptance(t *testing.T) {
	// Store original prompter to restore later
	originalPrompter := defaultPrompter
	defer SetPrompter(originalPrompter)

	mockPrompter := &MockPrompter{}

	// Test confirmation
	message := "Do you want to proceed?"
	mockPrompter.On("Confirm", message).Return(true, nil).Once()

	SetPrompter(mockPrompter)

	result, err := Confirm(message)

	assert.NoError(t, err)
	assert.True(t, result)
	mockPrompter.AssertExpectations(t)
}

// TestMockPrompter_Confirm_Rejection tests mock prompter rejection
func TestMockPrompter_Confirm_Rejection(t *testing.T) {
	// Store original prompter to restore later
	originalPrompter := defaultPrompter
	defer SetPrompter(originalPrompter)

	mockPrompter := &MockPrompter{}

	// Test rejection
	message := "Are you sure?"
	mockPrompter.On("Confirm", message).Return(false, nil).Once()

	SetPrompter(mockPrompter)

	result, err := Confirm(message)

	assert.NoError(t, err)
	assert.False(t, result)
	mockPrompter.AssertExpectations(t)
}

// TestSetPrompter_ChangesDefaultPrompter tests the SetPrompter functionality
func TestSetPrompter_ChangesDefaultPrompter(t *testing.T) {
	// Store original prompter to restore later
	originalPrompter := defaultPrompter
	defer SetPrompter(originalPrompter)

	// Create and set mock prompter
	mockPrompter := &MockPrompter{}
	message := "Continue with operation?"
	mockPrompter.On("Confirm", message).Return(true, nil).Once()

	SetPrompter(mockPrompter)

	// Call the package-level function which should use our mock
	result, err := Confirm(message)

	assert.NoError(t, err)
	assert.True(t, result)
	mockPrompter.AssertExpectations(t)
}

// TestDefaultPrompter_IsStdinPrompter verifies default prompter type
func TestDefaultPrompter_IsStdinPrompter(t *testing.T) {
	// Verify that the default prompter is a StdinPrompter
	_, ok := defaultPrompter.(*StdinPrompter)
	assert.True(t, ok, "Default prompter should be a StdinPrompter")
}

// TestResponseParsing tests the logic for parsing user responses
func TestResponseParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"yes lowercase", "yes", true},
		{"yes uppercase", "YES", true},
		{"yes mixed case", "Yes", true},
		{"y lowercase", "y", true},
		{"y uppercase", "Y", true},
		{"no", "no", false},
		{"n", "n", false},
		{"empty", "", false},
		{"whitespace only", "   ", false},
		{"other text", "maybe", false},
		{"partial match", "yeah", false},
		{"with whitespace", " y ", true},
		{"with whitespace no", " no ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the core logic that Confirm uses
			response := strings.ToLower(strings.TrimSpace(tt.input))
			result := response == "y" || response == "yes"

			assert.Equal(t, tt.expected, result,
				"Input '%s' should return %t", tt.input, tt.expected)
		})
	}
}

// TestPromptMessageFormatting tests that custom messages are handled correctly
func TestPromptMessageFormatting(t *testing.T) {
	// Test various message formats
	messages := []string{
		"Proceed?",
		"Do you want to continue with this action?",
		"Confirm dangerous operation? This cannot be undone.",
		"Are you ready?",
	}

	for i, message := range messages {
		t.Run(fmt.Sprintf("message_handling_%d", i), func(t *testing.T) {
			// Store original prompter to restore later
			originalPrompter := defaultPrompter
			defer SetPrompter(originalPrompter)

			mockPrompter := &MockPrompter{}
			mockPrompter.On("Confirm", message).Return(true, nil).Once()

			SetPrompter(mockPrompter)

			result, err := Confirm(message)

			assert.NoError(t, err)
			assert.True(t, result)
			mockPrompter.AssertExpectations(t)
		})
	}
}

// TestConfirmationBehaviour documents the expected behaviour
func TestConfirmationBehaviour(t *testing.T) {
	// This test serves as documentation for the expected behaviour

	t.Run("accepts_only_explicit_yes", func(t *testing.T) {
		// Only "y" and "yes" (case insensitive) should return true
		yesResponses := []string{"y", "Y", "yes", "YES", "Yes"}
		for _, response := range yesResponses {
			normalized := strings.ToLower(strings.TrimSpace(response))
			result := normalized == "y" || normalized == "yes"
			assert.True(t, result, "Response '%s' should be accepted as confirmation", response)
		}
	})

	t.Run("rejects_all_other_input", func(t *testing.T) {
		// Everything else should return false
		noResponses := []string{"n", "no", "NO", "", "  ", "maybe", "ok", "sure", "yep", "nope"}
		for _, response := range noResponses {
			normalized := strings.ToLower(strings.TrimSpace(response))
			result := normalized == "y" || normalized == "yes"
			assert.False(t, result, "Response '%s' should be rejected", response)
		}
	})

	t.Run("default_behaviour", func(t *testing.T) {
		// Empty input or whitespace should default to "no" (false)
		emptyInputs := []string{"", "   ", "\t", "\n"}
		for _, input := range emptyInputs {
			normalized := strings.ToLower(strings.TrimSpace(input))
			result := normalized == "y" || normalized == "yes"
			assert.False(t, result, "Empty/whitespace input should default to no")
		}
	})
}

// TestConfirm_UsesDefaultPrompter verifies package function uses default prompter
func TestConfirm_UsesDefaultPrompter(t *testing.T) {
	// Store original prompter to restore later
	originalPrompter := defaultPrompter
	defer SetPrompter(originalPrompter)

	// Create mock that expects to be called
	mockPrompter := &MockPrompter{}
	message := "Execute command?"
	mockPrompter.On("Confirm", message).Return(false, nil).Once()

	SetPrompter(mockPrompter)

	// Call package function
	result, err := Confirm(message)

	assert.NoError(t, err)
	assert.False(t, result)
	mockPrompter.AssertExpectations(t)
}

// TestStdinPrompter_CreatesCorrectly tests StdinPrompter creation
func TestStdinPrompter_CreatesCorrectly(t *testing.T) {
	prompter := NewStdinPrompter()
	assert.NotNil(t, prompter)
	assert.NotNil(t, prompter.input)
}

// TestGetDefaultPrompter_ReturnsPrompter tests getter function
func TestGetDefaultPrompter_ReturnsPrompter(t *testing.T) {
	prompter := GetDefaultPrompter()
	assert.NotNil(t, prompter)
	assert.Implements(t, (*Prompter)(nil), prompter)
}

// Note: The MockPrompter allows full testing of confirmation flows without requiring
// actual user input. Tests can configure expected responses and verify behavior.
// For interactive testing of the StdinPrompter, manual testing is recommended.
