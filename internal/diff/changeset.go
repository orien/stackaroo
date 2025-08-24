/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"context"
	"fmt"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/orien/stackaroo/internal/aws"
)

// DefaultChangeSetManager implements ChangeSetManager using AWS CloudFormation
type DefaultChangeSetManager struct {
	cfClient aws.CloudFormationOperations
}

// NewChangeSetManager creates a new changeset manager
func NewChangeSetManager(cfClient aws.CloudFormationOperations) ChangeSetManager {
	return &DefaultChangeSetManager{
		cfClient: cfClient,
	}
}

// CreateChangeSet creates a CloudFormation changeset and returns the changes
func (m *DefaultChangeSetManager) CreateChangeSet(ctx context.Context, stackName string, template string, parameters map[string]string) (*ChangeSetInfo, error) {
	// Generate a unique changeset name
	changeSetName := fmt.Sprintf("stackaroo-diff-%d", time.Now().Unix())

	// Convert parameters to AWS format
	awsParameters := make([]types.Parameter, 0, len(parameters))
	for key, value := range parameters {
		awsParameters = append(awsParameters, types.Parameter{
			ParameterKey:   awssdk.String(key),
			ParameterValue: awssdk.String(value),
		})
	}

	// Create the changeset
	createInput := &cloudformation.CreateChangeSetInput{
		StackName:     awssdk.String(stackName),
		ChangeSetName: awssdk.String(changeSetName),
		TemplateBody:  awssdk.String(template),
		Parameters:    awsParameters,
		ChangeSetType: types.ChangeSetTypeUpdate, // Assume it's an update for existing stacks
	}

	createOutput, err := m.cfClient.CreateChangeSet(ctx, createInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create changeset: %w", err)
	}

	changeSetID := awssdk.ToString(createOutput.Id)

	// Wait for changeset to be created
	err = m.waitForChangeSet(ctx, changeSetID)
	if err != nil {
		// Clean up the changeset if it failed
		_ = m.DeleteChangeSet(ctx, changeSetID)
		return nil, fmt.Errorf("changeset creation failed: %w", err)
	}

	// Describe the changeset to get the actual changes
	changeSetInfo, err := m.describeChangeSet(ctx, changeSetID)
	if err != nil {
		// Clean up the changeset
		_ = m.DeleteChangeSet(ctx, changeSetID)
		return nil, fmt.Errorf("failed to describe changeset: %w", err)
	}

	// Clean up the changeset (we only needed it for preview)
	defer func() {
		if deleteErr := m.DeleteChangeSet(ctx, changeSetID); deleteErr != nil {
			// Log the error but don't fail the operation
			fmt.Printf("Warning: failed to delete changeset %s: %v\n", changeSetID, deleteErr)
		}
	}()

	return changeSetInfo, nil
}

// DeleteChangeSet deletes a CloudFormation changeset
func (m *DefaultChangeSetManager) DeleteChangeSet(ctx context.Context, changeSetID string) error {
	_, err := m.cfClient.DeleteChangeSet(ctx, &cloudformation.DeleteChangeSetInput{
		ChangeSetName: awssdk.String(changeSetID),
	})

	if err != nil {
		return fmt.Errorf("failed to delete changeset %s: %w", changeSetID, err)
	}

	return nil
}

// waitForChangeSet waits for a changeset to reach a terminal state
func (m *DefaultChangeSetManager) waitForChangeSet(ctx context.Context, changeSetID string) error {
	// Set a reasonable timeout for changeset creation
	timeout := 5 * time.Minute
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Describe the changeset to check its status
		describeOutput, err := m.cfClient.DescribeChangeSet(ctx, &cloudformation.DescribeChangeSetInput{
			ChangeSetName: awssdk.String(changeSetID),
		})

		if err != nil {
			return fmt.Errorf("failed to describe changeset while waiting: %w", err)
		}

		status := describeOutput.Status
		switch status {
		case types.ChangeSetStatusCreateComplete:
			return nil
		case types.ChangeSetStatusFailed:
			reason := awssdk.ToString(describeOutput.StatusReason)
			if reason == "" {
				reason = "unknown reason"
			}
			return fmt.Errorf("changeset creation failed: %s", reason)
		case types.ChangeSetStatusCreatePending, types.ChangeSetStatusCreateInProgress:
			// Still creating, wait a bit more
			time.Sleep(2 * time.Second)
			continue
		default:
			return fmt.Errorf("unexpected changeset status: %s", status)
		}
	}

	return fmt.Errorf("timeout waiting for changeset to be created")
}

// describeChangeSet gets the detailed information about a changeset
func (m *DefaultChangeSetManager) describeChangeSet(ctx context.Context, changeSetID string) (*ChangeSetInfo, error) {
	describeOutput, err := m.cfClient.DescribeChangeSet(ctx, &cloudformation.DescribeChangeSetInput{
		ChangeSetName: awssdk.String(changeSetID),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to describe changeset: %w", err)
	}

	// Convert AWS changeset to our format
	changeSetInfo := &ChangeSetInfo{
		ChangeSetID: changeSetID,
		Status:      string(describeOutput.Status),
		Changes:     make([]ResourceChange, 0, len(describeOutput.Changes)),
	}

	// Convert each change
	for _, awsChange := range describeOutput.Changes {
		if awsChange.ResourceChange != nil {
			resourceChange := ResourceChange{
				Action:       string(awsChange.ResourceChange.Action),
				ResourceType: awssdk.ToString(awsChange.ResourceChange.ResourceType),
				LogicalID:    awssdk.ToString(awsChange.ResourceChange.LogicalResourceId),
				PhysicalID:   awssdk.ToString(awsChange.ResourceChange.PhysicalResourceId),
				Replacement:  string(awsChange.ResourceChange.Replacement),
				Details:      make([]string, 0),
			}

			// Extract details from the change
			for _, detail := range awsChange.ResourceChange.Details {
				if detail.Target != nil {
					detailText := fmt.Sprintf("Property: %s", awssdk.ToString(detail.Target.Name))
					if detail.Target.Attribute != "" {
						detailText += fmt.Sprintf(" (%s)", detail.Target.Attribute)
					}
					resourceChange.Details = append(resourceChange.Details, detailText)
				}
			}

			changeSetInfo.Changes = append(changeSetInfo.Changes, resourceChange)
		}
	}

	return changeSetInfo, nil
}
