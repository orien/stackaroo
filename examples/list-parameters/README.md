# List Parameters in Stackaroo

This example demonstrates Stackaroo's support for CloudFormation list parameters, including `List<Type>` and `CommaDelimitedList` parameter types.

## Overview

CloudFormation supports list parameters that accept multiple values, such as:
- `List<AWS::EC2::VPC::Id>` - Multiple VPC IDs  
- `List<AWS::EC2::SecurityGroup::Id>` - Multiple Security Group IDs
- `CommaDelimitedList` - Generic comma-separated list of strings

Stackaroo now supports these through a clean YAML array syntax where each list item can be resolved using different methods (literals, stack outputs, etc.).

## Basic Syntax

### Simple Literal Lists

```yaml
parameters:
  # Simple string list
  AllowedPorts:
    - "80"
    - "443"
    - "8080"
  
  # CIDR blocks
  TrustedCIDRs:
    - "10.0.0.0/8"
    - "172.16.0.0/12" 
    - "192.168.0.0/16"
```

### Mixed Lists (Literals + Stack Outputs)

```yaml
parameters:
  SecurityGroupIds:
    - sg-baseline123        # Literal hardcoded value
    - type: stack-output    # Dynamic from another stack
      stack: security-stack
      output: WebSGId
    - type: stack-output    # Another dynamic value
      stack: database-stack
      output: DatabaseSGId
    - sg-additional456      # Another literal
```

### All Dynamic Lists

```yaml
parameters:
  SubnetIds:
    - type: stack-output
      stack: vpc-stack
      output: PublicSubnet1Id
    - type: stack-output
      stack: vpc-stack
      output: PublicSubnet2Id  
    - type: stack-output
      stack: vpc-stack
      output: PublicSubnet3Id
```

## CloudFormation Parameter Type Mapping

Stackaroo list parameters map directly to CloudFormation parameter types:

| CloudFormation Type | Stackaroo YAML | Resolved Value |
|---------------------|----------------|----------------|
| `List<AWS::EC2::VPC::Id>` | `["vpc-123", "vpc-456"]` | `"vpc-123,vpc-456"` |
| `CommaDelimitedList` | `["web", "api", "db"]` | `"web,api,db"` |
| `List<AWS::EC2::SecurityGroup::Id>` | `[{type: stack-output, ...}]` | `"sg-123,sg-456,sg-789"` |

## Advanced Usage Patterns

### 1. Context-Specific List Overrides

Different environments can have completely different lists:

```yaml
stacks:
  web-app:
    parameters:
      InstanceTypes:
        - "t3.micro"
        - "t3.small"
    contexts:
      prod:
        parameters:
          InstanceTypes:
            - "t3.large"
            - "t3.xlarge"
            - "c5.large"    # Production needs more power
```

### 2. Cross-Stack Dependencies

Lists can reference outputs from multiple different stacks:

```yaml
parameters:
  TargetGroupArns:
    - type: stack-output
      stack: us-east-1-alb
      output: WebTargetGroupArn
    - type: stack-output
      stack: us-west-2-alb
      output: WebTargetGroupArn
    - type: stack-output
      stack: legacy-lb
      output: ApiTargetGroupArn
```

### 3. Incremental Environment Configuration

Add additional items for specific environments:

```yaml
# Base configuration
parameters:
  SecurityGroupIds:
    - sg-baseline
    - type: stack-output
      stack: web-security
      output: WebSGId

contexts:
  staging:
    parameters:
      SecurityGroupIds:
        - sg-baseline
        - type: stack-output
          stack: web-security
          output: WebSGId
        - type: stack-output
          stack: monitoring
          output: MonitoringSGId    # Add monitoring in staging
          
  prod:
    parameters:
      SecurityGroupIds:
        - sg-baseline
        - type: stack-output
          stack: web-security
          output: WebSGId
        - type: stack-output
          stack: monitoring
          output: MonitoringSGId
        - type: stack-output
          stack: compliance
          output: ComplianceSGId    # Add compliance in prod
```

### 4. Mixed Resolution Types

Combine different resolution methods in a single list:

```yaml
parameters:
  AllowedCIDRs:
    - "203.0.113.0/24"      # Office network (literal)
    - type: stack-output    # VPC CIDR (dynamic)
      stack: vpc-stack
      output: VpcCidrBlock
    - type: stack-output    # Partner VPN CIDR (dynamic)
      stack: vpn-stack
      output: PartnerCidrBlock
```

## Real-World Examples

### Multi-Tier Application

```yaml
stacks:
  # Load Balancer with multiple subnets
  load-balancer:
    parameters:
      SubnetIds:
        - type: stack-output
          stack: vpc
          output: PublicSubnet1Id
        - type: stack-output
          stack: vpc
          output: PublicSubnet2Id
        - type: stack-output
          stack: vpc
          output: PublicSubnet3Id

  # Web app with mixed security groups
  web-application:
    parameters:
      SecurityGroupIds:
        - sg-company-baseline    # Standard corporate SG
        - type: stack-output     # Application-specific SG
          stack: app-security
          output: WebAppSGId
        - type: stack-output     # Load balancer SG
          stack: load-balancer
          output: ALBSecurityGroupId
```

### Multi-Region Deployment

```yaml
stacks:
  global-service:
    parameters:
      # Target groups from multiple regions
      TargetGroupArns:
        - type: stack-output
          stack: us-east-1-app
          output: WebTargetGroupArn
        - type: stack-output
          stack: us-west-2-app
          output: WebTargetGroupArn
        - type: stack-output
          stack: eu-west-1-app
          output: WebTargetGroupArn
```

## Resolution Process

1. **Parse YAML**: Stackaroo detects YAML arrays and creates list parameters
2. **Resolve Items**: Each list item is resolved independently:
   - Literals use their string value directly
   - Stack outputs are fetched from CloudFormation
   - Future resolver types (SSM, Secrets Manager, etc.) work seamlessly
3. **Join Values**: All resolved values are joined with commas
4. **Pass to CloudFormation**: The comma-separated string is passed as the parameter value

Example resolution:
```yaml
SecurityGroupIds:
  - sg-literal123
  - type: stack-output
    stack: security
    output: WebSGId      # Resolves to "sg-dynamic456"
```

Becomes: `"sg-literal123,sg-dynamic456"`

## Best Practices

### 1. Consistent Ordering
Keep list items in a consistent order for predictable results:
```yaml
# Good: Consistent pattern
SecurityGroupIds:
  - sg-baseline           # Always first: baseline
  - type: stack-output    # Then: app-specific
    stack: app-security
    output: WebSGId
  - type: stack-output    # Finally: environment-specific  
    stack: monitoring
    output: MonitoringSGId
```

### 2. Comment Your Lists
Use comments to explain the purpose of each item:
```yaml
SecurityGroupIds:
  - sg-baseline123        # Company security baseline
  - type: stack-output    # Web tier access rules
    stack: web-security
    output: WebTierSGId
  - type: stack-output    # Application-specific rules
    stack: app-security  
    output: AppSGId
```

### 3. Environment-Specific Variations
Use context overrides for environment-specific variations:
```yaml
# Base: minimal for development
parameters:
  SecurityGroupIds:
    - sg-dev-baseline

contexts:
  prod:
    parameters:
      # Production: comprehensive security
      SecurityGroupIds:
        - sg-prod-baseline
        - type: stack-output
          stack: security
          output: WebSGId
        - type: stack-output
          stack: monitoring
          output: MonitoringSGId
        - type: stack-output
          stack: compliance
          output: ComplianceSGId
```

### 4. Dependency Management
Ensure stacks are properly ordered in `depends_on`:
```yaml
stacks:
  vpc:
    # No dependencies
    
  security:
    depends_on: [vpc]       # Security groups need VPC
    
  application:
    depends_on: [vpc, security]  # App needs both VPC and security groups
    parameters:
      SecurityGroupIds:
        - type: stack-output
          stack: security  # This dependency is declared above
          output: WebSGId
```

## Migration from Single Values

If you have existing single-value parameters, you can easily migrate:

### Before (single value):
```yaml
parameters:
  SecurityGroupId:
    type: stack-output
    stack: security-stack
    output: WebSGId
```

### After (list with single item):
```yaml
parameters:
  SecurityGroupIds:  # Note: parameter name typically changes to plural
    - type: stack-output
      stack: security-stack
      output: WebSGId
```

### After (list with multiple items):
```yaml
parameters:
  SecurityGroupIds:
    - sg-baseline123    # Add baseline security group
    - type: stack-output
      stack: security-stack
      output: WebSGId
    - sg-monitoring456  # Add monitoring access
```

## CloudFormation Template Compatibility

Your CloudFormation templates need to use list parameter types:

```yaml
# CloudFormation template
Parameters:
  SecurityGroupIds:
    Type: List<AWS::EC2::SecurityGroup::Id>
    Description: List of security group IDs

  AllowedPorts:
    Type: CommaDelimitedList
    Description: List of allowed port numbers

Resources:
  LaunchTemplate:
    Type: AWS::EC2::LaunchTemplate
    Properties:
      LaunchTemplateData:
        SecurityGroupIds: !Ref SecurityGroupIds  # Direct reference to list
```

## Troubleshooting

### Empty Values
Empty resolved values are automatically filtered out:
```yaml
parameters:
  MyList:
    - "value1"
    - ""          # Empty string - will be filtered out
    - "value2"
# Result: "value1,value2" (not "value1,,value2")
```

### Debugging Resolution
Use `stackaroo diff` to see resolved parameter values before deployment:
```bash
stackaroo diff dev web-application
```

This will show you exactly what comma-separated values will be passed to CloudFormation.

## Getting Started

1. **Update your CloudFormation templates** to use list parameter types
2. **Convert single parameters to lists** in your `stackaroo.yaml`
3. **Test with `stackaroo diff`** to verify resolution
4. **Deploy incrementally** to validate the changes

The `stackaroo.yaml` file in this directory provides a comprehensive example you can adapt for your own infrastructure.
