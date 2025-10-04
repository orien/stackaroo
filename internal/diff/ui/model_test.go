/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package ui

import (
	"testing"

	"github.com/orien/stackaroo/internal/diff"
	"github.com/stretchr/testify/assert"
)

// Test helper functions that don't require TUI instantiation

func TestMin(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"a less than b", 5, 10, 5},
		{"b less than a", 10, 5, 5},
		{"equal values", 7, 7, 7},
		{"negative numbers", -5, -10, -10},
		{"zero and positive", 0, 5, 0},
		{"zero and negative", 0, -5, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"a greater than b", 10, 5, 10},
		{"b greater than a", 5, 10, 10},
		{"equal values", 7, 7, 7},
		{"negative numbers", -5, -10, -5},
		{"zero and positive", 0, 5, 5},
		{"zero and negative", 0, -5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := max(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestViewMode_Constants(t *testing.T) {
	// Ensure view mode constants are defined
	assert.Equal(t, ViewMode(0), ViewOnly)
	assert.Equal(t, ViewMode(1), Confirmation)
}

func TestSection_Structure(t *testing.T) {
	// Test that Section struct can be created
	section := Section{
		Name:       "Test Section",
		Content:    "Test content",
		HasChanges: true,
		StartLine:  42,
	}

	assert.Equal(t, "Test Section", section.Name)
	assert.Equal(t, "Test content", section.Content)
	assert.True(t, section.HasChanges)
	assert.Equal(t, 42, section.StartLine)
}

// TestModel_ViewportIntegration verifies that viewport component is properly integrated
func TestModel_ViewportIntegration(t *testing.T) {
	t.Run("viewport is initialized with model", func(t *testing.T) {
		// Create a model with a minimal diff result
		result := &diff.Result{
			StackName:   "test-stack",
			Context:     "dev",
			StackExists: true,
		}

		m := NewModel(result, ViewOnly)

		// Verify viewport keys are initialized
		assert.NotEmpty(t, m.viewportKeys.Up.Keys(), "viewport should have up key bindings")
		assert.NotEmpty(t, m.viewportKeys.Down.Keys(), "viewport should have down key bindings")
		assert.NotEmpty(t, m.viewportKeys.PageUp.Keys(), "viewport should have page up key bindings")
		assert.NotEmpty(t, m.viewportKeys.PageDown.Keys(), "viewport should have page down key bindings")

		// Verify model is properly initialized
		assert.False(t, m.ready, "model should not be ready before window size")
		assert.Equal(t, ViewOnly, m.mode)
		assert.NotNil(t, m.help, "help component should be initialized")
	})
}

// TestModel_ScrollLogic removed - scrolling is now handled by the viewport component

func TestModel_SectionNavigation(t *testing.T) {
	t.Run("nextSection wraps around", func(t *testing.T) {
		m := Model{
			sections: []Section{
				{Name: "Section1"},
				{Name: "Section2"},
				{Name: "Section3"},
			},
			activeSection: 2,
		}
		m.nextSection()
		assert.Equal(t, 0, m.activeSection, "should wrap to first")
	})

	t.Run("nextSection with empty sections", func(t *testing.T) {
		m := Model{
			sections:      []Section{},
			activeSection: 0,
		}
		assert.NotPanics(t, func() {
			m.nextSection()
		})
	})

	t.Run("prevSection wraps around", func(t *testing.T) {
		m := Model{
			sections: []Section{
				{Name: "Section1"},
				{Name: "Section2"},
				{Name: "Section3"},
			},
			activeSection: 0,
		}
		m.prevSection()
		assert.Equal(t, 2, m.activeSection, "should wrap to last")
	})

	t.Run("prevSection with empty sections", func(t *testing.T) {
		m := Model{
			sections:      []Section{},
			activeSection: 0,
		}
		assert.NotPanics(t, func() {
			m.prevSection()
		})
	})
}

func TestModel_StateGetters(t *testing.T) {
	t.Run("Confirmed", func(t *testing.T) {
		m := Model{confirmed: false}
		assert.False(t, m.Confirmed())

		m.confirmed = true
		assert.True(t, m.Confirmed())
	})

	t.Run("Cancelled", func(t *testing.T) {
		m := Model{cancelled: false}
		assert.False(t, m.Cancelled())

		m.cancelled = true
		assert.True(t, m.Cancelled())
	})
}

func TestModel_HeightCalculations(t *testing.T) {
	t.Run("getHeaderHeight", func(t *testing.T) {
		m := Model{}
		height := m.getHeaderHeight()
		assert.Equal(t, 3, height)
	})

	t.Run("getFooterHeight without help", func(t *testing.T) {
		m := Model{showHelp: false}
		height := m.getFooterHeight()
		assert.Equal(t, 3, height)
	})

	t.Run("getFooterHeight with help", func(t *testing.T) {
		m := Model{showHelp: true}
		height := m.getFooterHeight()
		assert.Equal(t, 12, height)
	})
}

// Additional rendering tests removed - terminal detection in lipgloss can cause hangs in tests
// Viewport scrolling is now handled by the bubbles/viewport component
