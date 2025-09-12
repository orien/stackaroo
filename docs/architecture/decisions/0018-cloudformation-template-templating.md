# 18. CloudFormation template templating support

Date: 2025-09-12

## Status

Accepted

## Context

Stackaroo currently supports static CloudFormation templates with parameter substitution through the parameter resolution system (ADR 0017). However, teams have encountered scenarios where static templates are insufficient:

**Limitations of Static Templates:**
- Cannot conditionally include/exclude resources based on deployment context
- No support for injecting multiline scripts (UserData, cloud-init, bootstrap scripts)
- Repetitive template patterns across similar resource definitions
- Inability to generate dynamic resource names or configurations
- Limited reusability across different deployment scenarios

**Real-World Use Cases Requiring Templating:**
1. **Multiline Script Injection**: EC2 UserData scripts, ECS task definitions, and Lambda deployment packages often require multiline shell scripts or configuration files that are difficult to manage as single-line parameters.

2. **Conditional Resources**: Different environments may require different resources (e.g., ALB only in production, simplified security groups in development).

3. **Dynamic Resource Naming**: Generate resource names based on environment, application version, or deployment context.

4. **Template Reuse**: Single template serving multiple similar use cases with different resource configurations.

**Example Problem - Multiline UserData:**
```yaml
# Current workaround: Unwieldy single-line parameter
Parameters:
  UserDataScript: "#!/bin/bash\nyum update -y\nyum install -y docker\nsystemctl start docker\n# More commands..."

# Desired: Clean multiline injection
UserData:
  Fn::Base64: |
    #!/bin/bash
    yum update -y
    yum install -y docker
    systemctl start docker

    # Install application
    {{- range .ApplicationPorts }}
    echo "Configuring port {{ . }}"
    {{- end }}

    /opt/bootstrap/init.sh
```

**Technology Options Evaluated:**

1. **Go's `text/template`**: Built-in, zero dependencies, suitable for YAML/JSON generation
2. **Go's `html/template`**: HTML-focused with auto-escaping that interferes with CloudFormation syntax
3. **Sprig library**: Extends `text/template` with 100+ utility functions (string manipulation, date/time, collections)
4. **Gomplate**: Feature-rich but heavier dependency with external data source integration
5. **Mustache**: Logic-less templates, limited control flow capabilities

**Activation Strategy Considerations:**

**Always-On Approach:**
- Process all templates through templating engine
- Risk of accidental processing of literal `{{ }}` in existing templates
- Potential performance impact on projects with many static templates
- Backwards compatibility concerns

**Suffix-Based Activation:**
- Only process templates with specific file extensions (e.g., `.tpl.yml`)
- Explicit opt-in mechanism
- Clear separation between static and templated files
- Backwards compatibility guaranteed

## Decision

We will implement **CloudFormation template templating support** using:

1. **Go's `text/template` with Sprig enhancement library**
2. **Always-on templating** for all CloudFormation templates

## Consequences

### Positive Consequences

**1. Enhanced Template Capabilities:**
- Clean multiline string injection using `nindent` function
- Conditional resource generation based on deployment context
- Dynamic resource naming and configuration
- Template reuse across multiple deployment scenarios

**2. Developer Experience Improvements:**
- Intuitive YAML literal block syntax for multiline content
- Familiar Go template syntax for teams already using Go
- Rich function library (100+ Sprig functions) for complex transformations
- Clear file organisation with explicit template identification

**3. Backwards Compatibility:**
- Existing CloudFormation templates continue to work unchanged
- Templates without template directives pass through unmodified
- Gradual adoption possible (add template features as needed)
- No changes to CLI interface or core workflows

**4. Simplicity and Consistency:**
- Unified approach: all templates have templating capabilities available
- Template processing for all files provides consistent behavior
- No need to choose between static and templated files
- Simpler mental model and documentation
- Clear error messages for template syntax issues
- Integration with existing parameter resolution system (ADR 0017)

### Negative Consequences

**1. Template Processing Overhead:**
- All templates processed through templating engine (minor performance impact)
- Template processing overhead applies to all templates
- Additional dependency on Sprig library
- Learning curve for teams unfamiliar with Go template syntax
- Potential debugging complexity for template syntax errors

**2. Literal Brace Handling:**
- Templates containing literal `{{ }}` may require escaping
- Literal `{{ }}` in templates may require escaping in some cases
- Potential confusion when CloudFormation templates contain brace syntax
- Need for documentation on escaping template syntax when literal braces are required

**3. Template Debugging:**
- Template errors may not surface until deployment time
- Difficulty in debugging complex template logic
- Potential for template variables to mask CloudFormation parameter errors

**4. Maintenance Overhead:**
- Need to maintain template syntax compatibility with future Sprig versions
- Additional testing requirements for templated vs static templates
- Documentation and training requirements for template features

This enhancement maintains Stackaroo's principle of simplicity while providing powerful templating capabilities for advanced use cases, particularly multiline script injection scenarios common in CloudFormation deployments.
