# 17. Parameter resolution system enhancement

Date: 2025-09-12

## Status

Accepted

Amends [ADR 0010: File provider configuration structure](0010-file-provider-configuration-structure.md)

Amended by [ADR 0022: Simplified stack output reference field names](0022-simplified-stack-output-field-names.md)

## Context

The original configuration system implemented in ADR 0010 established a parameter structure using simple `map[string]string` values, which worked well for basic CloudFormation parameters but had significant limitations:

**Original Parameter Limitations:**
- Only supported literal string values in YAML configuration
- No support for CloudFormation `List<Type>` or `CommaDelimitedList` parameters
- No dynamic parameter resolution (e.g., referencing stack outputs)
- No cross-stack dependency resolution for parameter values
- Limited to static configuration at deploy time

**CloudFormation List Parameter Requirements:**
CloudFormation supports sophisticated list parameters that accept multiple values:
- `List<AWS::EC2::VPC::Id>` - Multiple VPC IDs
- `List<AWS::EC2::SecurityGroup::Id>` - Multiple Security Group IDs
- `CommaDelimitedList` - Generic comma-separated string lists

These parameters require comma-separated string values (e.g., `"sg-123,sg-456,sg-789"`) but teams needed the ability to compose these from:
- Hardcoded literal values (`sg-baseline123`)
- Dynamic stack outputs (`{stack: vpc-stack, output: WebSGId}`)
- Mixed combinations of both within a single parameter

**Real-World Use Case:**
```yaml
# Needed capability: Security groups from multiple sources
SecurityGroupIds:
  - sg-company-baseline      # Literal: corporate standard
  - {from: vpc-stack.WebSGId}     # Dynamic: application-specific
  - {from: monitoring-stack.DatadogSGId}  # Dynamic: monitoring
```

**Alternative Approaches Considered:**

1. **String interpolation approach**: `SecurityGroupIds: "sg-123,#{vpc-stack.WebSGId},sg-456"`
   - Complex parsing requirements
   - Poor error handling for missing references
   - Difficult to validate individual components

2. **Explicit list wrapper syntax**:
   ```yaml
   SecurityGroupIds:
     type: list
     items: [...]
   ```
   - Verbose and non-intuitive
   - Breaks YAML readability
   - Inconsistent with modern IaC tool patterns

3. **Function-based syntax**: `SecurityGroupIds: !Concat [!Ref BaselineSG, !StackOutput vpc-stack.WebSGId]`
   - CloudFormation-like complexity in configuration
   - Poor developer experience
   - Difficult to extend

The selected approach needed to:
- Support CloudFormation list parameter types naturally
- Allow heterogeneous lists (mix literals and dynamic values)
- Maintain clean, readable YAML syntax
- Preserve backward compatibility
- Enable future resolver extensions (SSM, Secrets Manager, etc.)

## Decision

We will implement a **sophisticated parameter resolution system** with clean YAML array syntax for list parameters.

### Core Architecture Changes

**1. Enhanced Parameter Value Structure:**
```go
// Before: Simple string map
type StackConfig struct {
    Parameters map[string]string  // Limited to literals only
}

// After: Rich resolution model
type StackConfig struct {
    Parameters map[string]*ParameterValue  // Supports multiple resolution types
}

type ParameterValue struct {
    ResolutionType   string            // "literal", "stack-output", "list"
    ResolutionConfig map[string]string // Type-specific configuration
    ListItems        []*ParameterValue // For list parameters
}
```

**2. Clean YAML Array Syntax:**
```yaml
# List parameters use intuitive YAML arrays
SecurityGroupIds:
  - sg-baseline123           # Literal value
  - type: stack-output       # Dynamic resolution
    stack_name: security-stack
    output_key: WebSGId
  - sg-additional456         # Another literal

# Simple literal lists
AllowedPorts:
  - "80"
  - "443"
  - "8080"
```

**3. Resolution Type System:**
- **`literal`**: Direct string values
- **`stack-output`**: References to CloudFormation stack outputs
- **`list`**: Arrays of mixed resolution types
- **Extensible**: Future types (SSM, Secrets Manager) plug in seamlessly

**4. Heterogeneous List Support:**
Each list item can be resolved using different mechanisms:
```yaml
DatabaseConfig:
  - "production"                    # Literal environment name
  - type: stack-output             # Dynamic database endpoint
    stack_name: rds-stack
    output_key: DatabaseEndpoint
  - type: stack-output             # Dynamic connection string
    stack_name: secrets-stack
    output_key: ConnectionString
```

**5. Backward Compatibility:**
Existing configurations continue to work unchanged:
```yaml
# Still supported: simple literal parameters
Environment: production
InstanceType: t3.micro
```

### Implementation Details

**YAML Processing Layer (`yamlParameterValue`):**
- Automatic detection of YAML node types (scalar, mapping, sequence)
- Seamless conversion between YAML arrays and list parameters
- Support for mixed literal/resolver arrays

**Configuration Layer (`config.ParameterValue`):**
- Unified resolution model across all parameter types
- Recursive list support (lists within lists)
- Clear separation between resolution type and configuration

**Resolution Engine (`StackResolver`):**
- Single `resolveSingleParameter()` method handles all types
- Comma-separated output for CloudFormation list parameters
- Empty value filtering and error handling

### CloudFormation Integration

**List Parameter Resolution:**
```yaml
# YAML Configuration
SecurityGroupIds:
  - sg-123
  - type: stack-output
    stack_name: vpc-stack
    output_key: WebSGId  # Resolves to "sg-456"
  - sg-789

# Resolves to CloudFormation Parameter
SecurityGroupIds: "sg-123,sg-456,sg-789"
```

**CloudFormation Template Compatibility:**
```yaml
Parameters:
  SecurityGroupIds:
    Type: List<AWS::EC2::SecurityGroup::Id>
    Description: List of security group IDs

Resources:
  LaunchTemplate:
    Type: AWS::EC2::LaunchTemplate
    Properties:
      LaunchTemplateData:
        SecurityGroupIds: !Ref SecurityGroupIds  # Direct reference
```

## Consequences

### Positive Consequences

**1. CloudFormation List Parameter Support:**
- Full support for `List<Type>` and `CommaDelimitedList` parameters
- Natural integration with existing CloudFormation templates
- Proper comma-separated value generation

**2. Enhanced Developer Experience:**
- Intuitive YAML array syntax matches developer expectations
- Mix static and dynamic values in single parameter
- Clear error messages for resolution failures
- Easy migration from simple parameters to lists

**3. Architectural Flexibility:**
- Extensible resolver architecture for future parameter sources
- Clean separation between configuration format and resolution logic
- Support for complex deployment scenarios (multi-account, cross-stack)

**4. Backward Compatibility:**
- Existing configurations work without changes
- Gradual adoption possible (convert parameters incrementally)
- No breaking changes to CLI interface or existing workflows

**5. Enterprise-Ready Features:**
- Cross-stack parameter dependencies
- Context-specific parameter overrides for lists
- Support for complex multi-tier application architectures

### Negative Consequences

**1. Increased Complexity:**
- More sophisticated configuration structure
- Learning curve for teams unfamiliar with resolver concepts
- Additional validation and error handling requirements

**2. Performance Considerations:**
- Multiple CloudFormation API calls for stack output resolution
- Increased memory usage for complex parameter structures
- Dependency resolution overhead

**3. Debugging Complexity:**
- Parameter resolution errors can be nested and complex
- Need for better tooling to trace resolution paths
- Potential confusion between YAML structure and resolved values

This enhancement maintains Stackaroo's principle of clean, intuitive configuration while significantly expanding its capabilities to handle enterprise-scale CloudFormation deployments with complex parameter requirements.
