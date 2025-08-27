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
    DeployStackWithCallback(ctx context.Context, input DeployStackInput, eventCallback func(StackEvent)) error
    UpdateStack(ctx context.Context, input UpdateStackInput) error
    DeleteStack(ctx context.Context, input DeleteStackInput) error
    GetStack(ctx context.Context, stackName string) (*Stack, error)
    ListStacks(ctx context.Context) ([]*Stack, error)
    ValidateTemplate(ctx context.Context, templateBody string) error
    StackExists(ctx context.Context, stackName string) (bool, error)
    GetTemplate(ctx context.Context, stackName string) (string, error)
    DescribeStack(ctx context.Context, stackName string) (*StackInfo, error)
    DescribeStackEvents(ctx context.Context, stackName string) ([]StackEvent, error)
    WaitForStackOperation(ctx context.Context, stackName string, eventCallback func(StackEvent)) error
    // Changeset operations
    ExecuteChangeSet(ctx context.Context, changeSetID string) error
    DeleteChangeSet(ctx context.Context, changeSetID string) error
    CreateChangeSetPreview(ctx context.Context, stackName string, template string, parameters map[string]string) (*ChangeSetInfo, error)
    CreateChangeSetForDeployment(ctx context.Context, stackName string, template string, parameters map[string]string, capabilities []string, tags map[string]string) (*ChangeSetInfo, error)
}
```

**Core Operations:**
- `DeployStack()` - Create or update CloudFormation stacks (simple version)
- `DeployStackWithCallback()` - Create or update stacks with real-time event streaming
- `UpdateStack()` - Update existing stacks
- `DeleteStack()` - Delete stacks
- `GetStack()` - Retrieve stack information
- `ListStacks()` - List all stacks
- `ValidateTemplate()` - Validate CloudFormation templates
- `StackExists()` - Check stack existence
- `GetTemplate()` - Retrieve template content for existing stacks
- `DescribeStack()` - Get detailed stack information including template
- `DescribeStackEvents()` - Retrieve CloudFormation events for a stack
- `WaitForStackOperation()` - Wait for stack operation completion with event streaming
- `CreateChangeSet()` - Create CloudFormation changesets for diff operations
- `DeleteChangeSet()` - Remove CloudFormation changesets
- `DescribeChangeSet()` - Get changeset details and proposed changes

**Data Types:**
- `Stack` - Represents CloudFormation stack with cleaned-up fields
- `StackInfo` - Detailed stack information including template content
- `StackEvent` - Represents CloudFormation stack events with timestamp and resource information
- `Parameter` - Key-value pairs for stack parameters
- `StackStatus` - Enumerated stack status values
- `NoChangesError` - Special error type indicating no changes need to be deployed
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

// Deploy a stack with event streaming
eventCallback := func(event aws.StackEvent) {
    timestamp := event.Timestamp.Format("2006-01-02 15:04:05")
    fmt.Printf("[%s] %-20s %-40s %s\n", 
        timestamp, event.ResourceStatus, event.ResourceType, event.LogicalResourceId)
}

err := cfnOps.DeployStackWithCallback(ctx, aws.DeployStackInput{
    StackName:    "my-stack",
    TemplateBody: templateContent,
    Parameters: []aws.Parameter{
        {Key: "Environment", Value: "prod"},
    },
    Tags: map[string]string{
        "Project": "stackaroo",
    },
}, eventCallback)

// Handle no changes scenario
if errors.As(err, &aws.NoChangesError{}) {
    fmt.Println("Stack is already up to date - no changes to deploy")
} else if err != nil {
    return fmt.Errorf("deployment failed: %w", err)
}

// Simple deployment without events
err := cfnOps.DeployStack(ctx, aws.DeployStackInput{
    StackName:    "my-stack",
    TemplateBody: templateContent,
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

## Advanced Deployment Features

### Smart Create vs Update Detection

The `DeployStack()` and `DeployStackWithCallback()` methods automatically detect whether a stack exists and perform the appropriate operation:

- **New stacks**: Calls CloudFormation `CreateStack` operation
- **Existing stacks**: Calls CloudFormation `UpdateStack` operation
- **No changes needed**: Returns `NoChangesError` for graceful handling

```go
// This automatically determines create vs update
err := cfnOps.DeployStack(ctx, aws.DeployStackInput{
    StackName:    "my-stack",
    TemplateBody: templateContent,
})

// Handle the no changes scenario
var noChangesErr aws.NoChangesError
if errors.As(err, &noChangesErr) {
    fmt.Printf("Stack %s is already up to date\n", noChangesErr.StackName)
    return nil
}
```

### Real-time Event Streaming

The `DeployStackWithCallback()` method provides real-time CloudFormation event streaming during deployment operations:

```go
// Define event callback for real-time feedback
eventCallback := func(event aws.StackEvent) {
    timestamp := event.Timestamp.Format("2006-01-02 15:04:05")
    fmt.Printf("[%s] %-20s %-40s %s %s\n", 
        timestamp,
        event.ResourceStatus,
        event.ResourceType, 
        event.LogicalResourceId,
        event.ResourceStatusReason,
    )
}

// Deploy with event streaming
err := cfnOps.DeployStackWithCallback(ctx, deployInput, eventCallback)
```

**Event Output Example:**
```
Starting create operation for stack my-app...
[2025-01-09 15:30:45] CREATE_IN_PROGRESS   AWS::CloudFormation::Stack  my-app       User Initiated
[2025-01-09 15:30:46] CREATE_IN_PROGRESS   AWS::S3::Bucket              AppBucket    
[2025-01-09 15:30:48] CREATE_COMPLETE      AWS::S3::Bucket              AppBucket    
[2025-01-09 15:30:50] CREATE_IN_PROGRESS   AWS::Lambda::Function        AppFunction  
[2025-01-09 15:30:55] CREATE_COMPLETE      AWS::Lambda::Function        AppFunction  
[2025-01-09 15:30:56] CREATE_COMPLETE      AWS::CloudFormation::Stack  my-app       
Stack my-app create completed successfully
```

### Operation Waiting and Polling

The deployment operations automatically wait for completion:

- **Polling interval**: 5 seconds between status checks
- **Event deduplication**: Only new events are reported via callback
- **Completion detection**: Monitors stack status for terminal states
- **Error handling**: Distinguishes between successful and failed operations

```go
// This will wait until the operation completes or fails
err := cfnOps.WaitForStackOperation(ctx, "my-stack", eventCallback)
if err != nil {
    // Handle deployment failure
    return fmt.Errorf("stack operation failed: %w", err)
}
```

**Terminal Stack States:**
- **Success**: `CREATE_COMPLETE`, `UPDATE_COMPLETE`, `DELETE_COMPLETE`
- **Failure**: `CREATE_FAILED`, `UPDATE_FAILED`, `ROLLBACK_COMPLETE`, etc.

## Changeset Operations

The CloudFormation operations include high-level changeset methods for advanced deployment workflows with automatic cleanup and error handling.

### Changeset Workflow Methods

#### Preview Changes (Auto-Cleanup)

Use `CreateChangeSetPreview()` to preview changes without deploying. The changeset is automatically deleted after analysis:

```go
// Preview changes (changeset auto-deleted)
changeSetInfo, err := cfnOps.CreateChangeSetPreview(ctx, stackName, template, parameters)
if err != nil {
    return fmt.Errorf("failed to preview changes: %w", err)
}

// Analyze changes
for _, change := range changeSetInfo.Changes {
    fmt.Printf("%s: %s (%s)\n", change.Action, change.LogicalID, change.ResourceType)
}
```

#### Deploy with Changeset

Use `CreateChangeSetForDeployment()` to create a changeset for execution (persists until executed):

```go
// Create changeset for deployment
changeSetInfo, err := cfnOps.CreateChangeSetForDeployment(ctx, 
    stackName, template, parameters, capabilities, tags)
if err != nil {
    return fmt.Errorf("failed to create changeset: %w", err)
}

// Execute the changeset
err = cfnOps.ExecuteChangeSet(ctx, changeSetInfo.ChangeSetID)
```

### ChangeSetInfo Structure

The high-level methods return a `ChangeSetInfo` struct with parsed change details:

```go
type ChangeSetInfo struct {
    ChangeSetID string
    Status      string
    Changes     []ResourceChange
}

type ResourceChange struct {
    Action       string   // CREATE, UPDATE, DELETE
    ResourceType string   // AWS::S3::Bucket, etc.
    LogicalID    string   // Resource name in template
    PhysicalID   string   // AWS resource ID
    Replacement  string   // True, False, or Conditional
    Details      []string // Property change details
}
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
- **No changes needed**: Special `NoChangesError` type for update operations with no changes

### Special Error Types

#### NoChangesError
When updating a stack that requires no changes, a special `NoChangesError` is returned:

```go
type NoChangesError struct {
    StackName string
}

func (e NoChangesError) Error() string {
    return fmt.Sprintf("stack %s is already up to date - no changes to deploy", e.StackName)
}
```

This allows applications to distinguish between actual deployment failures and successful "no changes" scenarios.

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

// Handle deployment with no changes
err := cfnOps.DeployStack(ctx, deployInput)
if err != nil {
    var noChangesErr aws.NoChangesError
    if errors.As(err, &noChangesErr) {
        fmt.Printf("Stack %s is already up to date\n", noChangesErr.StackName)
        return nil // This is success, not an error
    }
    return fmt.Errorf("deployment failed: %w", err)
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

The AWS package provides a comprehensive `MockCloudFormationClient` that implements the `CloudFormationClient` interface for testing all CloudFormation operations, including changeset functionality:

```go
import "github.com/stretchr/testify/mock"

// MockCloudFormationClient implements CloudFormationClient for testing
type MockCloudFormationClient struct {
    mock.Mock
}

// Example: Testing high-level changeset operations
func TestCreateChangeSetPreview(t *testing.T) {
    ctx := context.Background()
    mockClient := &MockCloudFormationClient{}
    cf := &DefaultCloudFormationOperations{client: mockClient}
    
    // Mock the underlying AWS calls
    mockClient.On("CreateChangeSet", ctx, mock.AnythingOfType("*cloudformation.CreateChangeSetInput")).
        Return(createTestChangeSetOutput("changeset-123"), nil)
    mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).
        Return(createTestDescribeOutput(), nil)
    mockClient.On("DeleteChangeSet", ctx, mock.AnythingOfType("*cloudformation.DeleteChangeSetInput")).
        Return(&cloudformation.DeleteChangeSetOutput{}, nil)
    
    // Test the high-level operation
    changeSetInfo, err := cf.CreateChangeSetPreview(ctx, "test-stack", template, parameters)
    
    require.NoError(t, err)
    assert.Equal(t, "changeset-123", changeSetInfo.ChangeSetID)
    mockClient.AssertExpectations(t)
}
```

**Key Testing Benefits:**
- **Single Mock**: One consolidated mock eliminates code duplication
- **Full Coverage**: Tests both low-level SDK calls and high-level workflows  
- **Changeset Support**: Complete testing of changeset preview and deployment flows
- **Professional Mocking**: Uses testify/mock with expectations and assertions

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

#### CloudFormation Operations Testing
- **Mock Consolidation**: Single `MockCloudFormationClient` handles all testing scenarios
- Mock all interfaces (`Client`, `CloudFormationOperations`)
- Use `testify/mock` for professional mocking with expectations
- Test business logic in isolation from AWS SDK
- Fast, deterministic tests with no external dependencies

#### Event Callback Testing
When testing operations with event callbacks, use function type matchers:

```go
// Test deployment with event streaming
mockCfnOps.On("DeployStackWithCallback", 
    ctx,
    mock.MatchedBy(func(input aws.DeployStackInput) bool {
        return input.StackName == "test-stack"
    }),
    mock.AnythingOfType("func(aws.StackEvent)"),  // Event callback matcher
).Return(nil)

// Test event callback invocation
var capturedEvents []aws.StackEvent
eventCallback := func(event aws.StackEvent) {
    capturedEvents = append(capturedEvents, event)
}

err := cfnOps.DeployStackWithCallback(ctx, input, eventCallback)
assert.NoError(t, err)
assert.Len(t, capturedEvents, expectedEventCount)
```

#### Testing NoChangesError
```go
// Test no changes scenario
mockCfnOps.On("DeployStackWithCallback", ctx, input, mock.AnythingOfType("func(aws.StackEvent)")).
    Return(aws.NoChangesError{StackName: "test-stack"})

err := deployer.DeployStack(ctx, stack)
assert.NoError(t, err)  // Should handle NoChangesError gracefully
```

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
    DeployStack(ctx context.Context, resolvedStack *model.Stack) error
    ValidateTemplate(ctx context.Context, templateFile string) error
}

// Production: AWS implementation
deployer := deploy.NewDefaultDeployer(ctx)

// Testing: Mock implementation  
mockDeployer := &MockDeployer{}
cmd.SetDeployer(mockDeployer)
```

This layered architecture ensures clean separation of concerns and comprehensive testability throughout the application.
