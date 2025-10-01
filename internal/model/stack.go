/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package model

// Context holds context-specific information for stack operations
type Context struct {
	Name    string
	Region  string
	Account string
}

// Stack represents a fully resolved stack ready for deployment
type Stack struct {
	Name         string
	Context      *Context
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
