/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package describe

import (
	"context"
	"time"

	"github.com/orien/stackaroo/internal/model"
)

// Describer defines the interface for retrieving detailed stack information
type Describer interface {
	DescribeStack(ctx context.Context, stack *model.Stack) (*StackDescription, error)
}

// StackDescription contains comprehensive information about a CloudFormation stack
type StackDescription struct {
	// Basic stack information
	Name        string
	Status      string
	StackID     string
	CreatedTime time.Time
	UpdatedTime *time.Time
	Description string

	// Stack configuration
	Parameters map[string]string
	Outputs    map[string]string
	Tags       map[string]string

	// Additional metadata
	Region string
}
