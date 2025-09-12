/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package resolve

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// TemplateProcessor defines the interface for processing CloudFormation templates with templating
type TemplateProcessor interface {
	Process(templateContent string, variables map[string]interface{}) (string, error)
}

// CfnTemplateProcessor implements TemplateProcessor using Go's text/template with Sprig functions
type CfnTemplateProcessor struct{}

// NewCfnTemplateProcessor creates a new CloudFormation template processor
func NewCfnTemplateProcessor() *CfnTemplateProcessor {
	return &CfnTemplateProcessor{}
}

// Process processes a CloudFormation template with the provided variables using Go templates and Sprig functions
func (tp *CfnTemplateProcessor) Process(templateContent string, variables map[string]interface{}) (string, error) {
	// Create template with Sprig function map
	tmpl, err := template.New("cloudformation").
		Funcs(sprig.TxtFuncMap()).
		Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute template with variables
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, variables)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
