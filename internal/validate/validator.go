/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package validate

import (
	"context"
	"fmt"

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
		fmt.Printf("\n✗ Validation failed for stack '%s'\n", stackName)
		fmt.Printf("  Error: %v\n", err)
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
			results = append(results, ValidationResult{
				StackName: stackName,
				Valid:     false,
				Error:     fmt.Sprintf("failed to resolve stack: %v", err),
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
		return fmt.Errorf("template validation failed: %w", err)
	}

	return nil
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
			fmt.Printf("  Error: %s\n", result.Error)
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
	Error     string
}
