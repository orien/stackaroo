/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Prompter defines the interface for user prompting
type Prompter interface {
	ConfirmDeployment(stackName string) (bool, error)
}

// StdinPrompter implements Prompter using standard input
type StdinPrompter struct {
	input io.Reader
}

// NewStdinPrompter creates a new prompter that reads from stdin
func NewStdinPrompter() *StdinPrompter {
	return &StdinPrompter{input: os.Stdin}
}

// ConfirmDeployment prompts the user via stdin to confirm deployment changes
func (p *StdinPrompter) ConfirmDeployment(stackName string) (bool, error) {
	fmt.Printf("\nDo you want to apply these changes to stack %s? [y/N]: ", stackName)

	scanner := bufio.NewScanner(p.input)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, fmt.Errorf("failed to read user input: %w", err)
		}
		// EOF or empty input - treat as "no"
		return false, nil
	}

	response := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return response == "y" || response == "yes", nil
}

// defaultPrompter is the package-level default prompter
var defaultPrompter Prompter = NewStdinPrompter()

// SetPrompter allows injection of a custom prompter (for testing)
func SetPrompter(p Prompter) {
	defaultPrompter = p
}

// GetDefaultPrompter returns the current default prompter (for testing)
func GetDefaultPrompter() Prompter {
	return defaultPrompter
}

// ConfirmDeployment prompts the user to confirm deployment changes using the default prompter
// Returns true if the user confirms (y/yes), false otherwise
func ConfirmDeployment(stackName string) (bool, error) {
	return defaultPrompter.ConfirmDeployment(stackName)
}
