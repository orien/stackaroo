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

// StackDiffer implements the Differ interface using AWS CloudFormation
type StackDiffer struct {
	clientFactory       aws.ClientFactory
	templateComparator  TemplateComparator
	parameterComparator ParameterComparator
	tagComparator       TagComparator
}

// NewStackDiffer creates a new StackDiffer with provided client factory
func NewStackDiffer(clientFactory aws.ClientFactory) *StackDiffer {
	return &StackDiffer{
		clientFactory:       clientFactory,
		templateComparator:  NewYAMLTemplateComparator(),
		parameterComparator: NewParameterComparator(),
		tagComparator:       NewTagComparator(),
	}
}

// DiffStack compares a resolved stack configuration with the deployed stack
func (d *StackDiffer) DiffStack(ctx context.Context, stack *model.Stack, options Options) (*Result, error) {
	// Get region-specific CloudFormation operations
	cfClient, err := d.clientFactory.GetCloudFormationOperations(ctx, stack.Context.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to get CloudFormation operations for region %s: %w", stack.Context.Region, err)
	}

	result := &Result{
		StackName: stack.Name,
		Context:   stack.Context.Name,
		Options:   options,
	}

	// Check if stack exists in AWS
	exists, err := cfClient.StackExists(ctx, stack.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check if stack exists: %w", err)
	}

	result.StackExists = exists

	// If stack doesn't exist, this is a new stack scenario
	if !exists {
		return d.handleNewStack(ctx, stack, result)
	}

	// Get current stack state from AWS
	currentStack, err := cfClient.DescribeStack(ctx, stack.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to describe stack: %w", err)
	}

	// Compare templates (if not filtered out)
	if !options.ParametersOnly && !options.TagsOnly {
		templateChange, err := d.compareTemplates(ctx, stack, currentStack, cfClient)
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
		changeSetInfo, err := d.generateChangeSet(ctx, stack, options, cfClient)
		if err != nil {
			// Don't fail the entire diff if changeset generation fails
			// Store the error in the result for display in formatted output
			result.ChangeSetError = err
		} else {
			result.ChangeSet = changeSetInfo
		}
	}

	return result, nil
}

// handleNewStack handles the case where the stack doesn't exist yet
func (d *StackDiffer) handleNewStack(ctx context.Context, stack *model.Stack, result *Result) (*Result, error) {
	// For a new stack, everything is "added"

	// Get the proposed template content
	proposedTemplate, err := stack.GetTemplateContent()
	if err != nil {
		return nil, fmt.Errorf("failed to get template content: %w", err)
	}

	// Compare empty template with proposed template to show unified diff
	emptyTemplate := "{}" // Empty JSON object represents no existing stack
	templateChange, err := d.templateComparator.Compare(ctx, emptyTemplate, proposedTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to compare templates: %w", err)
	}

	result.TemplateChange = templateChange

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
func (d *StackDiffer) compareTemplates(ctx context.Context, stack *model.Stack, currentStack *aws.StackInfo, cfClient aws.CloudFormationOperations) (*TemplateChange, error) {
	// Get current template from AWS
	currentTemplate, err := cfClient.GetTemplate(ctx, stack.Name)
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
func (d *StackDiffer) compareParameters(currentStack *aws.StackInfo, stack *model.Stack) ([]ParameterDiff, error) {
	return d.parameterComparator.Compare(currentStack.Parameters, stack.Parameters)
}

// compareTags compares current stack tags with resolved tags
func (d *StackDiffer) compareTags(currentStack *aws.StackInfo, stack *model.Stack) ([]TagDiff, error) {
	return d.tagComparator.Compare(currentStack.Tags, stack.Tags)
}

// generateChangeSet creates an AWS changeset to preview changes
func (d *StackDiffer) generateChangeSet(ctx context.Context, stack *model.Stack, options Options, cfClient aws.CloudFormationOperations) (*aws.ChangeSetInfo, error) {
	// Get proposed template content
	templateContent, err := stack.GetTemplateContent()
	if err != nil {
		return nil, fmt.Errorf("failed to get template content: %w", err)
	}

	// Create changeset - use deployment version if we need to keep it alive
	var changeSetInfo *aws.ChangeSetInfo

	// Get capabilities from stack configuration
	capabilities := stack.Capabilities
	if len(capabilities) == 0 {
		capabilities = []string{"CAPABILITY_IAM"} // Default capability
	}

	if options.KeepChangeSet {
		// Use deployment-style changeset that doesn't auto-delete
		changeSetInfo, err = cfClient.CreateChangeSetForDeployment(
			ctx,
			stack.Name,
			templateContent,
			stack.Parameters,
			capabilities,
			stack.Tags,
		)
	} else {
		// Use standard changeset that auto-deletes for preview only
		changeSetInfo, err = cfClient.CreateChangeSetPreview(ctx, stack.Name, templateContent, stack.Parameters, capabilities, stack.Tags)
	}

	if err != nil {
		return nil, err
	}

	return changeSetInfo, nil
}
