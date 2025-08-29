/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"context"
	"fmt"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/model"
)

// DefaultDiffer implements the Differ interface using AWS CloudFormation
type DefaultDiffer struct {
	cfClient            aws.CloudFormationOperations
	templateComparator  TemplateComparator
	parameterComparator ParameterComparator
	tagComparator       TagComparator
}

// NewDiffer creates a new DefaultDiffer with provided CloudFormation operations
func NewDiffer(cfClient aws.CloudFormationOperations) *DefaultDiffer {
	return &DefaultDiffer{
		cfClient:            cfClient,
		templateComparator:  NewYAMLTemplateComparator(),
		parameterComparator: NewParameterComparator(),
		tagComparator:       NewTagComparator(),
	}
}

// DiffStack compares a resolved stack configuration with the deployed stack
func (d *DefaultDiffer) DiffStack(ctx context.Context, stack *model.Stack, options Options) (*Result, error) {
	result := &Result{
		StackName: stack.Name,
		Context:   stack.Context,
		Options:   options,
	}

	// Check if stack exists in AWS
	stackExists, err := d.cfClient.StackExists(ctx, stack.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check if stack exists: %w", err)
	}

	result.StackExists = stackExists

	// If stack doesn't exist, this is a new stack scenario
	if !stackExists {
		return d.handleNewStack(ctx, stack, result)
	}

	// Get current stack state from AWS
	currentStack, err := d.cfClient.DescribeStack(ctx, stack.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to describe stack: %w", err)
	}

	// Compare templates (if not filtered out)
	if !options.ParametersOnly && !options.TagsOnly {
		templateChange, err := d.compareTemplates(ctx, stack, currentStack)
		if err != nil {
			return nil, fmt.Errorf("failed to compare templates: %w", err)
		}
		result.TemplateChange = templateChange
	}

	// Compare parameters (if not filtered out)
	if !options.TemplateOnly && !options.TagsOnly {
		parameterDiffs, err := d.compareParameters(currentStack, stack)
		if err != nil {
			return nil, fmt.Errorf("failed to compare parameters: %w", err)
		}
		result.ParameterDiffs = parameterDiffs
	}

	// Compare tags (if not filtered out)
	if !options.TemplateOnly && !options.ParametersOnly {
		tagDiffs, err := d.compareTags(currentStack, stack)
		if err != nil {
			return nil, fmt.Errorf("failed to compare tags: %w", err)
		}
		result.TagDiffs = tagDiffs
	}

	// Generate changeset if there are potential changes and we're doing a full diff
	if result.HasChanges() && !options.TemplateOnly && !options.ParametersOnly && !options.TagsOnly {
		changeSetInfo, err := d.generateChangeSet(ctx, stack, options)
		if err != nil {
			// Don't fail the entire diff if changeset generation fails
			// Just log and continue without changeset info
			fmt.Printf("Warning: failed to generate changeset: %v\n", err)
		} else {
			result.ChangeSet = changeSetInfo
		}
	}

	return result, nil
}

// handleNewStack handles the case where the stack doesn't exist yet
func (d *DefaultDiffer) handleNewStack(ctx context.Context, stack *model.Stack, result *Result) (*Result, error) {
	// For a new stack, everything is "added"

	// Template is new
	result.TemplateChange = &TemplateChange{
		HasChanges:   true,
		CurrentHash:  "",
		ProposedHash: "", // We'll calculate this when we implement the template comparator
		Diff:         "New stack - entire template will be created",
	}

	// All parameters are new
	for key, value := range stack.Parameters {
		result.ParameterDiffs = append(result.ParameterDiffs, ParameterDiff{
			Key:           key,
			CurrentValue:  "",
			ProposedValue: value,
			ChangeType:    ChangeTypeAdd,
		})
	}

	// All tags are new
	for key, value := range stack.Tags {
		result.TagDiffs = append(result.TagDiffs, TagDiff{
			Key:           key,
			CurrentValue:  "",
			ProposedValue: value,
			ChangeType:    ChangeTypeAdd,
		})
	}

	return result, nil
}

// compareTemplates compares the current deployed template with the resolved template
func (d *DefaultDiffer) compareTemplates(ctx context.Context, stack *model.Stack, currentStack *aws.StackInfo) (*TemplateChange, error) {
	// Get current template from AWS
	currentTemplate, err := d.cfClient.GetTemplate(ctx, stack.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get current template: %w", err)
	}

	// Get proposed template content
	proposedTemplate, err := stack.GetTemplateContent()
	if err != nil {
		return nil, fmt.Errorf("failed to get proposed template content: %w", err)
	}

	// Compare templates
	templateChange, err := d.templateComparator.Compare(ctx, currentTemplate, proposedTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to compare templates: %w", err)
	}

	return templateChange, nil
}

// compareParameters compares current stack parameters with resolved parameters
func (d *DefaultDiffer) compareParameters(currentStack *aws.StackInfo, stack *model.Stack) ([]ParameterDiff, error) {
	return d.parameterComparator.Compare(currentStack.Parameters, stack.Parameters)
}

// compareTags compares current stack tags with resolved tags
func (d *DefaultDiffer) compareTags(currentStack *aws.StackInfo, stack *model.Stack) ([]TagDiff, error) {
	return d.tagComparator.Compare(currentStack.Tags, stack.Tags)
}

// generateChangeSet creates an AWS changeset to preview changes
func (d *DefaultDiffer) generateChangeSet(ctx context.Context, stack *model.Stack, options Options) (*aws.ChangeSetInfo, error) {
	// Get proposed template content
	templateContent, err := stack.GetTemplateContent()
	if err != nil {
		return nil, fmt.Errorf("failed to get template content: %w", err)
	}

	// Create changeset - use deployment version if we need to keep it alive
	var changeSetInfo *aws.ChangeSetInfo
	if options.KeepChangeSet {
		// Use deployment-style changeset that doesn't auto-delete
		capabilities := stack.Capabilities
		if len(capabilities) == 0 {
			capabilities = []string{"CAPABILITY_IAM"} // Default capability
		}

		changeSetInfo, err = d.cfClient.CreateChangeSetForDeployment(
			ctx,
			stack.Name,
			templateContent,
			stack.Parameters,
			capabilities,
			stack.Tags,
		)
	} else {
		// Use standard changeset that auto-deletes for preview only
		changeSetInfo, err = d.cfClient.CreateChangeSetPreview(ctx, stack.Name, templateContent, stack.Parameters)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create changeset: %w", err)
	}

	return changeSetInfo, nil
}
