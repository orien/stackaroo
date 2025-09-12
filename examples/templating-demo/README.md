# Stackaroo CloudFormation Templating Demo

This example demonstrates Stackaroo's CloudFormation template templating capabilities using Go templates with Sprig functions.

## What This Demo Shows

The `webapp.yml` template showcases key templating features:

- **Context-aware resources**: Different resources based on environment
- **Dynamic configuration**: Environment-specific settings and values
- **Multiline script injection**: Clean UserData script templating
- **Conditional logic**: Resources that only exist in certain contexts
- **String transformations**: Using Sprig functions for formatting

## Template Features Demonstrated

### 1. Context Variables
```yaml
Description: {{ .Context | title }} web application for {{ .StackName }}
```
- `{{ .Context }}` - deployment context (development/production)
- `{{ .StackName }}` - stack name from configuration

### 2. Conditional Resources
```yaml
{{- if eq .Context "production" }}
  MonitoringRole:
    Type: AWS::IAM::Role
    # ... only created in production
{{- end }}
```

### 3. Multiline Script Injection
```yaml
UserData:
  Fn::Base64: |
{{- `#!/bin/bash
    yum update -y
    echo "ENVIRONMENT=` | nindent 10 }}{{ .Context | upper }}{{ `" > /etc/webapp.conf` | nindent 10 }}
```

### 4. Dynamic Values
```yaml
BucketName: {{ .StackName }}-data-{{ .Context }}-{{ randAlphaNum 6 | lower }}
InstanceSize: {{ if eq .Context "production" }}large{{ else }}small{{ end }}
```

### 5. Sprig Functions
- `{{ .Context | title }}` - Capitalize first letter
- `{{ .Context | upper }}` - Convert to uppercase
- `{{ randAlphaNum 6 | lower }}` - Generate random string
- `{{ if eq .Context "production" }}` - Conditional logic

## Usage

Deploy to development environment:
```bash
cd examples/templating-demo
stackaroo deploy development webapp
```

Deploy to production environment:
```bash
stackaroo deploy production webapp
```

## Environment Differences

### Development Context
- Creates basic web server
- No monitoring role
- No security group
- S3 versioning disabled
- Instance type: t3.micro

### Production Context
- Creates monitoring role
- Adds security group with HTTPS
- Enables S3 versioning
- Instance type: t3.large
- Additional HTTPS port in security group

## Template Processing Flow

1. **Raw template** is read from `templates/webapp.yml`
2. **Template variables** are injected:
   - `Context`: "development" or "production"
   - `StackName`: "webapp"
3. **Template is processed** using Go templates + Sprig
4. **Parameters are resolved** from stackaroo.yml
5. **Final CloudFormation** template is deployed

## Key Benefits

- **Single template** serves multiple environments
- **Clean multiline scripts** without escaping
- **Dynamic resource creation** based on context
- **Consistent naming** across environments
- **Maintainable** environment-specific logic
