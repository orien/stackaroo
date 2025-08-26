/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// StackStatus represents the status of a CloudFormation stack
type StackStatus string

const (
	StackStatusCreateInProgress         StackStatus = "CREATE_IN_PROGRESS"
	StackStatusCreateComplete           StackStatus = "CREATE_COMPLETE"
	StackStatusCreateFailed             StackStatus = "CREATE_FAILED"
	StackStatusDeleteInProgress         StackStatus = "DELETE_IN_PROGRESS"
	StackStatusDeleteComplete           StackStatus = "DELETE_COMPLETE"
	StackStatusDeleteFailed             StackStatus = "DELETE_FAILED"
	StackStatusUpdateInProgress         StackStatus = "UPDATE_IN_PROGRESS"
	StackStatusUpdateComplete           StackStatus = "UPDATE_COMPLETE"
	StackStatusUpdateFailed             StackStatus = "UPDATE_FAILED"
	StackStatusUpdateRollbackInProgress StackStatus = "UPDATE_ROLLBACK_IN_PROGRESS"
	StackStatusUpdateRollbackComplete   StackStatus = "UPDATE_ROLLBACK_COMPLETE"
	StackStatusUpdateRollbackFailed     StackStatus = "UPDATE_ROLLBACK_FAILED"
	StackStatusRollbackInProgress       StackStatus = "ROLLBACK_IN_PROGRESS"
	StackStatusRollbackComplete         StackStatus = "ROLLBACK_COMPLETE"
	StackStatusRollbackFailed           StackStatus = "ROLLBACK_FAILED"
	StackStatusReviewInProgress         StackStatus = "REVIEW_IN_PROGRESS"
	StackStatusImportInProgress         StackStatus = "IMPORT_IN_PROGRESS"
	StackStatusImportComplete           StackStatus = "IMPORT_COMPLETE"
	StackStatusImportRollbackInProgress StackStatus = "IMPORT_ROLLBACK_IN_PROGRESS"
	StackStatusImportRollbackComplete   StackStatus = "IMPORT_ROLLBACK_COMPLETE"
	StackStatusImportRollbackFailed     StackStatus = "IMPORT_ROLLBACK_FAILED"
)

// Stack represents a CloudFormation stack with essential information
type Stack struct {
	Name        string
	Status      StackStatus
	CreatedTime *time.Time
	UpdatedTime *time.Time
	Description string
	Parameters  map[string]string
	Outputs     map[string]string
	Tags        map[string]string
}

// StackInfo represents detailed CloudFormation stack information for diff operations
type StackInfo struct {
	Name        string
	Status      StackStatus
	CreatedTime *time.Time
	UpdatedTime *time.Time
	Description string
	Parameters  map[string]string
	Outputs     map[string]string
	Tags        map[string]string
	Template    string // The actual template content
}

// Parameter represents a CloudFormation stack parameter
type Parameter struct {
	Key   string
	Value string
}

// DeployStackInput contains parameters for deploying a stack
type DeployStackInput struct {
	StackName    string
	TemplateBody string
	Parameters   []Parameter
	Tags         map[string]string
	Capabilities []string
}

// UpdateStackInput contains parameters for updating a stack
type UpdateStackInput struct {
	StackName    string
	TemplateBody string
	Parameters   []Parameter
	Tags         map[string]string
	Capabilities []string
}

// DeleteStackInput contains parameters for deleting a stack
type DeleteStackInput struct {
	StackName string
}

// StackEvent represents a CloudFormation stack event
type StackEvent struct {
	EventId              string
	StackName            string
	LogicalResourceId    string
	PhysicalResourceId   string
	ResourceType         string
	Timestamp            time.Time
	ResourceStatus       string
	ResourceStatusReason string
}

// NoChangesError indicates that a stack operation had no changes to apply
type NoChangesError struct {
	StackName string
}

func (e NoChangesError) Error() string {
	return fmt.Sprintf("stack %s is already up to date - no changes to deploy", e.StackName)
}

// DefaultCloudFormationOperations provides CloudFormation-specific operations
type DefaultCloudFormationOperations struct {
	client CloudFormationClient
}

// NewCloudFormationOperations creates a new CloudFormation operations wrapper
func (c *DefaultClient) NewCloudFormationOperations() CloudFormationOperations {
	return &DefaultCloudFormationOperations{
		client: c.cfn,
	}
}

// NewCloudFormationOperationsWithClient creates operations with a custom client (for testing)
func NewCloudFormationOperationsWithClient(client CloudFormationClient) *DefaultCloudFormationOperations {
	return &DefaultCloudFormationOperations{
		client: client,
	}
}

// DeployStack creates or updates a CloudFormation stack and waits for completion
func (cf *DefaultCloudFormationOperations) DeployStack(ctx context.Context, input DeployStackInput) error {
	return cf.DeployStackWithCallback(ctx, input, nil)
}

// DeployStackWithCallback creates or updates a CloudFormation stack and waits for completion,
// calling the provided callback for each event
func (cf *DefaultCloudFormationOperations) DeployStackWithCallback(ctx context.Context, input DeployStackInput, eventCallback func(StackEvent)) error {
	// Convert parameters to AWS format
	params := make([]types.Parameter, len(input.Parameters))
	for i, p := range input.Parameters {
		params[i] = types.Parameter{
			ParameterKey:   aws.String(p.Key),
			ParameterValue: aws.String(p.Value),
		}
	}

	tags := make([]types.Tag, 0, len(input.Tags))
	for k, v := range input.Tags {
		tags = append(tags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	capabilities := make([]types.Capability, len(input.Capabilities))
	for i, cap := range input.Capabilities {
		capabilities[i] = types.Capability(cap)
	}

	// Check if stack exists
	exists, err := cf.StackExists(ctx, input.StackName)
	if err != nil {
		return fmt.Errorf("failed to check if stack exists: %w", err)
	}

	var operationType string
	if exists {
		// Update existing stack
		operationType = "update"
		_, err = cf.client.UpdateStack(ctx, &cloudformation.UpdateStackInput{
			StackName:    aws.String(input.StackName),
			TemplateBody: aws.String(input.TemplateBody),
			Parameters:   params,
			Tags:         tags,
			Capabilities: capabilities,
		})

		if err != nil {
			// Check if it's a "no changes" error
			if isNoChangesError(err) {
				return NoChangesError{StackName: input.StackName}
			}
			return fmt.Errorf("failed to update stack %s: %w", input.StackName, err)
		}
	} else {
		// Create new stack
		operationType = "create"
		_, err = cf.client.CreateStack(ctx, &cloudformation.CreateStackInput{
			StackName:    aws.String(input.StackName),
			TemplateBody: aws.String(input.TemplateBody),
			Parameters:   params,
			Tags:         tags,
			Capabilities: capabilities,
		})

		if err != nil {
			return fmt.Errorf("failed to create stack %s: %w", input.StackName, err)
		}
	}

	// Wait for operation to complete
	err = cf.WaitForStackOperation(ctx, input.StackName, eventCallback)
	if err != nil {
		return fmt.Errorf("stack %s operation failed: %w", operationType, err)
	}

	return nil
}

// isNoChangesError checks if the error indicates no changes are needed
func isNoChangesError(err error) bool {
	if err == nil {
		return false
	}
	errorStr := err.Error()
	return contains(errorStr, "No updates are to be performed") ||
		contains(errorStr, "ValidationError")
}

// UpdateStack updates an existing CloudFormation stack
func (cf *DefaultCloudFormationOperations) UpdateStack(ctx context.Context, input UpdateStackInput) error {
	params := make([]types.Parameter, len(input.Parameters))
	for i, p := range input.Parameters {
		params[i] = types.Parameter{
			ParameterKey:   aws.String(p.Key),
			ParameterValue: aws.String(p.Value),
		}
	}

	tags := make([]types.Tag, 0, len(input.Tags))
	for k, v := range input.Tags {
		tags = append(tags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	capabilities := make([]types.Capability, len(input.Capabilities))
	for i, cap := range input.Capabilities {
		capabilities[i] = types.Capability(cap)
	}

	_, err := cf.client.UpdateStack(ctx, &cloudformation.UpdateStackInput{
		StackName:    aws.String(input.StackName),
		TemplateBody: aws.String(input.TemplateBody),
		Parameters:   params,
		Tags:         tags,
		Capabilities: capabilities,
	})

	if err != nil {
		return fmt.Errorf("failed to update stack %s: %w", input.StackName, err)
	}

	return nil
}

// DeleteStack deletes a CloudFormation stack
func (cf *DefaultCloudFormationOperations) DeleteStack(ctx context.Context, input DeleteStackInput) error {
	_, err := cf.client.DeleteStack(ctx, &cloudformation.DeleteStackInput{
		StackName: aws.String(input.StackName),
	})

	if err != nil {
		return fmt.Errorf("failed to delete stack %s: %w", input.StackName, err)
	}

	return nil
}

// GetStack retrieves information about a specific stack
func (cf *DefaultCloudFormationOperations) GetStack(ctx context.Context, stackName string) (*Stack, error) {
	result, err := cf.client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to describe stack %s: %w", stackName, err)
	}

	if len(result.Stacks) == 0 {
		return nil, fmt.Errorf("stack %s not found", stackName)
	}

	cfnStack := result.Stacks[0]
	stack := &Stack{
		Name:        aws.ToString(cfnStack.StackName),
		Status:      StackStatus(cfnStack.StackStatus),
		CreatedTime: cfnStack.CreationTime,
		UpdatedTime: cfnStack.LastUpdatedTime,
		Description: aws.ToString(cfnStack.Description),
		Parameters:  make(map[string]string),
		Outputs:     make(map[string]string),
		Tags:        make(map[string]string),
	}

	// Convert parameters
	for _, param := range cfnStack.Parameters {
		stack.Parameters[aws.ToString(param.ParameterKey)] = aws.ToString(param.ParameterValue)
	}

	// Convert outputs
	for _, output := range cfnStack.Outputs {
		stack.Outputs[aws.ToString(output.OutputKey)] = aws.ToString(output.OutputValue)
	}

	// Convert tags
	for _, tag := range cfnStack.Tags {
		stack.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}

	return stack, nil
}

// ListStacks returns a list of all CloudFormation stacks
func (cf *DefaultCloudFormationOperations) ListStacks(ctx context.Context) ([]*Stack, error) {
	var stacks []*Stack
	paginator := cloudformation.NewListStacksPaginator(cf.client, &cloudformation.ListStacksInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list stacks: %w", err)
		}

		for _, summary := range page.StackSummaries {
			// Skip deleted stacks
			if summary.StackStatus == types.StackStatusDeleteComplete {
				continue
			}

			stack := &Stack{
				Name:        aws.ToString(summary.StackName),
				Status:      StackStatus(summary.StackStatus),
				CreatedTime: summary.CreationTime,
				UpdatedTime: summary.LastUpdatedTime,
				Description: aws.ToString(summary.TemplateDescription),
			}
			stacks = append(stacks, stack)
		}
	}

	return stacks, nil
}

// ValidateTemplate validates a CloudFormation template
func (cf *DefaultCloudFormationOperations) ValidateTemplate(ctx context.Context, templateBody string) error {
	_, err := cf.client.ValidateTemplate(ctx, &cloudformation.ValidateTemplateInput{
		TemplateBody: aws.String(templateBody),
	})

	if err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	return nil
}

// StackExists checks if a stack exists
func (cf *DefaultCloudFormationOperations) StackExists(ctx context.Context, stackName string) (bool, error) {
	_, err := cf.client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})

	if err != nil {
		// Check if it's a "does not exist" error
		if isStackNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if stack exists: %w", err)
	}

	return true, nil
}

// isStackNotFoundError checks if the error indicates the stack doesn't exist
func isStackNotFoundError(err error) bool {
	// This is a simplified check - in practice you might want to check the specific AWS error codes
	return err != nil && (contains(err.Error(), "does not exist") ||
		contains(err.Error(), "ValidationError"))
}

// GetTemplate retrieves the template for a CloudFormation stack
func (cf *DefaultCloudFormationOperations) GetTemplate(ctx context.Context, stackName string) (string, error) {
	result, err := cf.client.GetTemplate(ctx, &cloudformation.GetTemplateInput{
		StackName: aws.String(stackName),
	})

	if err != nil {
		return "", fmt.Errorf("failed to get template for stack %s: %w", stackName, err)
	}

	return aws.ToString(result.TemplateBody), nil
}

// DescribeStack retrieves detailed information about a specific stack including template
func (cf *DefaultCloudFormationOperations) DescribeStack(ctx context.Context, stackName string) (*StackInfo, error) {
	// Get basic stack information
	stack, err := cf.GetStack(ctx, stackName)
	if err != nil {
		return nil, err
	}

	// Get template content
	template, err := cf.GetTemplate(ctx, stackName)
	if err != nil {
		return nil, fmt.Errorf("failed to get template for stack %s: %w", stackName, err)
	}

	// Convert Stack to StackInfo
	stackInfo := &StackInfo{
		Name:        stack.Name,
		Status:      stack.Status,
		CreatedTime: stack.CreatedTime,
		UpdatedTime: stack.UpdatedTime,
		Description: stack.Description,
		Parameters:  stack.Parameters,
		Outputs:     stack.Outputs,
		Tags:        stack.Tags,
		Template:    template,
	}

	return stackInfo, nil
}

// CreateChangeSet creates a CloudFormation changeset
func (cf *DefaultCloudFormationOperations) CreateChangeSet(ctx context.Context, params *cloudformation.CreateChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateChangeSetOutput, error) {
	return cf.client.CreateChangeSet(ctx, params, optFns...)
}

// DeleteChangeSet deletes a CloudFormation changeset
func (cf *DefaultCloudFormationOperations) DeleteChangeSet(ctx context.Context, params *cloudformation.DeleteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteChangeSetOutput, error) {
	return cf.client.DeleteChangeSet(ctx, params, optFns...)
}

// DescribeChangeSet describes a CloudFormation changeset
func (cf *DefaultCloudFormationOperations) DescribeChangeSet(ctx context.Context, params *cloudformation.DescribeChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeChangeSetOutput, error) {
	return cf.client.DescribeChangeSet(ctx, params, optFns...)
}

// DescribeStackEvents retrieves events for a CloudFormation stack
func (cf *DefaultCloudFormationOperations) DescribeStackEvents(ctx context.Context, stackName string) ([]StackEvent, error) {
	var events []StackEvent
	paginator := cloudformation.NewDescribeStackEventsPaginator(cf.client, &cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stackName),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe stack events for %s: %w", stackName, err)
		}

		for _, event := range page.StackEvents {
			stackEvent := StackEvent{
				EventId:              aws.ToString(event.EventId),
				StackName:            aws.ToString(event.StackName),
				LogicalResourceId:    aws.ToString(event.LogicalResourceId),
				PhysicalResourceId:   aws.ToString(event.PhysicalResourceId),
				ResourceType:         aws.ToString(event.ResourceType),
				Timestamp:            aws.ToTime(event.Timestamp),
				ResourceStatus:       string(event.ResourceStatus),
				ResourceStatusReason: aws.ToString(event.ResourceStatusReason),
			}
			events = append(events, stackEvent)
		}
	}

	return events, nil
}

// WaitForStackOperation waits for a CloudFormation stack operation to complete,
// calling the provided callback for each new event
func (cf *DefaultCloudFormationOperations) WaitForStackOperation(ctx context.Context, stackName string, eventCallback func(StackEvent)) error {
	const pollInterval = 5 * time.Second
	seenEvents := make(map[string]bool)

	for {
		// Check stack status
		stack, err := cf.GetStack(ctx, stackName)
		if err != nil {
			return fmt.Errorf("failed to get stack status: %w", err)
		}

		// Get latest events
		events, err := cf.DescribeStackEvents(ctx, stackName)
		if err != nil {
			return fmt.Errorf("failed to get stack events: %w", err)
		}

		// Process new events (events are returned in reverse chronological order)
		for i := len(events) - 1; i >= 0; i-- {
			event := events[i]
			if !seenEvents[event.EventId] {
				seenEvents[event.EventId] = true
				if eventCallback != nil {
					eventCallback(event)
				}
			}
		}

		// Check if operation is complete
		if isStackOperationComplete(stack.Status) {
			if isStackOperationSuccessful(stack.Status) {
				return nil
			}
			return fmt.Errorf("stack operation failed with status: %s", stack.Status)
		}

		// Wait before next poll
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
			continue
		}
	}
}

// isStackOperationComplete checks if a stack operation has completed
func isStackOperationComplete(status StackStatus) bool {
	switch status {
	case StackStatusCreateComplete,
		StackStatusCreateFailed,
		StackStatusUpdateComplete,
		StackStatusUpdateFailed,
		StackStatusUpdateRollbackComplete,
		StackStatusUpdateRollbackFailed,
		StackStatusDeleteComplete,
		StackStatusDeleteFailed,
		StackStatusRollbackComplete,
		StackStatusRollbackFailed,
		StackStatusImportComplete,
		StackStatusImportRollbackComplete,
		StackStatusImportRollbackFailed:
		return true
	default:
		return false
	}
}

// isStackOperationSuccessful checks if a completed stack operation was successful
func isStackOperationSuccessful(status StackStatus) bool {
	switch status {
	case StackStatusCreateComplete,
		StackStatusUpdateComplete,
		StackStatusDeleteComplete,
		StackStatusImportComplete:
		return true
	default:
		return false
	}
}

// contains is a simple string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || s[len(s)-len(substr):] == substr || s[:len(substr)] == substr || indexString(s, substr) >= 0)
}

// indexString returns the index of substr in s, or -1 if not present
func indexString(s, substr string) int {
	n := len(substr)
	if n == 0 {
		return 0
	}
	for i := 0; i <= len(s)-n; i++ {
		if s[i:i+n] == substr {
			return i
		}
	}
	return -1
}
