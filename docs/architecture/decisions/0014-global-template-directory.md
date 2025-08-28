# 14. Global template directory

Date: 2025-01-28

## Status

Accepted

Amends [ADR 0010: File provider configuration structure](0010-file-provider-configuration-structure.md)

## Context

Currently, the file provider configuration structure defined in [ADR 0010: File provider configuration structure](0010-file-provider-configuration-structure.md) requires each stack to specify its complete template path explicitly. This creates verbose configuration files and makes it harder to maintain consistent directory structures:

```yaml
stacks:
  - name: vpc
    template: templates/vpc.yaml
  - name: app
    template: templates/app.yaml
  - name: database
    template: templates/database.yaml
  - name: monitoring
    template: templates/monitoring.yaml
```

This approach has several drawbacks:
- **Configuration verbosity**: The `templates/` prefix must be repeated for every stack
- **Inconsistent structure**: Nothing prevents mixing template locations arbitrarily
- **Maintenance overhead**: Moving templates requires updating every stack reference
- **Team confusion**: No clear convention for where templates should be stored

Most infrastructure-as-code tools provide a global template directory concept to address these issues. For example:
- **Terraform**: modules can reference a local path prefix
- **Pulumi**: supports workspace-level template directories
- **CDK**: uses consistent directory conventions

The current URI-based template interface ([ADR 0013: URI-based template interface](0013-uri-based-template-interface.md)) provides the foundation for this enhancement, as the file provider can resolve template paths to appropriate file:// URIs regardless of how the paths are specified in configuration.

## Decision

We will add **global template directory support** to the file provider configuration structure, extending the design established in [ADR 0010: File provider configuration structure](0010-file-provider-configuration-structure.md).

The enhanced configuration will support a global `templates` section:

```yaml
project: myapp
region: us-east-1

templates:
  directory: "templates/"

contexts:
  dev:
    account: "123456789012"

stacks:
  - name: vpc
    template: vpc.yaml          # Resolves to templates/vpc.yaml
  - name: app
    template: app.yaml          # Resolves to templates/app.yaml
  - name: shared-component
    template: shared/db.yaml    # Resolves to templates/shared/db.yaml
```

**Resolution Rules:**
1. **If global directory is specified**: Stack template paths are resolved relative to the global directory
2. **If no global directory**: Stack template paths are resolved relative to the configuration file directory (current behaviour)
3. **Absolute paths always win**: Absolute template paths bypass directory resolution entirely
4. **Context overrides not supported**: The templates directory is global-only (no context-specific template directories in this iteration)

**Path Resolution Logic:**
```
Given: templates.directory = "templates/" and stack.template = "vpc.yaml"
Result: file://<config_dir>/templates/vpc.yaml

Given: no templates.directory and stack.template = "templates/vpc.yaml"
Result: file://<config_dir>/templates/vpc.yaml

Given: templates.directory = "templates/" and stack.template = "/absolute/vpc.yaml"
Result: file:///absolute/vpc.yaml
```

**Backward Compatibility:**
- Existing configurations without `templates.directory` continue to work unchanged
- Existing stack template paths are resolved relative to configuration file directory as before
- No breaking changes to configuration schema or behaviour

## Consequences

**Positive:**
- **Reduced configuration verbosity**: Template paths become much shorter and cleaner
- **Consistent project structure**: Encourages standardised template organisation
- **Easier maintenance**: Changing global template location requires single configuration change
- **Clear conventions**: Teams have obvious place to store CloudFormation templates
- **Flexible usage**: Supports both simple (single directory) and complex (subdirectory) template organisation
- **Backward compatible**: No disruption to existing configurations
- **URI architecture benefit**: Implementation leverages existing URI resolution infrastructure

**Negative:**
- **Additional configuration complexity**: Introduces new optional configuration section
- **Potential confusion**: Teams might be unclear whether to use global directory or per-stack paths
- **Limited flexibility**: Global directory applies to all stacks (no per-stack override capability in this iteration)
- **Migration effort**: Teams wanting to adopt this feature need to reorganise existing configurations

**Implementation Requirements:**
- Extend `file.Config` struct to include optional `Templates` section with `Directory` field
- Update `resolveTemplateURI()` method to check for global template directory
- Maintain existing path resolution logic as fallback for backward compatibility
- Add validation to ensure template directory exists if specified
- Update configuration examples and documentation
- Ensure URI generation works correctly with global directory prefix

**Usage Impact:**
- New projects can use clean, minimal template references
- Existing projects can migrate incrementally by adding `templates.directory` and shortening stack template paths
- Template organisation becomes more standardised across projects
- Error messages can be improved to suggest template directory structure when templates are not found

This enhancement provides immediate value for configuration management whilst maintaining full backward compatibility, establishing a foundation for more advanced template source features in future iterations.
