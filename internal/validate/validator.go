/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package validate

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/config"
	"github.com/orien/stackaroo/internal/model"
	"github.com/orien/stackaroo/internal/resolve"
)

// Validator orchestrates template validation
type Validator interface {
	ValidateSingleStack(ctx context.Context, stackName, contextName string) error
	ValidateAllStacks(ctx context.Context, contextName string) error
}

// TemplateValidator implements the Validator interface
type TemplateValidator struct {
	clientFactory  aws.ClientFactory
	configProvider config.ConfigProvider
	resolver       resolve.Resolver
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
	}
}

// ValidateSingleStack validates a single stack's template
func (v *TemplateValidator) ValidateSingleStack(ctx context.Context, stackName, contextName string) error {
	fmt.Printf("Validating template for stack '%s' in context '%s'...\n", stackName, contextName)

	// Resolve the stack (handles template loading and processing)
	stack, err := v.resolver.ResolveStack(ctx, contextName, stackName)
	if err != nil {
		return fmt.Errorf("failed to resolve stack %s: %w", stackName, err)
	}

	// Validate the template
	if err := v.validateStack(ctx, stack); err != nil {
		v.printValidationError(stackName, err)
		return err
	}

	fmt.Printf("\n✓ Template is valid for stack '%s'\n", stackName)
	return nil
}

// ValidateAllStacks validates all stacks in a context
func (v *TemplateValidator) ValidateAllStacks(ctx context.Context, contextName string) error {
	// Get list of all stacks in the context
	stackNames, err := v.configProvider.ListStacks(contextName)
	if err != nil {
		return fmt.Errorf("failed to list stacks for context %s: %w", contextName, err)
	}

	if len(stackNames) == 0 {
		fmt.Printf("No stacks defined in context '%s'\n", contextName)
		return nil
	}

	fmt.Printf("Validating %d stack(s) in context '%s'...\n\n", len(stackNames), contextName)

	results := make([]ValidationResult, 0, len(stackNames))
	hasErrors := false

	// Validate each stack
	for _, stackName := range stackNames {
		fmt.Printf("→ Validating '%s'... ", stackName)

		// Resolve the stack
		stack, err := v.resolver.ResolveStack(ctx, contextName, stackName)
		if err != nil {
			fmt.Printf("✗\n")
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
			fmt.Printf("✗\n")
			results = append(results, ValidationResult{
				StackName: stackName,
				Valid:     false,
				Error:     err.Error(),
				ErrorObj:  err,
			})
			hasErrors = true
		} else {
			fmt.Printf("✓\n")
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
	fmt.Printf("\n✗ Validation failed for stack '%s'\n", stackName)
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Template Validation Issues")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	issues := parseValidationError(err)
	if len(issues) > 0 {
		for i, issue := range issues {
			fmt.Printf("\n%d. %s\n", i+1, issue.Title)
			if issue.Detail != "" {
				fmt.Printf("   %s\n", issue.Detail)
			}
		}
	} else {
		// Fallback to raw error if we can't parse it
		fmt.Printf("\n%s\n", err.Error())
	}

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
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
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Validation Summary")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	validCount := 0
	invalidCount := 0

	for _, result := range results {
		if result.Valid {
			validCount++
			fmt.Printf("✓ %s\n", result.StackName)
		} else {
			invalidCount++
			fmt.Printf("✗ %s\n", result.StackName)

			// Parse and display validation issues
			issues := parseValidationError(result.ErrorObj)
			for _, issue := range issues {
				fmt.Printf("  • %s", issue.Title)
				if issue.Detail != "" {
					fmt.Printf(": %s", issue.Detail)
				}
				fmt.Println()
			}
		}
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Total:   %d\n", len(results))
	fmt.Printf("Valid:   %d\n", validCount)
	fmt.Printf("Invalid: %d\n", invalidCount)

	if invalidCount == 0 {
		fmt.Println("\n✓ All templates are valid")
	} else {
		fmt.Println("\n✗ Some templates failed validation")
	}
}

// ValidationResult contains the outcome of a single stack validation
type ValidationResult struct {
	StackName string
	Valid     bool
	Error     string // Deprecated: Use ErrorObj for better error parsing
	ErrorObj  error  // The actual error object for better parsing
}
