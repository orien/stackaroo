/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package model

// NewTestContext creates a Context for testing purposes
func NewTestContext(name, region, account string) *Context {
	return &Context{
		Name:    name,
		Region:  region,
		Account: account,
	}
}

// NewDefaultTestContext creates a Context with default test values
func NewDefaultTestContext() *Context {
	return &Context{
		Name:    "test",
		Region:  "us-east-1",
		Account: "123456789012",
	}
}

// NewTestContextForRegion creates a Context for a specific region with default test values
func NewTestContextForRegion(region string) *Context {
	return &Context{
		Name:    "test",
		Region:  region,
		Account: "123456789012",
	}
}

// NewTestStack creates a Stack for testing purposes with proper Context
func NewTestStack(name string, context *Context) *Stack {
	if context == nil {
		context = NewDefaultTestContext()
	}

	return &Stack{
		Name:         name,
		Context:      context,
		TemplateBody: `{"AWSTemplateFormatVersion": "2010-09-09"}`,
		Parameters:   make(map[string]string),
		Tags:         make(map[string]string),
		Capabilities: []string{},
		Dependencies: []string{},
	}
}

// NewTestStackWithDefaults creates a Stack with default test values
func NewTestStackWithDefaults(name string) *Stack {
	return NewTestStack(name, NewDefaultTestContext())
}
