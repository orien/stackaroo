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

The main `DefaultClient` struct serves as the entry point for all AWS operations:

```go
type DefaultClient struct {
    config aws.Config
    cfn    *cloudformation.Client
}
```

The `DefaultClient` implements the `Client` interface:

```go
type Client interface {
    NewCloudFormationOperations() CloudFormationOperations
}
```

**Responsibilities:**
- Load and manage AWS configuration
- Create and cache service-specific clients
- Provide factory methods for service operations
- Implement interfaces for testability and dependency injection

**Key Methods:**
- `NewDefaultClient(ctx, Config)` - Create new client with custom configuration
- `NewCloudFormationOperations()` - Create CloudFormation operations (implements interface)
- `CloudFormation()` - Access underlying CloudFormation client
- `Region()` - Get configured AWS region

### CloudFormation Operations (`internal/aws/cloudformation.go`)

The `DefaultCloudFormationOperations` struct provides high-level CloudFormation operations:

```go
type DefaultCloudFormationOperations struct {
    client CloudFormationClient
}
```

The operations implement the `CloudFormationOperations` interface:

```go
// Import context for interface:
// "github.com/aws/aws-sdk-go-v2/service/cloudformation"

type CloudFormationOperations interface {
    DeployStack(ctx context.Context, input DeployStackInput) error
    UpdateStack(ctx context.Context, input UpdateStackInput) error
    DeleteStack(ctx context.Context, input DeleteStackInput) error
    GetStack(ctx context.Context, stackName string) (*Stack, error)
    ListStacks(ctx context.Context) ([]*Stack, error)
    ValidateTemplate(ctx context.Context, templateBody string) error
    StackExists(ctx context.Context, stackName string) (bool, error)
    GetTemplate(ctx context.Context, stackName string) (string, error)
    DescribeStack(ctx context.Context, stackName string) (*StackInfo, error)
    CreateChangeSet(ctx context.Context, params *cloudformation.CreateChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateChangeSetOutput, error)
    DeleteChangeSet(ctx context.Context, params *cloudformation.DeleteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteChangeSetOutput, error)
    DescribeChangeSet(ctx context.Context, params *cloudformation.DescribeChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeChangeSetOutput, error)
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
- `GetTemplate()` - Retrieve template content for existing stacks
- `DescribeStack()` - Get detailed stack information including template
- `CreateChangeSet()` - Create CloudFormation changesets for diff operations
- `DeleteChangeSet()` - Remove CloudFormation changesets
- `DescribeChangeSet()` - Get changeset details and proposed changes

**Data Types:**
- `Stack` - Represents CloudFormation stack with cleaned-up fields
- `StackInfo` - Detailed stack information including template content
- `Parameter` - Key-value pairs for stack parameters
- `StackStatus` - Enumerated stack status values
- Input structs for each operation with required and optional fields

**Interface Design:**
All operations are interface-based to support dependency injection and testing:
- `Client` - Main AWS client abstraction
- `CloudFormationOperations` - CloudFormation-specific operations
- `CloudFormationClient` - Low-level CloudFormation client interface

## Usage Patterns

### Basic Client Creation

```go
// Default configuration (uses AWS default credential chain)
client, err := aws.NewDefaultClient(ctx, aws.Config{})

// Custom configuration
client, err := aws.NewDefaultClient(ctx, aws.Config{
    Region:  "us-west-2",
    Profile: "production",
})
```

### CloudFormation Operations

```go
// Get CloudFormation operations (returns interface)
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

1. **Add service client to `DefaultClient` struct:**
   ```go
   type DefaultClient struct {
       config aws.Config
       cfn    *cloudformation.Client
       s3     *s3.Client  // New service
   }
   ```

2. **Initialize in `NewDefaultClient()`:**
   ```go
   s3Client := s3.NewFromConfig(awsCfg)
   ```

3. **Create operations wrapper:**
   ```go
   // internal/aws/s3.go
   type DefaultS3Operations struct {
       client *s3.Client
   }
   
   func (c *DefaultClient) NewS3Operations() S3Operations {
       return &DefaultS3Operations{client: c.s3}
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
client, err := aws.NewDefaultClient(ctx, aws.Config{
    // Custom credentials via AWS SDK options
})
```

## Testing Strategy

### Interface-Based Testing
The AWS client architecture is designed for comprehensive testing using interface-based mocking:

#### Primary Testing Pattern
```go
import "github.com/stretchr/testify/mock"

// MockCloudFormationOperations implements CloudFormationOperations
type MockCloudFormationOperations struct {
    mock.Mock
}

func (m *MockCloudFormationOperations) DeployStack(ctx context.Context, input aws.DeployStackInput) error {
    args := m.Called(ctx, input)
    return args.Error(0)
}
```

#### Integration with Business Logic
The `internal/deploy` package uses dependency injection for testability:

```go
// In production
deployer := deploy.NewAWSDeployer(awsClient)

// In tests  
mockClient := &MockAWSClient{}
deployer := deploy.NewAWSDeployer(mockClient)
```

### Unit Testing
- Mock all interfaces (`Client`, `CloudFormationOperations`)
- Use `testify/mock` for professional mocking with expectations
- Test business logic in isolation from AWS SDK
- Fast, deterministic tests with no external dependencies

### Integration Testing  
- Use AWS localstack or moto for local AWS service simulation
- Provide test configuration for different AWS environments
- End-to-end testing with real AWS SDK behavior

### Testing Best Practices
- All external dependencies are abstracted behind interfaces
- Use dependency injection to substitute mocks in tests
- Follow the patterns established in `cmd/deploy_test.go`
- Verify mock expectations with `AssertExpectations()`

## Performance Considerations

### Connection Reuse
- Single `DefaultClient` instance should be reused across operations
- AWS SDK v2 handles connection pooling automatically
- Service clients are cached within the `DefaultClient` instance

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

## Integration with Stackaroo Components

### Deployment Layer
The AWS client integrates with the `internal/deploy` package:

```go
// AWSDeployer uses the Client interface
type AWSDeployer struct {
    awsClient aws.Client
}

func NewAWSDeployer(awsClient aws.Client) *AWSDeployer {
    return &AWSDeployer{awsClient: awsClient}
}
```

### CLI Integration
CLI commands use the deployment layer through dependency injection:

```go
// CLI uses Deployer interface
type Deployer interface {
    DeployStack(ctx context.Context, stackName, templateFile string) error
}

// Production: AWS implementation
deployer := deploy.NewDefaultDeployer(ctx)

// Testing: Mock implementation  
mockDeployer := &MockDeployer{}
cmd.SetDeployer(mockDeployer)
```

This layered architecture ensures clean separation of concerns and comprehensive testability throughout the application.
