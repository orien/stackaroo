# 22. Simplified stack output reference field names

Date: 2025/11/03

## Status

Accepted

Amends [ADR 0017: Parameter resolution system enhancement](0017-parameter-resolution-system.md)

## Context

The stack output resolution syntax introduced in ADR 0017 uses verbose field names for referencing CloudFormation stack outputs:

```yaml
SecurityGroupIds:
  - type: stack-output
    stack_name: security-stack
    output_key: WebSGId
```

This verbosity creates several friction points:

1. **Redundant suffixes**: The `_name` and `_key` suffixes add little semantic value since the context (referencing a stack and its output) is already established by `type: stack-output`.

2. **Increased cognitive load**: Longer field names require more mental parsing when reading configurations, particularly in lists where multiple stack outputs are referenced.

3. **Verbose YAML**: In complex configurations with many stack output references, the extra characters accumulate into significantly longer, less scannable files.

4. **Inconsistent with ecosystem conventions**: Modern infrastructure-as-code tools (Terraform, Pulumi, etc.) tend toward concise field names when the context is clear.

Example of current verbosity in a realistic scenario:
```yaml
DatabaseSecurityGroups:
  - sg-baseline
  - type: stack-output
    stack_name: vpc-stack
    output_key: DatabaseSGId
  - type: stack-output
    stack_name: compliance-stack
    output_key: AuditSGId
  - type: stack-output
    stack_name: monitoring-stack
    output_key: DatadogSGId
```

The suffixes `_name` and `_key` don't add meaningful information—it's self-evident that we're referencing a stack's name and an output's key. The context is already provided by `type: stack-output`.

## Decision

Simplify stack output reference field names by removing redundant suffixes:

- `stack_name` → `stack`
- `output_key` → `output`

### Updated Syntax

```yaml
SecurityGroupIds:
  - type: stack-output
    stack: security-stack
    output: WebSGId
```

### Comparative Example

**Before:**
```yaml
DatabaseConfig:
  - type: stack-output
    stack_name: rds-stack
    output_key: DatabaseEndpoint
  - type: stack-output
    stack_name: secrets-stack
    output_key: ConnectionString
```

**After:**
```yaml
DatabaseConfig:
  - type: stack-output
    stack: rds-stack
    output: DatabaseEndpoint
  - type: stack-output
    stack: secrets-stack
    output: ConnectionString
```

The simplified version reduces character count by ~25% while maintaining complete clarity.

## Consequences

### Positive Consequences

**1. Improved readability:**
- Shorter field names are faster to read and comprehend
- Reduced visual clutter in configuration files
- Easier to scan complex parameter lists

**2. Better developer experience:**
- Less typing when writing configurations
- Faster to remember (shorter names are more memorable)
- Aligns with expectations from other IaC tools

**3. Cleaner examples and documentation:**
- Documentation examples are more concise
- Tutorial code is easier to follow
- Reduced scrolling in complex configurations

**4. Maintained semantic clarity:**
- Context is preserved through `type: stack-output`
- Field purposes remain self-evident
- No ambiguity introduced

**5. Consistency with modern conventions:**
- Matches patterns seen in Terraform (`data.aws_cloudformation_stack.name.outputs`)
- Aligns with Pulumi's stack reference syntax
- Follows Go community preference for concise naming

### Negative Consequences

**1. Breaking change:**
- Existing configurations using `stack_name` and `output_key` will break
- Requires migration effort from users
- Temporary disruption during transition period

**2. Migration complexity:**
- All examples, documentation, and test fixtures need updating
- Users must update their configurations before upgrading
- Need to provide clear migration guidance

**3. Potential confusion during transition:**
- Mixed documentation showing both old and new syntax
- Community resources may lag behind the change
- Support burden during migration window

### Migration Path

Users can migrate by performing a simple find-and-replace transformation:

```yaml
# Old syntax
type: stack-output
stack_name: my-stack
output_key: MyOutput

# New syntax
type: stack-output
stack: my-stack
output: MyOutput
```

The transformation is mechanical and can be automated with text processing tools:
```bash
sed -i 's/stack_name:/stack:/g' stackaroo.yaml
sed -i 's/output_key:/output:/g' stackaroo.yaml
```

This change prioritises long-term developer experience and configuration clarity over short-term migration cost.
