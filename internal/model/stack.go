/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package model

// ResolvedStack represents a fully resolved stack ready for deployment
type ResolvedStack struct {
	Name         string
	Environment  string
	TemplateBody string
	Parameters   map[string]string
	Tags         map[string]string
	Capabilities []string
	Dependencies []string
}

// GetTemplateContent returns the template content for this resolved stack
func (rs *ResolvedStack) GetTemplateContent() (string, error) {
	return rs.TemplateBody, nil
}

// ResolvedStacks represents a collection of resolved stacks
type ResolvedStacks struct {
	Context         string
	Stacks          []*ResolvedStack
	DeploymentOrder []string
}
