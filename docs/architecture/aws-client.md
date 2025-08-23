# AWS Client Architecture

## Overview

The AWS client abstraction provides a clean, type-safe interface for interacting with AWS services in Stackaroo. It wraps the AWS SDK v2 to provide higher-level operations whilst maintaining flexibility and extensibility.

## Design Principles

### 1. **Type Safety**
- Use Go structs instead of AWS SDK pointers throughout the public API
- Provide clear, strongly-typed parameters and return values
- Hide AWS SDK complexity behind clean interfaces

### 2. **Extensibility**
- Modular design allowing easy addition of new AWS services
- Service-specific operation wrappers that share common configuration
- Direct access to underlying SDK clients when needed

### 3. **Error Handling**
- Wrap AWS errors with contextual information
- Provide meaningful error messages for common scenarios
- Distinguish between different types of failures (not found, permission denied, etc.)

### 4. **Configuration**
- Support multiple AWS configuration methods (profiles, environment variables, IAM roles)
- Allow per-operation overrides of region and credentials
- Consistent configuration across all AWS service interactions

## Architecture Components

### Core Client (`internal/aws/client.go`)

The main `Client` struct serves as the entry point for all AWS operations:

```go
type Client struct {
    config aws.Config
    cfn    *cloudformation.Client
}
```

**Responsibilities:**
- Load and manage AWS configuration
- Create and cache service-specific clients
- Provide factory methods for service operations

**Key Methods:**
- `NewClient(ctx, Config)` - Create new client with custom configuration
- `CloudFormation()` - Access underlying CloudFormation client
- `Region()` - Get configured AWS region

### CloudFormation Operations (`internal/aws/cloudformation.go`)

The `CloudFormationOperations` struct provides high-level CloudFormation operations:

```go
type CloudFormationOperations struct {
    client *cloudformation.Client
}
```

**Core Operations:**
- `DeployStack()` - Create new CloudFormation stacks
- `UpdateStack()` - Update existing stacks
- `DeleteStack()` - Delete stacks
- `GetStack()` - Retrieve stack information
- `ListStacks()` - List all stacks
- `ValidateTemplate()` - Validate CloudFormation templates
- `StackExists()` - Check stack existence

**Data Types:**
- `Stack` - Represents CloudFormation stack with cleaned-up fields
- `Parameter` - Key-value pairs for stack parameters
- `StackStatus` - Enumerated stack status values
- Input structs for each operation with required and optional fields

## Usage Patterns

### Basic Client Creation

```go
// Default configuration (uses AWS default credential chain)
client, err := aws.NewClient(ctx, aws.Config{})

// Custom configuration
client, err := aws.NewClient(ctx, aws.Config{
    Region:  "us-west-2",
    Profile: "production",
})
```

### CloudFormation Operations

```go
// Get CloudFormation operations
cfnOps := client.NewCloudFormationOperations()

// Deploy a stack
err := cfnOps.DeployStack(ctx, aws.DeployStackInput{
    StackName:    "my-stack",
    TemplateBody: templateContent,
    Parameters: []aws.Parameter{
        {Key: "Environment", Value: "prod"},
    },
    Tags: map[string]string{
        "Project": "stackaroo",
    },
})

// Check stack status
stack, err := cfnOps.GetStack(ctx, "my-stack")
fmt.Printf("Stack status: %s\n", stack.Status)
```

### Direct SDK Access

For advanced use cases, direct access to underlying SDK clients is available:

```go
// Direct CloudFormation client access
cfnClient := client.CloudFormation()

// Use SDK directly
result, err := cfnClient.DescribeStackEvents(ctx, &cloudformation.DescribeStackEventsInput{
    StackName: aws.String("my-stack"),
})
```

## Configuration Hierarchy

The client respects the standard AWS configuration hierarchy:

1. **Explicit parameters** to `NewClient()`
2. **Environment variables** (`AWS_REGION`, `AWS_PROFILE`, etc.)
3. **Shared configuration files** (`~/.aws/config`, `~/.aws/credentials`)
4. **IAM roles** for EC2/ECS/Lambda execution
5. **AWS SDK defaults**

## Error Handling Strategy

### Wrapped Errors
All operations wrap AWS SDK errors with contextual information:

```go
return fmt.Errorf("failed to create stack %s: %w", input.StackName, err)
```

### Error Classification
Common error scenarios are identified and handled appropriately:

- **Stack not found**: Differentiated from other validation errors
- **Permission denied**: Clear indication of IAM policy issues
- **Template validation**: Detailed error messages for template problems
- **Rate limiting**: Retryable errors with appropriate backoff

### Error Examples

```go
// Check for specific error types
stack, err := cfnOps.GetStack(ctx, "nonexistent-stack")
if err != nil {
    if isStackNotFoundError(err) {
        // Handle stack not found
    } else {
        // Handle other errors
    }
}
```

## Extension Points

### Adding New AWS Services

To add support for additional AWS services (S3, EC2, etc.):

1. **Add service client to `Client` struct:**
   ```go
   type Client struct {
       config aws.Config
       cfn    *cloudformation.Client
       s3     *s3.Client  // New service
   }
   ```

2. **Initialize in `NewClient()`:**
   ```go
   s3Client := s3.NewFromConfig(awsCfg)
   ```

3. **Create operations wrapper:**
   ```go
   // internal/aws/s3.go
   type S3Operations struct {
       client *s3.Client
   }
   
   func (c *Client) NewS3Operations() *S3Operations {
       return &S3Operations{client: c.s3}
   }
   ```

4. **Implement high-level operations:**
   ```go
   func (s3ops *S3Operations) UploadTemplate(ctx context.Context, input UploadInput) error {
       // Implementation
   }
   ```

### Custom Authentication

For special authentication requirements:

```go
// Custom credential provider
client, err := aws.NewClient(ctx, aws.Config{
    // Custom credentials via AWS SDK options
})
```

## Testing Strategy

### Unit Testing
- Mock the `CloudFormationOperations` interface for business logic testing
- Use AWS SDK v2's testing utilities for service-level testing

### Integration Testing  
- Use AWS localstack or moto for local CloudFormation testing
- Provide test configuration for different AWS environments

### Example Test Structure
```go
type MockCloudFormationOperations struct {
    deployStackFunc func(ctx context.Context, input aws.DeployStackInput) error
}

func (m *MockCloudFormationOperations) DeployStack(ctx context.Context, input aws.DeployStackInput) error {
    return m.deployStackFunc(ctx, input)
}
```

## Performance Considerations

### Connection Reuse
- Single `Client` instance should be reused across operations
- AWS SDK v2 handles connection pooling automatically
- Service clients are cached within the `Client` instance

### Context Handling
- All operations accept `context.Context` for timeout and cancellation
- Long-running operations (stack deployments) should use appropriate timeouts
- Context cancellation properly terminates AWS SDK operations

### Memory Management
- Large template bodies should be streamed rather than loaded entirely into memory
- Stack listings use pagination to handle large numbers of stacks
- Proper cleanup of resources in error scenarios

## Security Considerations

### Credential Handling
- Never log or expose AWS credentials in error messages
- Use IAM roles and temporary credentials where possible
- Support AWS credential rotation patterns

### Template Security
- Validate templates before deployment to prevent injection attacks
- Support template parameter validation
- Sanitise user inputs in template parameters

### Least Privilege
- Operations only request necessary permissions
- Clear documentation of required IAM permissions for each operation
- Support for cross-account deployment patterns