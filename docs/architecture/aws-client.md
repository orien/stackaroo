# AWS Client Architecture

## Overview

The AWS client architecture uses a ClientFactory pattern that creates region-specific CloudFormation operations with shared credentials. This enables each stack to deploy to its context-defined region while maintaining efficient credential reuse.

## Core Components

### ClientFactory (`internal/aws/factory.go`)

Creates region-specific AWS clients with shared authentication:

```go
type ClientFactory interface {
    GetCloudFormationOperations(ctx context.Context, region string) (CloudFormationOperations, error)
    GetBaseConfig() aws.Config
    ValidateRegion(region string) error
}

type DefaultClientFactory struct {
    baseConfig  aws.Config      // Shared credentials
    clientCache map[string]CloudFormationOperations  // Cached by region
    mutex       sync.RWMutex
}
```

**Key Features:**
- **Credential Sharing**: Single authentication across all regions
- **Client Caching**: Region-specific clients cached for performance
- **Thread Safety**: Concurrent access protection

### CloudFormation Operations (`internal/aws/cloudformation.go`)

Provides high-level CloudFormation operations:

```go
type CloudFormationOperations interface {
    // Stack lifecycle
    DeployStack(ctx context.Context, input DeployStackInput) error
    DeployStackWithCallback(ctx context.Context, input DeployStackInput, callback func(StackEvent)) error
    DeleteStack(ctx context.Context, input DeleteStackInput) error
    
    // Stack information
    StackExists(ctx context.Context, stackName string) (bool, error)
    DescribeStack(ctx context.Context, stackName string) (*StackInfo, error)
    GetTemplate(ctx context.Context, stackName string) (string, error)
    
    // Change management
    CreateChangeSetPreview(ctx context.Context, stackName, template string, params map[string]string) (*ChangeSetInfo, error)
    ExecuteChangeSet(ctx context.Context, changeSetID string) error
    DeleteChangeSet(ctx context.Context, changeSetID string) error
    
    // Operations
    ValidateTemplate(ctx context.Context, templateBody string) error
    WaitForStackOperation(ctx context.Context, stackName string, callback func(StackEvent)) error
}
```

## Usage Patterns

### Basic Usage

```go
// Create factory (once per application)
factory, err := aws.NewClientFactory(ctx)

// Get region-specific operations
cfnOps, err := factory.GetCloudFormationOperations(ctx, "us-east-1")

// Use operations
err = cfnOps.DeployStack(ctx, deployInput)
```

### Multi-Region Deployment

```go
// Deploy to multiple regions with shared credentials
regions := []string{"us-east-1", "eu-west-1", "ap-southeast-2"}

for _, region := range regions {
    cfnOps, err := factory.GetCloudFormationOperations(ctx, region)
    if err != nil {
        return err
    }
    
    err = cfnOps.DeployStack(ctx, input)
    if err != nil {
        return err
    }
}
```

### Cross-Region Stack Dependencies

```go
// Reference stack output from different region
parameters := map[string]*config.ParameterValue{
    "VpcId": {
        ResolutionType: "stack-output",
        ResolutionConfig: map[string]string{
            "stack_name": "vpc-stack",
            "output_key": "VpcId",
            "region":     "us-west-2",  // Different region
        },
    },
}
```

## Integration

### Command Layer

```go
func getClientFactory() aws.ClientFactory {
    if clientFactory != nil {
        return clientFactory
    }
    
    ctx := context.Background()
    factory, err := aws.NewClientFactory(ctx)
    if err != nil {
        panic(fmt.Sprintf("failed to create AWS client factory: %v", err))
    }
    
    clientFactory = factory
    return clientFactory
}
```

### Service Layer

```go
type StackDeployer struct {
    clientFactory aws.ClientFactory
    provider      config.ConfigProvider
    resolver      resolve.Resolver
}

func (d *StackDeployer) DeployStack(ctx context.Context, stack *model.Stack) error {
    // Get region-specific operations
    cfnOps, err := d.clientFactory.GetCloudFormationOperations(ctx, stack.Context.Region)
    if err != nil {
        return fmt.Errorf("failed to get CloudFormation operations for region %s: %w", stack.Context.Region, err)
    }
    
    // Deploy using region-specific client
    return cfnOps.DeployStack(ctx, deployInput)
}
```

## Testing

### Mock Factory

```go
// Create mock factory with region-specific operations
mockFactory := aws.NewMockClientFactory()
mockOps := &aws.MockCloudFormationOperations{}
mockFactory.SetOperations("us-east-1", mockOps)

// Use in tests
deployer := deploy.NewStackDeployer(mockFactory, provider, resolver)
```

### Test Utilities

```go
// Helper functions for testing
factory, mockOps := aws.NewMockClientFactoryForRegion("us-east-1")
multiRegionFactory := aws.SetupMockFactoryForMultiRegion(map[string]aws.CloudFormationOperations{
    "us-east-1": mockOpsUS,
    "eu-west-1": mockOpsEU,
})
```

## Error Handling

### Wrapped Context

All operations provide contextual error information:

```go
err = cfnOps.DeployStack(ctx, input)
// Returns: "failed to create stack my-stack in region us-east-1: ValidationError: ..."
```

### Special Error Types

```go
type NoChangesError struct {
    StackName string
}

func (e NoChangesError) Error() string {
    return fmt.Sprintf("no changes to deploy for stack %s", e.StackName)
}
```

## Security

- **Credential Chain**: Uses AWS SDK default credential chain (environment, profiles, IAM roles)
- **Least Privilege**: Each operation uses minimal required permissions
- **Region Isolation**: Client operations are scoped to specific regions

## Performance

- **Connection Reuse**: Clients cached by region to avoid recreation
- **Concurrent Safe**: Thread-safe factory operations
- **Memory Efficient**: Only creates clients for regions actually used