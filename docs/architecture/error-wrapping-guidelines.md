# Error Wrapping Guidelines

## Overview

This document establishes clear rules for when and where to wrap errors in Stackaroo to prevent message duplication while maintaining useful error context.

## Core Principle

**Only wrap errors when adding NEW, VALUABLE context that isn't already present in the wrapped error.**

## Layer-Based Rules

### Layer Classification

Stackaroo's architecture has four distinct layer types, each with different error wrapping responsibilities:

1. **Boundary Layers** - Interface with external systems (AWS, filesystem, user input)
2. **Business Logic Layers** - Implement domain logic and transformations
3. **Orchestration Layers** - Coordinate between other layers
4. **Entry Point Layers** - CLI commands that present errors to users

---

## Rule 1: Boundary Layers ALWAYS Wrap

**Layers**: `internal/aws/*`, `internal/config/file/*`, `internal/prompt/*`

**Responsibility**: Convert external errors into domain-specific errors with context.

**Why**: External errors (AWS SDK, OS, YAML parser) lack domain context. This is the ONLY layer that should add it.

**Pattern**:
```go
// ✅ CORRECT - Boundary layer wrapping external error
func (cf *DefaultCloudFormationOperations) GetStack(ctx context.Context, stackName string) (*Stack, error) {
    result, err := cf.client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
        StackName: aws.String(stackName),
    })
    if err != nil {
        // Wrap AWS SDK error with domain context
        return nil, fmt.Errorf("failed to describe stack %s: %w", stackName, err)
    }
    // ... rest of implementation
}

// ✅ CORRECT - Boundary layer wrapping filesystem error
func (fp *FileConfigProvider) ensureLoaded() error {
    data, err := os.ReadFile(fp.filename)
    if err != nil {
        // Wrap OS error with domain context
        return fmt.Errorf("failed to read config file '%s': %w", fp.filename, err)
    }
    // ... rest of implementation
}
```

**What to Include**:
- Entity names (stack name, file path, region)
- Operation type (describe, read, validate)
- Domain-relevant context

---

## Rule 2: Business Logic Layers CONDITIONALLY Wrap

**Layers**: `internal/resolve/*`, `internal/diff/template.go`

**Responsibility**: Add context ONLY when the operation changes semantic meaning.

**Why**: These layers transform or interpret data. Wrap only when adding genuinely new information.

**Decision Tree**:
```
Is the wrapped error from another internal layer?
  ├─ YES → Does wrapping add NEW semantic context?
  │         ├─ YES → WRAP with specific context
  │         └─ NO  → PASS THROUGH (don't wrap)
  └─ NO (external) → WRAP (shouldn't happen - use boundary layer)
```

**Examples**:

```go
// ✅ CORRECT - Pass through when no new context
func (r *StackResolver) ResolveStack(ctx context.Context, context string, stackName string) (*model.Stack, error) {
    cfg, err := r.configProvider.LoadConfig(ctx, context)
    if err != nil {
        // Config provider already has full context ("context 'dev' not found")
        // Don't wrap with generic "failed to load config"
        return nil, err  // PASS THROUGH
    }

    stackConfig, err := r.configProvider.GetStack(stackName, context)
    if err != nil {
        // Config provider already has "stack 'vpc' not found for context 'dev'"
        // Don't wrap with generic "failed to get stack"
        return nil, err  // PASS THROUGH
    }
    
    // ... more implementation
}

// ✅ CORRECT - Wrap when adding NEW semantic context
func (r *StackResolver) ResolveStack(ctx context.Context, context string, stackName string) (*model.Stack, error) {
    // ... earlier code ...
    
    parameters, err := r.resolveParameters(ctx, stackConfig.Parameters, cfg.Context.Region)
    if err != nil {
        // This adds NEW context: we're specifically resolving PARAMETERS
        // The wrapped error might be about stack outputs, file reads, etc.
        // "parameter resolution" is semantically different from underlying operation
        return nil, fmt.Errorf("failed to resolve parameters for stack %s: %w", stackName, err)
    }
    
    // ... rest of implementation
}

// ❌ WRONG - Wrapping without adding value
func (r *StackResolver) ResolveStack(ctx context.Context, context string, stackName string) (*model.Stack, error) {
    cfg, err := r.configProvider.LoadConfig(ctx, context)
    if err != nil {
        // This just adds generic "failed to load config" 
        // which doesn't add value over config's "context 'dev' not found"
        return nil, fmt.Errorf("failed to load config: %w", err)  // ❌ DON'T DO THIS
    }
}
```

**Guidelines**:
- **DO wrap** when you're adding semantic context (e.g., "parameter resolution failed" when underlying error is about AWS stack outputs)
- **DON'T wrap** when you're just passing through (e.g., "failed to get stack" wrapping "stack not found")
- **DO include** entity names if they're not in the wrapped error
- **DON'T include** entity names if they're already in the wrapped error

---

## Rule 3: Orchestration Layers NEVER Wrap

**Layers**: `internal/deploy/*`, `internal/delete/*`, `internal/diff/*`, `internal/describe/*`, `internal/validate/*`

**Responsibility**: Coordinate operations but DON'T add error context.

**Why**: These layers just orchestrate calls to business logic and boundary layers. Those layers already provide full context.

**Pattern**:
```go
// ✅ CORRECT - Orchestration layer passes through
func (d *StackDeployer) DeploySingleStack(ctx context.Context, stackName, contextName string) error {
    stack, err := d.resolver.ResolveStack(ctx, contextName, stackName)
    if err != nil {
        return err  // PASS THROUGH - resolver has full context
    }

    return d.deployStackWithFeedback(ctx, stack, contextName)
}

// ✅ CORRECT - Orchestration layer passes through
func (d *StackDeleter) DeleteSingleStack(ctx context.Context, stackName, contextName string) error {
    stack, err := d.resolver.ResolveStack(ctx, contextName, stackName)
    if err != nil {
        return err  // PASS THROUGH
    }

    return d.deleteStackWithFeedback(ctx, stack, contextName)
}

// ❌ WRONG - Orchestration layer wrapping unnecessarily
func (d *StackDeployer) DeploySingleStack(ctx context.Context, stackName, contextName string) error {
    stack, err := d.resolver.ResolveStack(ctx, contextName, stackName)
    if err != nil {
        // This adds "failed to resolve stack dependencies" which is redundant
        // Resolver already says "failed to resolve stack 'vpc' for context 'dev'"
        return fmt.Errorf("failed to resolve stack dependencies: %w", err)  // ❌ DON'T
    }
}
```

**Exception**: Wrap ONLY when converting errors to domain-specific error types:

```go
// ✅ ACCEPTABLE - Converting to domain error type
func (d *StackDeployer) deployWithChangeSet(ctx context.Context, stack *model.Stack, cfnOps aws.CloudFormationOperations) error {
    diffResult, err := d.differ.CalculateChanges(ctx, stack)
    if err != nil {
        return fmt.Errorf("failed to calculate changes: %w", err)
    }

    // Check for special error type
    var noChangesErr aws.NoChangesError
    if errors.As(diffResult.ChangeSetError, &noChangesErr) {
        // Convert AWS domain error to deploy domain error
        return NoChangesError{StackName: stack.Name}  // ✅ This adds value
    }
}
```

---

## Rule 4: Entry Point Layers NEVER Wrap

**Layers**: `cmd/*`

**Responsibility**: Return errors directly to the CLI framework.

**Why**: All context is already present from lower layers. The cmd layer just presents errors to users.

**Pattern**:
```go
// ✅ CORRECT - Entry point passes through
func describeSingleStack(ctx context.Context, stackName, contextName, configFile string) error {
    _, resolver := createResolver(configFile)

    stack, err := resolver.ResolveStack(ctx, contextName, stackName)
    if err != nil {
        return err  // PASS THROUGH
    }

    d := getDescriber()
    stackDesc, err := d.DescribeStack(ctx, stack)
    if err != nil {
        return err  // PASS THROUGH
    }

    fmt.Print(describe.FormatStackDescription(stackDesc))
    return nil
}

// ❌ WRONG - Entry point wrapping
func describeSingleStack(ctx context.Context, stackName, contextName, configFile string) error {
    _, resolver := createResolver(configFile)

    stack, err := resolver.ResolveStack(ctx, contextName, stackName)
    if err != nil {
        // Resolver already has "failed to resolve stack 'vpc'"
        // This adds redundant "failed to resolve stack vpc"
        return fmt.Errorf("failed to resolve stack %s: %w", stackName, err)  // ❌ DON'T
    }
}
```

---

## Quick Reference Table

| Layer Type | Examples | Wrap External Errors? | Wrap Internal Errors? |
|------------|----------|----------------------|----------------------|
| **Boundary** | `internal/aws/*`<br>`internal/config/file/*`<br>`internal/prompt/*` | ✅ ALWAYS | N/A (shouldn't call internal layers) |
| **Business Logic** | `internal/resolve/*`<br>`internal/diff/template.go` | ✅ ALWAYS | ⚠️ ONLY if adding semantic value |
| **Orchestration** | `internal/deploy/*`<br>`internal/delete/*`<br>`internal/diff/*`<br>`internal/describe/*`<br>`internal/validate/*` | ✅ ALWAYS | ❌ NEVER (pass through) |
| **Entry Point** | `cmd/*` | N/A (shouldn't call external) | ❌ NEVER (pass through) |

---

## Checklist Before Wrapping

Before adding `fmt.Errorf("context: %w", err)`, ask:

1. ✅ **Am I in a boundary layer calling an external system?**
   - YES → Wrap with domain context
   - NO → Continue to question 2

2. ✅ **Does the wrapped error already contain the information I want to add?**
   - YES → Don't wrap, pass through
   - NO → Continue to question 3

3. ✅ **Am I adding NEW semantic meaning (not just rephrasing)?**
   - YES → Wrap with specific context
   - NO → Don't wrap, pass through

4. ✅ **Am I in an orchestration or entry point layer?**
   - YES → Don't wrap, pass through
   - NO → You can wrap if questions 1-3 say yes

---

## Common Patterns

### ✅ Good: Boundary Layer Wrapping

```go
// internal/aws/cloudformation.go
func (cf *DefaultCloudFormationOperations) ValidateTemplate(ctx context.Context, templateBody string) error {
    _, err := cf.client.ValidateTemplate(ctx, &cloudformation.ValidateTemplateInput{
        TemplateBody: aws.String(templateBody),
    })
    if err != nil {
        // ✅ Wrapping AWS SDK error with operation context
        return fmt.Errorf("template validation failed: %w", err)
    }
    return nil
}
```

### ✅ Good: Pass Through in Orchestration Layer

```go
// internal/deploy/deployer.go
func (d *StackDeployer) DeploySingleStack(ctx context.Context, stackName, contextName string) error {
    stack, err := d.resolver.ResolveStack(ctx, contextName, stackName)
    if err != nil {
        return err  // ✅ Resolver already has context
    }
    return d.deployStackWithFeedback(ctx, stack, contextName)
}
```

### ✅ Good: Semantic Context in Business Logic

```go
// internal/resolve/stack_resolver.go
func (r *StackResolver) resolveParameters(ctx context.Context, params map[string]*config.ParameterValue, region string) (map[string]string, error) {
    for key, paramValue := range params {
        value, err := r.resolveSingleParameter(ctx, paramValue, region)
        if err != nil {
            // ✅ Adding semantic context: parameter resolution (vs stack output fetch, file read, etc.)
            return nil, fmt.Errorf("failed to resolve parameter '%s': %w", key, err)
        }
        resolved[key] = value
    }
}
```

### ❌ Bad: Redundant Wrapping

```go
// internal/deploy/deployer.go
func (d *StackDeployer) DeploySingleStack(ctx context.Context, stackName, contextName string) error {
    stack, err := d.resolver.ResolveStack(ctx, contextName, stackName)
    if err != nil {
        // ❌ Resolver already says "failed to resolve stack 'vpc'"
        // This adds redundant "failed to resolve stack dependencies"
        return fmt.Errorf("failed to resolve stack dependencies: %w", err)
    }
}
```

### ❌ Bad: Duplicate Stack Names

```go
// internal/describe/describer.go
func (d *StackDescriber) DescribeStack(ctx context.Context, stack *model.Stack) (*StackDescription, error) {
    stackInfo, err := cfOps.DescribeStack(ctx, stack.Name)
    if err != nil {
        // ❌ AWS layer already says "failed to describe stack vpc"
        // This creates: "failed to describe stack vpc: failed to describe stack vpc: ..."
        return nil, fmt.Errorf("failed to describe stack %s: %w", stack.Name, err)
    }
}

// Fix: Pass through
func (d *StackDescriber) DescribeStack(ctx context.Context, stack *model.Stack) (*StackDescription, error) {
    stackInfo, err := cfOps.DescribeStack(ctx, stack.Name)
    if err != nil {
        return nil, err  // ✅ AWS layer already has full context
    }
}
```

---

## Testing Error Messages

Add tests to verify error messages don't contain duplication:

```go
func TestErrorMessages_NoDuplication(t *testing.T) {
    tests := []struct {
        name          string
        operation     func() error
        maxOccurrence map[string]int  // word -> max allowed occurrences
    }{
        {
            name: "stack not found",
            operation: func() error {
                return deployer.DeploySingleStack(ctx, "nonexistent", "dev")
            },
            maxOccurrence: map[string]int{
                "nonexistent": 2,  // Once in quote, once in context
                "stack":       2,  // "stack 'nonexistent'" and "for context"
                "failed":      2,  // Should not appear many times
                "resolve":     1,  // Should not appear multiple times
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.operation()
            require.Error(t, err)
            
            errMsg := err.Error()
            for word, maxCount := range tt.maxOccurrence {
                actual := strings.Count(strings.ToLower(errMsg), strings.ToLower(word))
                assert.LessOrEqual(t, actual, maxCount,
                    "Word '%s' appears %d times (max %d): %s",
                    word, actual, maxCount, errMsg)
            }
        })
    }
}
```

---

## Migration Strategy

To fix existing duplication:

1. **Start at the top** (cmd layer)
2. **Remove all wrapping** in entry point and orchestration layers
3. **Review business logic layers** - keep only wrapping that adds semantic value
4. **Verify boundary layers** wrap all external errors with domain context
5. **Add tests** to prevent regression

---

## Summary

**Simple Rule**: Wrap at the boundary with external systems. Pass through everywhere else unless adding genuinely new semantic meaning.

**Result**: Clear, actionable error messages without redundancy.

**Before**:
```
failed to resolve stack dependencies: failed to get stack vpc: failed to resolve stack 'vpc' for context 'dev': stack 'vpc' not found in configuration
```

**After**:
```
failed to resolve stack 'vpc' for context 'dev': stack 'vpc' not found in configuration
```

Or even better (if we improve the config layer message):
```
stack 'vpc' not found in configuration for context 'dev'
```
