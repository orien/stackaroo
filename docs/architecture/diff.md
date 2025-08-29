# Diff Module Architecture

## Overview

The diff module provides comprehensive comparison capabilities between deployed CloudFormation stacks and local configuration. It enables developers to preview changes before deployment, supporting template, parameter, and tag comparisons with multiple output formats.

## Architecture Diagram

```mermaid
graph TB
    subgraph "Command Layer"
        CLI[diff command<br/>cmd/diff.go]
    end
    
    subgraph "Core Diff Module"
        DI[Differ Interface<br/>types.go]
        DD[StackDiffer<br/>differ.go]
        
        subgraph "Comparators"
            TC[TemplateComparator<br/>template.go]
            PC[ParameterComparator<br/>comparators.go]
            TagC[TagComparator<br/>comparators.go]
        end
        
        subgraph "Output"
            OF[OutputFormatter<br/>output.go]
            RES[Result<br/>types.go]
        end
        
        subgraph "AWS Integration"
            AWS_OPS[AWS CloudFormation Operations<br/>internal/aws]
        end
    end
    
    subgraph "External Dependencies"
        AWS[AWS CloudFormation<br/>internal/aws]
        CFG[Configuration<br/>internal/config]
        RES_PKG[Stack Resolver<br/>internal/resolve]
    end
    
    CLI --> DI
    DI --> DD
    DD --> TC
    DD --> PC
    DD --> TagC
    DD --> AWS_OPS
    DD --> RES
    RES --> OF
    
    DD --> AWS
    CLI --> CFG
    CLI --> RES_PKG
    AWS_OPS --> AWS
```

## Component Architecture

### 1. Command Layer (`cmd/diff.go`)

**Responsibility:** CLI interface and user interaction

```mermaid
classDiagram
    class DiffCommand {
        +diffContextName: string
        +diffTemplateOnly: bool
        +diffParametersOnly: bool
        +diffTagsOnly: bool
        +diffFormat: string
        +RunE() error
        +diffWithConfig() error
    }
    
    class Differ {
        <<interface>>
        +DiffStack(ctx, stack, options) Result
    }
    
    DiffCommand --> Differ
```

**Key Features:**
- Flag validation and parsing
- Configuration resolution integration
- Options mapping to diff service
- Error handling and exit codes

### 2. Core Diff Engine (`internal/diff/`)

#### 2.1 Differ Interface and Implementation

```mermaid
classDiagram
    class Differ {
        <<interface>>
        +DiffStack(ctx, resolvedStack, options) Result
    }
    
    class StackDiffer {
        -cfClient: CloudFormationOperations
        -templateComparator: TemplateComparator
        -parameterComparator: ParameterComparator
        -tagComparator: TagComparator
        +DiffStack(ctx, resolvedStack, options) Result
        -handleNewStack() Result
        -compareTemplates() TemplateChange
        -compareParameters() []ParameterDiff
        -compareTags() []TagDiff
        -generateChangeSet() ChangeSetInfo
    }
    
    Differ <|-- StackDiffer
```

**Key Responsibilities:**
- Orchestrate comparison workflow
- Handle new vs. existing stack scenarios
- Integrate multiple comparison types
- Manage AWS changeset lifecycle

#### 2.2 Comparator Components

```mermaid
classDiagram
    class TemplateComparator {
        <<interface>>
        +Compare(current, proposed) TemplateChange
    }
    
    class ParameterComparator {
        <<interface>>
        +Compare(current, proposed) []ParameterDiff
    }
    
    class TagComparator {
        <<interface>>
        +Compare(current, proposed) []TagDiff
    }
    
    class YAMLTemplateComparator {
        +Compare(current, proposed) TemplateChange
        -calculateHash(template) string
        -compareResources() ResourceCounts
        -generateDiff() string
        -generateResourceDiff() string
    }
    
    class DefaultParameterComparator {
        +Compare(current, proposed) []ParameterDiff
    }
    
    class DefaultTagComparator {
        +Compare(current, proposed) []TagDiff
    }
    
    TemplateComparator <|-- YAMLTemplateComparator
    ParameterComparator <|-- DefaultParameterComparator
    TagComparator <|-- DefaultTagComparator
```

### 3. Data Models

```mermaid
classDiagram
    class Result {
        +StackName: string
        +Environment: string
        +StackExists: bool
        +TemplateChange: TemplateChange
        +ParameterDiffs: []ParameterDiff
        +TagDiffs: []TagDiff
        +ChangeSet: ChangeSetInfo
        +Options: Options
        +HasChanges() bool
        +String() string
        +toText() string
        +toJSON() string
    }
    
    class TemplateChange {
        +HasChanges: bool
        +CurrentHash: string
        +ProposedHash: string
        +Diff: string
        +ResourceCount: ResourceCounts
    }
    
    class ParameterDiff {
        +Key: string
        +CurrentValue: string
        +ProposedValue: string
        +ChangeType: ChangeType
    }
    
    class TagDiff {
        +Key: string
        +CurrentValue: string
        +ProposedValue: string
        +ChangeType: ChangeType
    }
    
    class Options {
        +TemplateOnly: bool
        +ParametersOnly: bool
        +TagsOnly: bool
        +Format: string
    }
    
    Result --> TemplateChange
    Result --> ParameterDiff
    Result --> TagDiff
    Result --> Options
```

## Data Flow Architecture

```mermaid
sequenceDiagram
    participant CLI as CLI Command
    participant Resolver as Stack Resolver
    participant Differ as Default Differ
    participant AWS as AWS Client
    participant Comparators as Comparators
    participant Output as Output Formatter
    
    CLI->>Resolver: Resolve stack configuration
    Resolver->>CLI: ResolvedStack
    
    CLI->>Differ: DiffStack(resolved, options)
    
    Differ->>AWS: StackExists(stackName)
    AWS->>Differ: bool
    
    alt Stack Exists
        Differ->>AWS: DescribeStack(stackName)
        AWS->>Differ: StackInfo (with template)
        
        Differ->>Comparators: Compare templates
        Comparators->>Differ: TemplateChange
        
        Differ->>Comparators: Compare parameters
        Comparators->>Differ: []ParameterDiff
        
        Differ->>Comparators: Compare tags
        Comparators->>Differ: []TagDiff
        
        alt Has Changes && Full Diff
            Differ->>AWS: CreateChangeSet()
            AWS->>Differ: ChangeSetInfo
            Differ->>AWS: DeleteChangeSet()
        end
    else Stack Doesn't Exist
        Differ->>Differ: handleNewStack()
    end
    
    Differ->>CLI: Result
    CLI->>Output: Format result
    Output->>CLI: Formatted string
```

## Integration Points

### 1. AWS Integration (`internal/aws`)

**Extended Interfaces:**
- `StackExists()` - Check stack existence
- `DescribeStack()` - Get detailed stack information including template
- `GetTemplate()` - Retrieve deployed template content
- `CreateChangeSet()` - Generate change preview
- `DeleteChangeSet()` - Clean up temporary changesets
- `DescribeChangeSet()` - Get changeset details

### 2. Configuration Integration (`internal/config`)

**Dependencies:**
- `ConfigProvider` - Load environment-specific configuration
- Stack resolution and parameter inheritance
- Template path resolution

### 3. Stack Resolution (`internal/resolve`)

**Enhanced Integration:**
- `ResolvedStack.GetTemplateContent()` - Access resolved template
- `ResolvedStacks.Context` - Track deployment context
- Dependency resolution for complete stack information

### 4. Deployment Integration (`internal/deploy`)

**Integrated Preview:**
- `NewDiffer(cfClient)` - Create differ with existing CloudFormation operations
- `DiffStack(ctx, resolvedStack, options)` - Generate change preview during deployment
- Consistent formatting between `stackaroo diff` and `stackaroo deploy` commands
- Automatic change preview before deployment execution for existing stacks
- Same changeset-based approach for both standalone diff and integrated deployment preview

**Deployment Flow Integration:**
```mermaid
sequenceDiagram
    participant Deploy as Deploy Command  
    participant Differ as Diff Engine
    participant AWS as AWS CloudFormation
    
    Deploy->>Differ: DiffStack(resolved, options)
    Differ->>AWS: Generate changeset & preview
    AWS->>Differ: Change details
    Differ->>Deploy: Formatted preview
    Deploy->>User: Display changes
    Deploy->>AWS: Execute deployment
```

## Error Handling Strategy

```mermaid
graph TD
    A[Diff Request] --> B{Stack Exists?}
    B -->|No| C[Handle New Stack]
    B -->|Yes| D{Get Stack Info}
    
    D -->|Success| E[Compare Components]
    D -->|Error| F[AWS Error Handling]
    
    E --> G{Has Changes?}
    G -->|Yes| H{Create ChangeSet?}
    G -->|No| I[Return No Changes]
    
    H -->|Success| J[Include ChangeSet Info]
    H -->|Error| K[Log Warning, Continue]
    
    F --> L[Return Error with Context]
    C --> M[Show New Stack Preview]
    J --> N[Return Complete Result]
    K --> N
    I --> N
    M --> N
    
    N --> O[Format Output]
```

**Error Categories:**
1. **Configuration Errors** - Invalid stack names, missing environments
2. **AWS Errors** - Credentials, permissions, API failures
3. **Template Errors** - Invalid YAML, parsing failures
4. **Changeset Errors** - Non-blocking warnings for preview failures

## Output Architecture

### Text Output Format
```
Stack: vpc (Environment: dev)
==================================================

Status: CHANGES DETECTED

Template Changes:
-----------------
✓ Template has been modified
Resource changes:
  + 2 resources to be added
  ~ 1 resources to be modified

Parameter Changes:
------------------
  + NewParam: value123
  ~ ExistingParam: oldvalue → newvalue
  - RemovedParam: oldvalue

Tag Changes:
------------
  + Environment: dev
  ~ Owner: oldteam → newteam

AWS CloudFormation Preview:
---------------------------
ChangeSet ID: arn:aws:cloudformation:...
Status: CREATE_COMPLETE

Resource Changes:
  + MyBucket (AWS::S3::Bucket)
  ~ MyRole (AWS::IAM::Role) - Replacement: False
    Property: PolicyDocument
```

### JSON Output Format
```json
{
  "stackName": "vpc",
  "environment": "dev",
  "stackExists": true,
  "hasChanges": true,
  "templateChanges": {
    "hasChanges": true,
    "resourceCount": {"added": 2, "modified": 1, "removed": 0}
  },
  "parameterDiffs": [
    {"key": "NewParam", "changeType": "ADD", "proposedValue": "value123"}
  ],
  "tagDiffs": [
    {"key": "Environment", "changeType": "ADD", "proposedValue": "dev"}
  ],
  "changeSet": {
    "changeSetId": "arn:aws:cloudformation:...",
    "status": "CREATE_COMPLETE",
    "changes": [...]
  }
}
```

## Testing Architecture

### Test Categories

```mermaid
graph LR
    subgraph "Unit Tests"
        A[Comparator Tests]
        B[Output Format Tests]
        C[Command Structure Tests]
    end
    
    subgraph "Integration Tests"
        D[AWS Client Tests]
        E[End-to-End Tests]
    end
    
    subgraph "Mock Infrastructure"
        F[Mock Differ]
        G[Mock AWS Client]
        H[Mock Comparators]
    end
    
    A --> F
    B --> F
    C --> F
    D --> G
    E --> H
```

**Test Coverage:**
- Parameter comparison: 7 test scenarios
- Tag comparison: 6 test scenarios  
- Template comparison: Basic semantic testing
- Command validation: Flag and argument testing
- Error scenarios: AWS failures, invalid input
- Output formatting: Text and JSON validation

## Performance Considerations

### Optimisation Strategies

1. **Template Hashing** - Quick change detection via SHA256
2. **Lazy ChangeSet Creation** - Only when changes detected and full diff requested
3. **Parallel Comparisons** - Template, parameter, and tag comparisons can run concurrently
4. **ChangeSet Cleanup** - Immediate cleanup to avoid AWS resource accumulation

### Resource Management

```mermaid
sequenceDiagram
    participant D as Differ
    participant CS as ChangeSet Manager
    participant AWS as AWS CloudFormation
    
    D->>CS: CreateChangeSet()
    CS->>AWS: CreateChangeSet()
    AWS->>CS: ChangeSetID
    
    CS->>AWS: WaitForChangeSet()
    AWS->>CS: Status: CREATE_COMPLETE
    
    CS->>AWS: DescribeChangeSet()
    AWS->>CS: ChangeSet Details
    
    CS->>D: ChangeSetInfo
    
    Note over CS,AWS: Immediate cleanup
    CS->>AWS: DeleteChangeSet()
    AWS->>CS: Deleted
```

## Security Considerations

1. **Credential Management** - Uses AWS SDK default credential chain
2. **Permissions** - Requires minimal CloudFormation read permissions:
   - `cloudformation:DescribeStacks`
   - `cloudformation:GetTemplate`
   - `cloudformation:CreateChangeSet`
   - `cloudformation:DeleteChangeSet`
   - `cloudformation:DescribeChangeSet`
3. **Resource Cleanup** - Ensures temporary changesets are always cleaned up
4. **Error Information** - Avoids exposing sensitive data in error messages

## Future Enhancements

### Phase 2 Considerations

1. **Enhanced Template Diffing**
   - Line-by-line YAML comparison
   - Syntax highlighting for differences
   - Resource property-level changes

2. **Advanced ChangeSet Analysis**
   - Resource replacement impact analysis
   - Dependency impact assessment
   - Cost estimation integration

3. **Performance Optimisation**
   - Caching of frequently accessed stacks
   - Parallel processing of multiple stacks
   - Incremental diff capabilities

4. **Output Enhancements**
   - HTML output format
   - Interactive diff viewing
   - Integration with external diff tools

### Extension Points

The architecture provides clear extension points through interfaces:
- `TemplateComparator` - Custom template comparison algorithms
- `CloudFormationOperations` - Alternative AWS integration strategies
- `OutputFormatter` - Additional output formats
- `Differ` - Alternative diff engines (e.g., client-side only)

This modular design ensures the diff functionality can evolve while maintaining backward compatibility and clear separation of concerns.