# Delete Module Architecture

## Overview

The delete module provides safe, dependency-aware deletion of CloudFormation stacks with comprehensive confirmation mechanisms. It ensures stacks are deleted in reverse dependency order to prevent AWS failures, while requiring explicit user confirmation for all destructive operations.

## Architecture Diagram

```mermaid
graph TB
    subgraph "Command Layer"
        CLI[delete command<br/>cmd/delete.go]
    end
    
    subgraph "Core Delete Module"
        DI[Deleter Interface<br/>deleter.go]
        AD[AWSDeleter<br/>deleter.go]
        
        subgraph "Safety Features"
            SP[Stack Preview<br/>deleter.go]
            UC[User Confirmation<br/>via prompt]
            EV[Existence Validation<br/>deleter.go]
        end
        
        subgraph "Deletion Orchestration"
            DO[Dependency Ordering<br/>cmd/delete.go]
            DS[Deletion Sequencing<br/>deleter.go]
            EM[Event Monitoring<br/>deleter.go]
        end
    end
    
    subgraph "External Dependencies"
        AWS[AWS CloudFormation<br/>internal/aws]
        PROMPT[Prompt System<br/>internal/prompt]
        CFG[Configuration<br/>internal/config]
        RES[Stack Resolver<br/>internal/resolve]
    end
    
    CLI --> DI
    DI --> AD
    AD --> SP
    AD --> UC
    AD --> EV
    CLI --> DO
    DO --> DS
    DS --> EM
    
    AD --> AWS
    UC --> PROMPT
    CLI --> CFG
    CLI --> RES
```

## Component Architecture

### 1. Command Layer (`cmd/delete.go`)

**Responsibility:** CLI interface, dependency resolution, and deletion orchestration

```mermaid
classDiagram
    class DeleteCommand {
        +contextName: string
        +stackName: string
        +RunE() error
        +deleteWithConfig() error
        +getDeleter() Deleter
    }
    
    class Deleter {
        <<interface>>
        +DeleteStack(ctx, stack) error
    }
    
    DeleteCommand --> Deleter
```

**Key Features:**
- Single and multiple stack deletion support
- Automatic dependency resolution and reverse ordering
- Configuration integration
- Error handling with proper exit codes

**Dependency Ordering Logic:**
```mermaid
graph LR
    A[Resolve Dependencies] --> B[Get Deployment Order]
    B --> C[Reverse Order]
    C --> D[Delete in Sequence]
    
    subgraph "Example"
        E[Deploy: vpc → db → app]
        F[Delete: app → db → vpc]
        E --> F
    end
```

### 2. Core Delete Engine (`internal/delete/`)

#### 2.1 Deleter Interface and Implementation

```mermaid
classDiagram
    class Deleter {
        <<interface>>
        +DeleteStack(ctx, stack) error
    }
    
    class AWSDeleter {
        -awsClient: Client
        +DeleteStack(ctx, stack) error
        -validateStackExists() bool
        -previewDeletion() error
        -confirmDeletion() bool
        -executeDelete() error
        -waitForCompletion() error
    }
    
    Deleter <|-- AWSDeleter
```

**Key Responsibilities:**
- Stack existence validation
- Deletion preview generation
- User confirmation handling
- AWS deletion execution
- Operation monitoring and feedback

#### 2.2 Safety and Confirmation Flow

```mermaid
stateDiagram-v2
    [*] --> CheckExists
    CheckExists --> NotExists : Stack doesn't exist
    CheckExists --> GetStackInfo : Stack exists
    
    NotExists --> Skip : Log skip message
    Skip --> [*]
    
    GetStackInfo --> ShowPreview : Success
    GetStackInfo --> Error : AWS error
    
    ShowPreview --> PromptUser : Display details
    PromptUser --> UserCancel : User says no
    PromptUser --> ExecuteDelete : User confirms
    
    UserCancel --> Cancel : Log cancellation
    Cancel --> [*]
    
    ExecuteDelete --> WaitCompletion : AWS delete initiated
    WaitCompletion --> Success : Delete completed
    WaitCompletion --> Error : Delete failed
    
    Success --> [*]
    Error --> [*]
```

### 3. Safety Features

#### 3.1 Stack Preview

```mermaid
classDiagram
    class StackPreview {
        +displayStackInfo(stackInfo) void
        +showDeletionWarning() void
        +formatStackStatus(status) string
    }
    
    class StackInfo {
        +Name: string
        +Status: string
        +Description: string
        +CreatedTime: time
        +UpdatedTime: time
    }
    
    StackPreview --> StackInfo
```

**Preview Components:**
- Stack name and context
- Current stack status
- Stack description
- Destructive operation warnings
- Cannot-be-undone disclaimers

#### 3.2 User Confirmation Integration

```mermaid
sequenceDiagram
    participant D as Deleter
    participant P as Prompt System
    participant U as User
    
    D->>D: Generate confirmation message
    D->>P: Confirm("Do you want to delete stack X? This cannot be undone.")
    P->>P: Format with "\n" + "[y/N]: "
    P->>U: Display formatted prompt
    U->>P: User input (y/n)
    P->>P: Parse response
    P->>D: Return boolean decision
    
    alt User confirms
        D->>D: Proceed with deletion
    else User cancels
        D->>D: Cancel operation
    end
```

## Data Flow Architecture

### Single Stack Deletion Flow

```mermaid
sequenceDiagram
    participant CLI as CLI Command
    participant Resolver as Stack Resolver
    participant Deleter as AWS Deleter
    participant AWS as AWS Client
    participant Prompt as Prompt System
    participant User as User
    
    CLI->>Resolver: Resolve stack configuration
    Resolver->>CLI: ResolvedStack
    
    CLI->>Deleter: DeleteStack(resolved)
    
    Deleter->>AWS: StackExists(stackName)
    AWS->>Deleter: true/false
    
    alt Stack Exists
        Deleter->>AWS: DescribeStack(stackName)
        AWS->>Deleter: StackInfo
        
        Deleter->>Deleter: Display preview
        Deleter->>Prompt: Confirm deletion
        Prompt->>User: Show formatted prompt
        User->>Prompt: y/n response
        Prompt->>Deleter: boolean confirmation
        
        alt User Confirms
            Deleter->>AWS: DeleteStack(stackName)
            AWS->>Deleter: Success
            
            Deleter->>AWS: WaitForStackOperation(callback)
            AWS->>Deleter: Deletion events
            Deleter->>Deleter: Display progress
            AWS->>Deleter: Deletion complete
        else User Cancels
            Deleter->>Deleter: Log cancellation
        end
    else Stack Doesn't Exist
        Deleter->>Deleter: Log skip message
    end
    
    Deleter->>CLI: Success/Error result
```

### Multiple Stack Deletion Flow

```mermaid
sequenceDiagram
    participant CLI as CLI Command
    participant Config as Config Provider
    participant Resolver as Stack Resolver
    participant Deleter as AWS Deleter
    
    CLI->>Config: ListStacks(context)
    Config->>CLI: []stackNames
    
    CLI->>Resolver: Resolve(context, stackNames)
    Resolver->>CLI: ResolvedStacks with order
    
    CLI->>CLI: Reverse deployment order
    
    loop For each stack in reverse order
        CLI->>Deleter: DeleteStack(stack)
        Deleter->>CLI: Result
        
        alt Deletion Failed
            CLI->>CLI: Return error (stop processing)
        else Deletion Succeeded
            CLI->>CLI: Continue to next stack
        end
    end
    
    CLI->>CLI: Report completion
```

## Integration Points

### 1. AWS Integration (`internal/aws`)

**Required CloudFormation Operations:**
- `StackExists(stackName)` - Validate stack existence
- `DescribeStack(stackName)` - Get detailed stack information
- `DeleteStack(input)` - Execute stack deletion
- `WaitForStackOperation(name, callback)` - Monitor deletion progress
- `DescribeStackEvents(stackName)` - Retrieve deletion events

**AWS Permissions Required:**
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "cloudformation:DescribeStacks",
                "cloudformation:DeleteStack",
                "cloudformation:DescribeStackEvents"
            ],
            "Resource": "*"
        }
    ]
}
```

### 2. Prompt System Integration (`internal/prompt`)

**Clean Separation of Concerns:**
```mermaid
graph LR
    subgraph "Business Logic"
        A[Delete Module] --> B[Generate Core Message]
        B --> C["Do you want to delete stack X? This cannot be undone."]
    end
    
    subgraph "UI Layer"
        D[Prompt System] --> E[Add Formatting]
        E --> F["\nDo you want to delete stack X? This cannot be undone. [y/N]: "]
    end
    
    C --> D
    F --> G[User Input]
```

**Benefits:**
- Delete module focuses on business logic and safety
- Prompt system handles UI formatting consistently
- No technical prompt details leak into deletion logic

### 3. Configuration Integration (`internal/config`)

**Dependencies:**
- `ConfigProvider.ListStacks(context)` - Get all stacks in context
- `ConfigProvider.GetStack(name, context)` - Retrieve specific stack config
- Context validation and error handling

### 4. Stack Resolution (`internal/resolve`)

**Dependency Management:**
- `StackResolver.Resolve(ctx, context, stackNames)` - Full dependency resolution
- `ResolvedStacks.DeploymentOrder` - Ordered list for reversal
- Complete dependency graph for safe deletion ordering

## Error Handling Strategy

```mermaid
graph TD
    A[Delete Request] --> B{Stack Exists?}
    B -->|No| C[Log Skip Message]
    B -->|Yes| D{Get Stack Info}
    
    D -->|Success| E[Show Preview]
    D -->|Error| F[AWS Error Handling]
    
    E --> G{User Confirms?}
    G -->|No| H[Log Cancellation]
    G -->|Yes| I[Execute Delete]
    
    I --> J{Delete Success?}
    J -->|Yes| K[Monitor Progress]
    J -->|No| L[Delete Error Handling]
    
    K --> M{Completion?}
    M -->|Success| N[Report Success]
    M -->|Timeout/Error| O[Operation Error]
    
    F --> P[Return Error with Context]
    H --> Q[Return Success - User Choice]
    L --> R[Return Error - AWS Failure]
    O --> S[Return Error - Operation Failed]
    
    C --> T[Continue]
    N --> T
    Q --> T
    P --> U[Stop]
    R --> U
    S --> U
```

**Error Categories:**
1. **Configuration Errors** - Invalid contexts, missing stacks
2. **AWS Errors** - Permissions, API failures, stack states
3. **User Cancellation** - Treated as successful completion
4. **Operation Errors** - Deletion timeouts, dependency violations

**Error Context Enhancement:**
```go
// Examples of contextual error wrapping
fmt.Errorf("failed to check if stack exists: %w", err)
fmt.Errorf("failed to delete stack %s: %w", stackName, err)
fmt.Errorf("stack deletion failed or timed out: %w", err)
```

## Safety Architecture

### 1. Multi-Layer Safety System

```mermaid
graph TD
    A[Delete Request] --> B[Layer 1: Existence Check]
    B --> C[Layer 2: Information Preview]
    C --> D[Layer 3: Explicit Confirmation]
    D --> E[Layer 4: Dependency Ordering]
    E --> F[Layer 5: AWS Execution]
    F --> G[Layer 6: Progress Monitoring]
    
    subgraph "Safety Gates"
        B1[Non-existent stacks skipped]
        C1[Complete stack information shown]
        D1[Cannot be undone warnings]
        E1[Safe deletion order guaranteed]
        F1[AWS-level validation]
        G1[Real-time feedback]
    end
```

### 2. Confirmation Message Design

**Business Logic (Domain-Specific):**
```go
message := fmt.Sprintf("Do you want to delete stack %s? This cannot be undone.", stack.Name)
```

**UI Layer (Technical Details):**
```go
formattedMessage := fmt.Sprintf("\n%s [y/N]: ", message)
```

**Key Safety Features:**
- Default to "No" (`[y/N]`)
- Explicit "cannot be undone" warning
- Clear stack identification
- Consistent formatting across all prompts

### 3. Dependency Safety

**Reverse Order Algorithm:**
```go
// Safe deletion order calculation
deletionOrder := make([]string, len(resolved.DeploymentOrder))
for i, stackName := range resolved.DeploymentOrder {
    deletionOrder[len(resolved.DeploymentOrder)-1-i] = stackName
}
```

**Benefits:**
- Prevents AWS dependency violations
- Reduces likelihood of partial failures
- Maintains referential integrity
- Follows CloudFormation best practices

## Testing Architecture

### Test Categories

```mermaid
graph LR
    subgraph "Unit Tests"
        A[Deleter Interface Tests]
        B[Safety Feature Tests] 
        C[Command Structure Tests]
        D[Error Handling Tests]
    end
    
    subgraph "Integration Tests"
        E[AWS Client Integration]
        F[Prompt Integration]
        G[End-to-End Workflows]
    end
    
    subgraph "Mock Infrastructure"
        H[Mock AWS Client]
        I[Mock Prompter]
        J[Mock Config Provider]
    end
    
    A --> H
    B --> I
    C --> J
    E --> H
    F --> I
    G --> H
```

**Comprehensive Test Scenarios:**

1. **Success Paths:**
   - Single stack deletion with confirmation
   - Multiple stack deletion in correct order
   - Non-existent stack handling

2. **Safety Tests:**
   - User cancellation handling
   - Confirmation prompt integration
   - Preview information display

3. **Error Scenarios:**
   - AWS client failures
   - Stack existence check errors
   - Deletion operation failures
   - Timeout handling

4. **Edge Cases:**
   - Empty contexts
   - Invalid stack names
   - Circular dependencies
   - Concurrent deletions

### Mock Implementation Examples

```go
// Mock AWS operations for safe testing
mockCfnOps.On("StackExists", ctx, "test-stack").Return(true, nil)
mockCfnOps.On("DescribeStack", ctx, "test-stack").Return(stackInfo, nil)
mockPrompter.On("Confirm", "Do you want to delete stack test-stack? This cannot be undone.").Return(true, nil)
mockCfnOps.On("DeleteStack", ctx, deleteInput).Return(nil)
```

## Security Considerations

### 1. Authentication and Authorization

**AWS Credential Chain:**
- Uses AWS SDK v2 default credential provider chain
- Supports AWS profiles, environment variables, IAM roles
- Respects AWS region configuration

**Required IAM Permissions:**
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "cloudformation:DescribeStacks",
                "cloudformation:DescribeStackEvents",
                "cloudformation:DeleteStack"
            ],
            "Resource": [
                "arn:aws:cloudformation:*:*:stack/*"
            ]
        }
    ]
}
```

### 2. Operational Security

**Audit Trail:**
- All deletion attempts logged with context
- User confirmation decisions recorded
- Stack state changes tracked
- Error conditions documented

**Fail-Safe Mechanisms:**
- Default denial on unclear user input
- Explicit confirmation required for each stack
- No bulk deletion without individual confirmation
- Operation halts on first failure in multi-stack deletion

### 3. Data Protection

**Information Handling:**
- Stack names and contexts logged safely
- No sensitive parameter values in logs
- Error messages sanitised to prevent information leakage
- AWS API responses handled securely

## Performance Considerations

### Optimisation Strategies

1. **Sequential Deletion** - Ensures dependency safety over speed
2. **Early Termination** - Stops processing on first error
3. **Efficient Existence Checks** - Quick validation before expensive operations
4. **Event Streaming** - Real-time progress feedback

### Resource Management

```mermaid
sequenceDiagram
    participant D as Deleter
    participant AWS as AWS CloudFormation
    
    D->>AWS: DeleteStack()
    AWS->>D: DeleteStack initiated
    
    D->>AWS: WaitForStackOperation(callback)
    
    loop Until completion
        AWS->>D: StackEvent
        D->>D: Process and display event
        AWS->>D: Next StackEvent
    end
    
    AWS->>D: DELETE_COMPLETE
    D->>D: Cleanup and return
```

**Monitoring Benefits:**
- Real-time user feedback
- Early error detection
- Resource cleanup tracking
- Progress visibility

## Future Enhancements

### Phase 2 Considerations

1. **Advanced Safety Features**
   - Dry-run mode with preview-only operations
   - Force flag for automation scenarios
   - Batch confirmation for multiple stacks
   - Rollback protection for critical resources

2. **Enhanced User Experience**
   - Interactive stack selection
   - Deletion impact analysis
   - Resource inventory before deletion
   - Confirmation timeouts

3. **Operational Features**
   - Deletion scheduling
   - Notification integration
   - Audit log exports
   - Multi-region support

4. **Integration Enhancements**
   - CI/CD pipeline integration
   - External approval workflows
   - Resource retention policies
   - Cost impact analysis

### Extension Points

The architecture provides clear extension points:
- `Deleter` interface - Alternative deletion strategies
- Confirmation system - Custom approval workflows
- Progress monitoring - Enhanced feedback mechanisms
- Safety validations - Additional pre-deletion checks

### Architectural Principles for Extensions

1. **Safety First** - All enhancements must maintain or improve safety
2. **User Confirmation** - Explicit approval required for destructive operations
3. **Dependency Awareness** - Respect CloudFormation dependency constraints
4. **Error Transparency** - Clear communication of failures and remediation
5. **Operational Visibility** - Comprehensive logging and monitoring

This modular, safety-focused design ensures that stack deletion operations are reliable, transparent, and reversible through proper planning, while maintaining the flexibility to evolve with operational requirements.