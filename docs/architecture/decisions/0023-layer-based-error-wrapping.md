# 23. Layer-based error wrapping strategy

Date: 2026/01/18

## Status

Accepted

## Context

As the Stackaroo codebase grew, error handling followed a consistent pattern of wrapping errors at every layer using `fmt.Errorf("context: %w", err)`. While technically correct (maintaining error chains with `%w`), this approach created verbose, redundant error messages that degraded user experience.

### Problem: Error Message Duplication

Users encountered error messages with repeated information:

```
Error: failed to resolve stack dependencies: failed to get stack vpc: failed to resolve stack 'vpc' for context 'dev': stack 'vpc' not found in configuration
```

This message contains:
- "resolve" appears 3 times
- "stack vpc" appears 3 times  
- "failed to" appears 4 times
- Semantic duplication ("resolve dependencies" vs "get stack" vs "resolve stack")

The actual actionable information is at the end: `stack 'vpc' not found in configuration for context 'dev'`.

### Root Cause Analysis

The codebase has a layered architecture:

1. **Entry Point Layer** (`cmd/*`) - CLI commands
2. **Orchestration Layer** (`internal/deploy`, `internal/delete`, `internal/diff`, etc.) - Coordinates operations
3. **Business Logic Layer** (`internal/resolve`) - Domain logic and transformations
4. **Boundary Layer** (`internal/aws`, `internal/config`, `internal/prompt`) - External system interfaces

Each layer was wrapping errors with generic context ("failed to resolve", "failed to get", "failed to load"), creating redundancy when errors propagated up the stack.

### Design Principles Violated

1. **Don't Repeat Yourself (DRY)**: Same information (stack name, operation type) appeared at multiple layers
2. **Clarity over Completeness**: Technical completeness (full error chain) degraded user clarity
3. **Signal-to-Noise Ratio**: Valuable context (root cause) was buried in redundant wrappers

## Decision

Implement a **layer-based error wrapping strategy** with clear rules for when each layer should wrap errors:

### Rule 1: Boundary Layers ALWAYS Wrap

**Applies to:** `internal/aws/*`, `internal/config/file/*`, `internal/prompt/*`

**Responsibility:** Convert external errors (AWS SDK, file I/O, YAML parsing) into domain-specific errors with context.

**Rationale:** External errors lack domain knowledge. This is the only layer that should add domain context.

**Pattern:**
```go
func (cf *DefaultCloudFormationOperations) GetStack(ctx context.Context, stackName string) (*Stack, error) {
    result, err := cf.client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
        StackName: aws.String(stackName),
    })
    if err != nil {
        // ✅ Wrap AWS SDK error with domain context
        return nil, fmt.Errorf("failed to describe stack %s: %w", stackName, err)
    }
    // ...
}
```

### Rule 2: Orchestration Layers NEVER Wrap

**Applies to:** `internal/deploy/*`, `internal/delete/*`, `internal/diff/*`, `internal/describe/*`, `internal/validate/*`

**Responsibility:** Coordinate operations but pass through errors unchanged.

**Rationale:** These layers don't add new information. Lower layers already provide complete context.

**Pattern:**
```go
func (d *StackDeployer) DeploySingleStack(ctx context.Context, stackName, contextName string) error {
    stack, err := d.resolver.ResolveStack(ctx, contextName, stackName)
    if err != nil {
        return err  // ✅ Pass through - resolver has full context
    }
    return d.deployStackWithFeedback(ctx, stack, contextName)
}
```

### Rule 3: Entry Point Layers NEVER Wrap

**Applies to:** `cmd/*`

**Responsibility:** Return errors directly to the CLI framework for user presentation.

**Rationale:** All context is already present. The command layer just presents errors to users.

**Pattern:**
```go
func describeSingleStack(ctx context.Context, stackName, contextName, configFile string) error {
    stack, err := resolver.ResolveStack(ctx, contextName, stackName)
    if err != nil {
        return err  // ✅ Pass through to CLI
    }
    // ...
}
```

### Rule 4: Business Logic Layers CONDITIONALLY Wrap

**Applies to:** `internal/resolve/*`

**Responsibility:** Wrap ONLY when adding genuinely new semantic context.

**Rationale:** These layers transform data. Wrap only when the operation changes meaning.

**Decision Tree:**
```
Is this error from an internal layer?
├─ YES → Does wrapping add NEW semantic meaning?
│        ├─ YES → Wrap with specific context
│        └─ NO  → Pass through
└─ NO  → Wrap (shouldn't happen - use boundary layer)
```

**Example - Pass Through:**
```go
func (r *StackResolver) ResolveStack(ctx context.Context, context string, stackName string) (*model.Stack, error) {
    cfg, err := r.configProvider.LoadConfig(ctx, context)
    if err != nil {
        // Config provider already has "context 'dev' not found in configuration"
        // Don't wrap with generic "failed to load config"
        return nil, err  // ✅ Pass through
    }
    // ...
}
```

**Example - Add Semantic Value:**
```go
func (r *StackResolver) ResolveStack(ctx context.Context, context string, stackName string) (*model.Stack, error) {
    parameters, err := r.resolveParameters(ctx, stackConfig.Parameters, cfg.Context.Region)
    if err != nil {
        // ✅ "Parameter resolution" is semantically different from underlying operations
        // (which might be stack output fetches, file reads, etc.)
        return nil, fmt.Errorf("failed to resolve parameters for stack %s: %w", stackName, err)
    }
    // ...
}
```

### Exception: Domain Error Type Conversion

Orchestration layers may wrap when converting between domain error types:

```go
func (d *StackDeployer) deployWithChangeSet(ctx context.Context, stack *model.Stack, cfnOps aws.CloudFormationOperations) error {
    // ...
    var noChangesErr aws.NoChangesError
    if errors.As(diffResult.ChangeSetError, &noChangesErr) {
        // ✅ Converting AWS domain error to deploy domain error
        return NoChangesError{StackName: stack.Name}
    }
}
```

## Consequences

### Positive Consequences

**1. Dramatically improved user experience:**

Users see clear, actionable error messages:

```
Before: failed to resolve stack dependencies: failed to get stack vpc: failed to resolve stack 'vpc' for context 'dev': stack 'vpc' not found in configuration

After:  failed to resolve stack 'vpc' for context 'dev': stack 'vpc' not found in configuration
```

Or even better with config layer improvements:
```
After:  stack 'vpc' not found in configuration for context 'dev'
```

**2. Clear architectural boundaries:**

Each layer has well-defined error handling responsibilities. New developers can easily understand when to wrap errors.

**3. Maintainability:**

Simple decision tree: "Am I at a system boundary? Yes = wrap. No = pass through (unless adding semantic value)."

**4. Debugging support maintained:**

Error chains are preserved using `%w`, so `errors.As()` and `errors.Is()` still work for programmatic error handling.

**5. Reduced cognitive load:**

Developers don't need to decide "should I wrap this?" at every error site. The layer determines the answer.

### Negative Consequences

**1. Migration effort:**

Existing code must be updated to follow new rules:
- Remove wrapping in `cmd/*`
- Remove wrapping in orchestration layers
- Review business logic layer wrapping

**2. Requires developer education:**

Team members must understand and follow layer-based rules. New contributors need clear guidelines.

**3. Potential for inconsistency during transition:**

Mixed old/new patterns may exist temporarily during migration.

**4. Less verbose error chains:**

Some developers prefer seeing every layer in error messages for debugging. This approach prioritises user experience over exhaustive technical detail.

## Related Decisions

- [ADR 0011: Testing framework and strategy](0011-testing-framework-and-strategy.md) - Error testing approach
- [ADR 0008: Configuration abstraction](0008-configuration-abstraction.md) - Configuration layer boundaries

## References

- [Go blog: Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors)
- [Go blog: Error handling and Go](https://go.dev/blog/error-handling-and-go)
- [Dave Cheney: Don't just check errors, handle them gracefully](https://dave.cheney.net/2016/04/27/dont-just-check-errors-handle-them-gracefully)