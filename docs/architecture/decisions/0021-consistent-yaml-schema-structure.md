# 21. Consistent YAML schema structure

Date: 2025/01/13

## Status

Accepted

Amends [ADR 0010: File provider configuration structure](0010-file-provider-configuration-structure.md)

## Context

The `stackaroo.yaml` schema contained a structural inconsistency in how entities were defined:

- **Stacks** used a list structure with `name` as an attribute:
  ```yaml
  stacks:
    - name: vpc
      template: vpc.yaml
  ```

- **Parameters** used a map structure with the name as the key:
  ```yaml
  parameters:
    VpcCidr: 10.0.0.0/16
  ```

This inconsistency created several issues:

1. **Cognitive overhead**: Users had to remember two different patterns for conceptually similar entities (both are named configuration blocks).
2. **Implementation complexity**: The codebase required different access patterns (slice iteration vs map lookup) for similar operations.
3. **Schema evolution**: Future additions of named entities would lack a clear precedent to follow.

Map-based structures offer practical advantages:
- Direct lookup by name without iteration
- Implicit uniqueness constraint (map keys must be unique)
- Natural alignment with parameter definitions already in use

## Decision

Change the `stacks` section to use a **map structure** where the stack name is the key, matching the pattern already used for parameters:

```yaml
stacks:
  vpc:
    template: vpc.yaml
    parameters:
      VpcCidr: 10.0.0.0/16
```

The stack name (previously in the `name` field) becomes the map key. The `Name` field is removed from the `Stack` struct in the internal representation, with the name provided as a separate parameter during resolution.

## Consequences

### Positive Consequences

- **Structural consistency**: Both stacks and parameters now use the same map-based pattern, reducing cognitive load.
- **Simpler implementation**: Stack lookup becomes `O(1)` map access rather than `O(n)` slice iteration.
- **Clear precedent**: Future named entities (e.g. modules, providers) have an established pattern to follow.
- **Implicit validation**: YAML parsers enforce unique stack names automatically through map key constraints.

### Negative Consequences

- **Breaking change**: Existing `stackaroo.yaml` files require manual migration to the new structure.
- **Migration effort**: All examples, documentation, and test fixtures needed updating.
- **Temporary disruption**: Users must update their configurations before upgrading to versions with this change.

### Migration Path

Users can migrate configurations by transforming:

```yaml
stacks:
  - name: stack-name
    template: template.yaml
```

To:

```yaml
stacks:
  stack-name:
    template: template.yaml
```

The transformation is mechanical and can be automated with text processing tools.