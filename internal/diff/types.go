/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"context"

	"github.com/orien/stackaroo/internal/model"
)

// Differ defines the interface for performing stack diffs
type Differ interface {
	// DiffStack compares a resolved stack configuration with the deployed stack
	DiffStack(ctx context.Context, stack *model.Stack, options Options) (*Result, error)
}

// Options configures what aspects of the stack to compare and how to format output
type Options struct {
	// Filter options - if all are false, compare everything
	TemplateOnly   bool // Only compare templates
	ParametersOnly bool // Only compare parameters
	TagsOnly       bool // Only compare tags

	// Output format
	Format string // "text" or "json"

	// Changeset lifecycle control
	KeepChangeSet bool // Keep changeset alive after diff (for deployment use)
}

// Result contains the results of a stack diff operation
type Result struct {
	StackName      string
	Environment    string
	StackExists    bool // Whether the stack exists in AWS
	TemplateChange *TemplateChange
	ParameterDiffs []ParameterDiff
	TagDiffs       []TagDiff
	ChangeSet      *ChangeSetInfo // AWS changeset information when available
	Options        Options        // Options used for this diff
}

// HasChanges returns true if any changes were detected
func (r *Result) HasChanges() bool {
	if !r.StackExists {
		return true // New stack is a change
	}

	if r.TemplateChange != nil && r.TemplateChange.HasChanges {
		return true
	}

	if len(r.ParameterDiffs) > 0 {
		return true
	}

	if len(r.TagDiffs) > 0 {
		return true
	}

	return false
}

// String returns a human-readable representation of the diff results
func (r *Result) String() string {
	if r.Options.Format == "json" {
		return r.toJSON()
	}
	return r.toText()
}

// TemplateChange represents differences in CloudFormation templates
type TemplateChange struct {
	HasChanges    bool
	CurrentHash   string // Hash of currently deployed template
	ProposedHash  string // Hash of proposed template
	Diff          string // Human-readable diff output
	ResourceCount struct {
		Added    int
		Modified int
		Removed  int
	}
}

// ParameterDiff represents a difference in stack parameters
type ParameterDiff struct {
	Key           string
	CurrentValue  string
	ProposedValue string
	ChangeType    ChangeType
}

// TagDiff represents a difference in stack tags
type TagDiff struct {
	Key           string
	CurrentValue  string
	ProposedValue string
	ChangeType    ChangeType
}

// ChangeType indicates the type of change detected
type ChangeType string

const (
	ChangeTypeAdd    ChangeType = "ADD"
	ChangeTypeModify ChangeType = "MODIFY"
	ChangeTypeRemove ChangeType = "REMOVE"
)

// ChangeSetInfo contains information from AWS CloudFormation changeset
type ChangeSetInfo struct {
	ChangeSetID string
	Status      string
	Changes     []ResourceChange
}

// ResourceChange represents a change to a CloudFormation resource
type ResourceChange struct {
	Action       string // CREATE, UPDATE, DELETE
	ResourceType string
	LogicalID    string
	PhysicalID   string
	Replacement  string // True, False, or Conditional
	Details      []string
}

// Comparator interfaces for different types of comparisons

// TemplateComparator handles CloudFormation template comparisons
type TemplateComparator interface {
	Compare(ctx context.Context, currentTemplate, proposedTemplate string) (*TemplateChange, error)
}

// ParameterComparator handles parameter comparisons
type ParameterComparator interface {
	Compare(currentParams, proposedParams map[string]string) ([]ParameterDiff, error)
}

// TagComparator handles tag comparisons
type TagComparator interface {
	Compare(currentTags, proposedTags map[string]string) ([]TagDiff, error)
}

// ChangeSetManager handles AWS CloudFormation changeset operations
type ChangeSetManager interface {
	CreateChangeSet(ctx context.Context, stackName string, template string, parameters map[string]string) (*ChangeSetInfo, error)
	CreateChangeSetForDeployment(ctx context.Context, stackName string, template string, parameters map[string]string, capabilities []string, tags map[string]string) (*ChangeSetInfo, error)
	DeleteChangeSet(ctx context.Context, changeSetID string) error
}
