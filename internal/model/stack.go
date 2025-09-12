/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package model

// Stack represents a fully resolved stack ready for deployment
type Stack struct {
	Name         string
	Context      string
	TemplateBody string
	Parameters   map[string]string
	Tags         map[string]string
	Capabilities []string
	Dependencies []string
}

// GetTemplateContent returns the template content for this stack
func (rs *Stack) GetTemplateContent() (string, error) {
	return rs.TemplateBody, nil
}
