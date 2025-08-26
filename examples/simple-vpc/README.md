# Simple VPC Example

This example demonstrates basic Stackaroo usage with a simple VPC deployment across multiple contexts.

## What This Example Shows

- **Multi-environment deployment** - Deploy the same infrastructure to dev, staging, and prod
- **Context-specific overrides** - Different VPC CIDR blocks per context
- **Cross-account deployment** - Production uses a separate AWS account
- **Parameter inheritance** - Global defaults with context-specific overrides
- **Tag management** - Consistent tagging across contexts
- **Change preview** - See exactly what infrastructure changes before deployment

## Prerequisites

- Go 1.21+ (to build stackaroo)
- AWS CLI configured with appropriate credentials
- Access to AWS accounts specified in the configuration:
  - Account `123456789012` for dev and staging
  - Account `987654321098` for production

## Project Structure

```
simple-vpc/
├── README.md              # This file
├── stackaroo.yaml         # Stackaroo configuration
└── templates/
    └── vpc.yaml           # CloudFormation VPC template
```

## Configuration Highlights

The `stackaroo.yaml` file defines:

- **3 deployment contexts**: dev (us-west-2), staging (us-east-1), prod (us-east-1)
- **Different VPC CIDR blocks**: 10.1.0.0/16 (dev), 10.2.0.0/16 (staging), 10.3.0.0/16 (prod)
- **Cross-account deployment**: Production uses a separate AWS account
- **Consistent tagging**: Context-specific tags with global defaults

## Usage

1. **Build Stackaroo** (from the project root):
   ```bash
   cd ../../  # Go to stackaroo project root
   go build -o stackaroo .
   ```

2. **Navigate to this example**:
   ```bash
   cd examples/simple-vpc
   ```

3. **Deploy to development** (shows preview before applying changes):
   ```bash
   ../../stackaroo deploy dev vpc
   ```

4. **Deploy to staging** (shows preview before applying changes):
   ```bash
   ../../stackaroo deploy staging vpc
   ```

5. **Deploy to production** (requires production account access):
   ```bash
   ../../stackaroo deploy prod vpc
   ```

## Preview Output

When you run the deploy commands, Stackaroo will show you exactly what changes will be made:

```
=== Calculating changes for stack vpc ===
Changes to be applied to stack vpc:

Status: CHANGES DETECTED (for updates) or Creating new stack: vpc (for new deployments)

Template Changes:
-----------------
✓ Template has been modified (if updating)
Resource changes:
  + 6 resources to be added (for new stacks)

AWS CloudFormation Preview:
---------------------------
Resource Changes:
  + VPC (AWS::EC2::VPC)
  + InternetGateway (AWS::EC2::InternetGateway)
  + PublicSubnet (AWS::EC2::Subnet)
  + PrivateSubnet (AWS::EC2::Subnet)
  + PublicRouteTable (AWS::EC2::RouteTable)
  + PrivateRouteTable (AWS::EC2::RouteTable)

=== Deploying stack vpc ===
[Live deployment events appear here...]
```

## What Gets Deployed

Each deployment creates:
- **VPC** with DNS support and hostnames enabled
- **Internet Gateway** for internet access
- **Public subnet** with auto-assign public IP
- **Private subnet** for internal resources
- **Route tables** with appropriate routing
- **Proper tagging** for environment identification

## Environment Differences

| Context | Region    | Account      | VPC CIDR      | Tags |
|---------|-----------|--------------|---------------|------|
| dev     | us-west-2 | 123456789012 | 10.1.0.0/16   | Environment: dev |
| staging | us-east-1 | 123456789012 | 10.2.0.0/16   | Environment: staging |
| prod    | us-east-1 | 987654321098 | 10.3.0.0/16   | Environment: prod, Monitoring: enabled |

## Viewing Deployment Status

Check the status of your deployments:
```bash
../../stackaroo status dev vpc
```

## Cleanup

To remove the infrastructure:
```bash
../../stackaroo delete dev vpc
../../stackaroo delete staging vpc
../../stackaroo delete prod vpc
```

## Learning Points

This example demonstrates:

1. **Configuration inheritance** - How global settings cascade to specific contexts
2. **Multi-environment patterns** - Same template, different parameters
3. **Cross-account deployment** - Managing resources across AWS accounts
4. **Infrastructure consistency** - Identical setup with context-appropriate sizing
5. **Tag management** - Consistent tagging strategy across contexts
6. **Change preview** - How Stackaroo shows you exactly what will change before deployment

## Next Steps

- Explore adding more stacks (databases, applications) that depend on this VPC
- Try different parameter combinations
- Add custom tags for cost allocation or compliance
- Experiment with different AWS regions and accounts
