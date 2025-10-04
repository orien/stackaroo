/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package ui

// KeyMap defines the keyboard shortcuts for the diff viewer
type KeyMap struct {
	confirmMode bool
}

// DefaultKeyMap returns the default keyboard shortcuts
func DefaultKeyMap(confirmMode bool) KeyMap {
	return KeyMap{
		confirmMode: confirmMode,
	}
}

// KeyBindings returns a map of key descriptions for display
func (k KeyMap) KeyBindings() map[string]string {
	bindings := map[string]string{
		// Navigation
		"↑/k":          "scroll up",
		"↓/j":          "scroll down",
		"pgup/b":       "page up",
		"pgdn/f/space": "page down",
		"u/ctrl+u":     "half page up",
		"d/ctrl+d":     "half page down",
		"g/home":       "go to top",
		"G/end":        "go to bottom",

		// Section navigation
		"tab/→/l":       "next section",
		"shift+tab/←/h": "previous section",

		// General
		"?": "toggle help",
	}

	if k.confirmMode {
		bindings["enter/y"] = "confirm & deploy"
		bindings["n"] = "cancel"
		bindings["q/esc"] = "cancel"
	} else {
		bindings["q/esc"] = "quit"
	}

	return bindings
}
