/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import "fmt"

// Hyperlink wraps text with terminal hyperlink escape codes (OSC 8).
// This allows terminals that support it to display clickable links.
//
// The OSC 8 format is: ESC ]8;;URL ESC \ TEXT ESC ]8;; ESC \
// Where ESC is the escape character (\033).
//
// Supported terminals include:
//   - iTerm2 (3.1+)
//   - Terminal.app (macOS 10.15+)
//   - GNOME Terminal (3.26+)
//   - Konsole
//   - VS Code integrated terminal
//   - Windows Terminal
//
// Terminals without hyperlink support will display the text normally
// without any visual artifacts.
//
// Parameters:
//   - url: The target URL for the hyperlink
//   - text: The visible text to display
//
// Returns the text wrapped with hyperlink escape codes, or just the text
// if either url or text is empty.
//
// Example:
//
//	link := Hyperlink("https://example.com", "Example")
//	fmt.Println(link) // Displays "Example" as a clickable link
func Hyperlink(url, text string) string {
	if url == "" || text == "" {
		return text
	}

	// OSC 8 format: \033]8;;URL\033\\TEXT\033]8;;\033\\
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}
