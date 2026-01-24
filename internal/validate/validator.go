/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package validate

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
	"codeberg.org/orien/stackaroo/internal/aws"
	"codeberg.org/orien/stackaroo/internal/config"
	"codeberg.org/orien/stackaroo/internal/model"
	"codeberg.org/orien/stackaroo/internal/resolve"
)

// Validator orchestrates template validation
type Validator interface {
	ValidateSingleStack(ctx context.Context, stackName, contextName string) error
	ValidateAllStacks(ctx context.Context, contextName string) error
}

// ValidationStyles contains styles for validation output
type ValidationStyles struct {
	Success lipgloss.Style
	Error   lipgloss.Style
	Warning lipgloss.Style
	Title   lipgloss.Style
	Detail  lipgloss.Style
	Subtle  lipgloss.Style
}

// NewValidationStyles creates styles for validation output
func NewValidationStyles() *ValidationStyles {
	// Detect terminal background
	hasDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)

	var successColor, errorColor, warningColor, titleColor, subtleColor string

	if hasDark {
		// Dark background colors
		successColor = "10" // Green
		errorColor = "9"    // Red
		warningColor = "11" // Yellow
		titleColor = "12"   // Blue
		subtleColor = "8"   // Dark Grey
	} else {
		// Light background colors
		successColor = "2" // Green
		errorColor = "1"   // Red
		warningColor = "3" // Yellow/Brown
		titleColor = "4"   // Blue
		subtleColor = "8"  // Grey
	}

	return &ValidationStyles{
		Success: lipgloss.NewStyle().Foreground(lipgloss.Color(successColor)).Bold(true),
		Error:   lipgloss.NewStyle().Foreground(lipgloss.Color(errorColor)).Bold(true),
		Warning: lipgloss.NewStyle().Foreground(lipgloss.Color(warningColor)),
		Title:   lipgloss.NewStyle().Foreground(lipgloss.Color(titleColor)).Bold(true),
		Detail:  lipgloss.NewStyle(),
		Subtle:  lipgloss.NewStyle().Foreground(lipgloss.Color(subtleColor)),
	}
}

// TemplateValidator implements the Validator interface
type TemplateValidator struct {
	clientFactory  aws.ClientFactory
	configProvider config.ConfigProvider
	resolver       resolve.Resolver
	styles         *ValidationStyles
}

// NewTemplateValidator creates a new validator
func NewTemplateValidator(
	clientFactory aws.ClientFactory,
	configProvider config.ConfigProvider,
	resolver resolve.Resolver,
) *TemplateValidator {
	return &TemplateValidator{
		clientFactory:  clientFactory,
		configProvider: configProvider,
		resolver:       resolver,
		styles:         NewValidationStyles(),
	}
}

// ValidateSingleStack validates a single stack's template
func (v *TemplateValidator) ValidateSingleStack(ctx context.Context, stackName, contextName string) error {
	fmt.Printf("Validating template for stack '%s' in context '%s'...\n", stackName, contextName)

	// Resolve the stack (handles template loading and processing)
	stack, err := v.resolver.ResolveStack(ctx, contextName, stackName)
	if err != nil {
		return err
	}

	// Validate the template
	if err := v.validateStack(ctx, stack); err != nil {
		v.printValidationError(stackName, err)
		return err
	}

	fmt.Printf("\n%s Template is valid for stack '%s'\n", v.styles.Success.Render("✓"), stackName)
	return nil
}

// ValidateAllStacks validates all stacks in a context
func (v *TemplateValidator) ValidateAllStacks(ctx context.Context, contextName string) error {
	// Get list of all stacks in the context
	stackNames, err := v.configProvider.ListStacks(contextName)
	if err != nil {
		return err
	}

	if len(stackNames) == 0 {
		fmt.Printf("No stacks defined in context '%s'\n", contextName)
		return nil
	}

	fmt.Printf("Validating %s stack(s) in context '%s'...\n\n", v.styles.Title.Render(fmt.Sprintf("%d", len(stackNames))), contextName)

	results := make([]ValidationResult, 0, len(stackNames))
	hasErrors := false

	// Validate each stack
	for _, stackName := range stackNames {
		fmt.Printf("→ Validating '%s'... ", stackName)

		// Resolve the stack
		stack, err := v.resolver.ResolveStack(ctx, contextName, stackName)
		if err != nil {
			fmt.Printf("%s\n", v.styles.Error.Render("✗"))
			resolveErr := fmt.Errorf("failed to resolve stack: %w", err)
			results = append(results, ValidationResult{
				StackName: stackName,
				Valid:     false,
				Error:     resolveErr.Error(),
				ErrorObj:  resolveErr,
			})
			hasErrors = true
			continue
		}

		// Validate the template
		if err := v.validateStack(ctx, stack); err != nil {
			fmt.Printf("%s\n", v.styles.Error.Render("✗"))
			results = append(results, ValidationResult{
				StackName: stackName,
				Valid:     false,
				Error:     err.Error(),
				ErrorObj:  err,
			})
			hasErrors = true
		} else {
			fmt.Printf("%s\n", v.styles.Success.Render("✓"))
			results = append(results, ValidationResult{
				StackName: stackName,
				Valid:     true,
			})
		}
	}

	// Print summary
	v.printSummary(results)

	if hasErrors {
		return fmt.Errorf("validation failed for one or more stacks")
	}

	return nil
}

// validateStack validates a resolved stack's template using AWS CloudFormation API
func (v *TemplateValidator) validateStack(ctx context.Context, stack *model.Stack) error {
	// Get CloudFormation operations for the stack's region
	cfnOps, err := v.clientFactory.GetCloudFormationOperations(ctx, stack.Context.Region)
	if err != nil {
		return fmt.Errorf("failed to get CloudFormation operations: %w", err)
	}

	// Validate with AWS
	if err := cfnOps.ValidateTemplate(ctx, stack.TemplateBody); err != nil {
		return err
	}

	return nil
}

// printValidationError formats and prints a user-friendly validation error report
func (v *TemplateValidator) printValidationError(stackName string, err error) {
	fmt.Printf("\n%s Validation failed for stack '%s'\n\n", v.styles.Error.Render("✗"), stackName)

	issues := parseValidationError(err)
	if len(issues) > 0 {
		for _, issue := range issues {
			fmt.Printf("  %s\n", v.styles.Warning.Render(issue.Title))
			if issue.Detail != "" {
				fmt.Printf("    %s\n", v.styles.Detail.Render(issue.Detail))
			}
		}
	} else {
		// Fallback to raw error if we can't parse it
		fmt.Printf("  %s\n", v.styles.Error.Render(err.Error()))
	}
	fmt.Println()
}

// ValidationIssue represents a parsed validation issue
type ValidationIssue struct {
	Title  string
	Detail string
}

// parseValidationError extracts structured validation issues from AWS errors
func parseValidationError(err error) []ValidationIssue {
	if err == nil {
		return nil
	}

	errMsg := err.Error()
	var issues []ValidationIssue

	// Extract the actual ValidationError message from the AWS SDK error
	// Pattern: "api error ValidationError: <actual message>"
	validationErrorPattern := regexp.MustCompile(`api error ValidationError: (.+?)(?:\n|$)`)
	if matches := validationErrorPattern.FindStringSubmatch(errMsg); len(matches) > 1 {
		errMsg = matches[1]
	}

	// Pattern 1: Unrecognized resource types
	unrecognizedPattern := regexp.MustCompile(`Unrecognized resource types?: \[(.+?)\]`)
	if matches := unrecognizedPattern.FindStringSubmatch(errMsg); len(matches) > 1 {
		resourceTypes := strings.Split(matches[1], ",")
		for _, rt := range resourceTypes {
			rt = strings.TrimSpace(rt)
			issues = append(issues, ValidationIssue{
				Title:  "Invalid Resource Type",
				Detail: fmt.Sprintf("Resource type '%s' is not recognized by CloudFormation", rt),
			})
		}
	}

	// Pattern 2: Invalid parameter type
	invalidParamPattern := regexp.MustCompile(`Invalid value for parameter type: (.+)`)
	if matches := invalidParamPattern.FindStringSubmatch(errMsg); len(matches) > 1 {
		issues = append(issues, ValidationIssue{
			Title:  "Invalid Parameter Type",
			Detail: fmt.Sprintf("Parameter type '%s' is not valid", strings.TrimSpace(matches[1])),
		})
	}

	// Pattern 3: Undefined resource references
	undefinedResourcePattern := regexp.MustCompile(`references undefined resource (.+?)(?:\s|$|,|\.)`)
	if matches := undefinedResourcePattern.FindStringSubmatch(errMsg); len(matches) > 1 {
		issues = append(issues, ValidationIssue{
			Title:  "Undefined Resource Reference",
			Detail: fmt.Sprintf("Template references resource '%s' which is not defined", strings.TrimSpace(matches[1])),
		})
	}

	// Pattern 4: JSON/YAML syntax errors
	if strings.Contains(errMsg, "not well-formed") || strings.Contains(errMsg, "JSON") {
		issues = append(issues, ValidationIssue{
			Title:  "Template Syntax Error",
			Detail: "Template is not well-formed JSON or YAML",
		})
	}

	// Pattern 5: Missing required properties
	missingPropPattern := regexp.MustCompile(`[Mm]issing required property: (.+)`)
	if matches := missingPropPattern.FindStringSubmatch(errMsg); len(matches) > 1 {
		issues = append(issues, ValidationIssue{
			Title:  "Missing Required Property",
			Detail: fmt.Sprintf("Required property '%s' is missing", strings.TrimSpace(matches[1])),
		})
	}

	// Pattern 6: Generic template format error (fallback)
	if len(issues) == 0 && strings.Contains(errMsg, "Template format error") {
		// Extract the message after "Template format error:"
		parts := strings.SplitN(errMsg, "Template format error:", 2)
		if len(parts) > 1 {
			issues = append(issues, ValidationIssue{
				Title:  "Template Format Error",
				Detail: strings.TrimSpace(parts[1]),
			})
		}
	}

	// If no patterns matched, return the cleaned error message
	if len(issues) == 0 {
		// Remove redundant prefixes
		cleanMsg := strings.TrimPrefix(errMsg, "template validation failed: ")
		cleanMsg = strings.TrimPrefix(cleanMsg, "operation error CloudFormation: ValidateTemplate, ")
		cleanMsg = strings.TrimSpace(cleanMsg)

		issues = append(issues, ValidationIssue{
			Title:  "Validation Error",
			Detail: cleanMsg,
		})
	}

	return issues
}

// printSummary prints validation results summary
func (v *TemplateValidator) printSummary(results []ValidationResult) {
	fmt.Printf("\n%s\n", v.styles.Title.Render("VALIDATION SUMMARY"))
	fmt.Println()

	validCount := 0
	invalidCount := 0

	for _, result := range results {
		if result.Valid {
			validCount++
			fmt.Printf("  %s %s\n", v.styles.Success.Render("✓"), result.StackName)
		} else {
			invalidCount++
			fmt.Printf("  %s %s\n", v.styles.Error.Render("✗"), result.StackName)

			// Parse and display validation issues
			issues := parseValidationError(result.ErrorObj)
			for _, issue := range issues {
				fmt.Printf("      %s", v.styles.Warning.Render(issue.Title))
				if issue.Detail != "" {
					fmt.Printf(": %s", v.styles.Detail.Render(issue.Detail))
				}
				fmt.Println()
			}
		}
	}

	fmt.Println()

	// Build summary line
	var summaryParts []string

	if invalidCount > 0 {
		summaryParts = append(summaryParts, v.styles.Error.Render(fmt.Sprintf("%d failed", invalidCount)))
	}
	summaryParts = append(summaryParts, v.styles.Success.Render(fmt.Sprintf("%d passed", validCount)))
	summaryParts = append(summaryParts, fmt.Sprintf("%d total", len(results)))

	fmt.Println(strings.Join(summaryParts, ", "))
}

// ValidationResult contains the outcome of a single stack validation
type ValidationResult struct {
	StackName string
	Valid     bool
	Error     string // Deprecated: Use ErrorObj for better error parsing
	ErrorObj  error  // The actual error object for better parsing
}
