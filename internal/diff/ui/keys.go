/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package ui

import "github.com/charmbracelet/bubbles/v2/key"

// keyMap defines the keyboard shortcuts for the diff viewer
// Note: Viewport scrolling keys (up/down/pgup/pgdn/etc) are handled
// directly by the viewport component
type keyMap struct {
	// Section navigation
	NextSection key.Binding
	PrevSection key.Binding

	// Actions
	Quit    key.Binding
	Help    key.Binding
	Confirm key.Binding
	Cancel  key.Binding

	confirmMode bool
}

// defaultKeyMap returns the default keyboard shortcuts
func defaultKeyMap(confirmMode bool) keyMap {
	km := keyMap{
		confirmMode: confirmMode,
		NextSection: key.NewBinding(
			key.WithKeys("tab", "right"),
			key.WithHelp("tab/→", "next section"),
		),
		PrevSection: key.NewBinding(
			key.WithKeys("shift+tab", "left"),
			key.WithHelp("shift+tab/←", "previous section"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
	}

	// Mode-specific keybindings
	if confirmMode {
		km.Quit = key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q/esc", "cancel"),
		)
		km.Confirm = key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "confirm & deploy"),
		)
		km.Cancel = key.NewBinding(
			key.WithKeys("n", "enter"),
			key.WithHelp("n/enter", "cancel"),
		)
	} else {
		km.Quit = key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q/esc", "quit"),
		)
	}

	return km
}

// ShortHelp returns keybindings to be shown in the short help view
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.NextSection,
		k.Help,
		k.Quit,
	}
}

// FullHelp returns keybindings to be shown in the full help view
// Note: Viewport scrolling keys are shown by the viewport's own help
func (k keyMap) FullHelp() [][]key.Binding {
	if k.confirmMode {
		return [][]key.Binding{
			{k.NextSection, k.PrevSection},
			{k.Confirm, k.Cancel, k.Quit, k.Help},
		}
	}

	return [][]key.Binding{
		{k.NextSection, k.PrevSection},
		{k.Quit, k.Help},
	}
}
