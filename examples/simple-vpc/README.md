# Simple VPC Example

This example demonstrates basic Stackaroo usage with a simple VPC deployment across multiple environments.

## What This Example Shows

- **Multi-environment deployment** - Deploy the same infrastructure to dev, staging, and prod
- **Context-specific overrides** - Different VPC CIDR blocks per environment
- **Cross-account deployment** - Production uses a separate AWS account
- **Parameter inheritance** - Global defaults with environment-specific overrides
- **Tag management** - Consistent tagging across environments

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
- **Consistent tagging**: Environment-specific tags with global defaults

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

3. **Deploy to development**:
   ```bash
   ../../stackaroo deploy vpc --context dev
   ```

4. **Deploy to staging**:
   ```bash
   ../../stackaroo deploy vpc --context staging
   ```

5. **Deploy to production** (requires production account access):
   ```bash
   ../../stackaroo deploy vpc --context prod
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

| Environment | Region    | Account      | VPC CIDR      | Tags |
|-------------|-----------|--------------|---------------|------|
| dev         | us-west-2 | 123456789012 | 10.1.0.0/16   | Environment: dev |
| staging     | us-east-1 | 123456789012 | 10.2.0.0/16   | Environment: staging |
| prod        | us-east-1 | 987654321098 | 10.3.0.0/16   | Environment: prod, Monitoring: enabled |

## Viewing Deployment Status

Check the status of your deployments:
```bash
../../stackaroo status vpc --context dev
```

## Cleanup

To remove the infrastructure:
```bash
../../stackaroo delete vpc --context dev
../../stackaroo delete vpc --context staging
../../stackaroo delete vpc --context prod
```

## Learning Points

This example demonstrates:

1. **Configuration inheritance** - How global settings cascade to specific contexts
2. **Multi-environment patterns** - Same template, different parameters
3. **Cross-account deployment** - Managing resources across AWS accounts
4. **Infrastructure consistency** - Identical setup with environment-appropriate sizing
5. **Tag management** - Consistent tagging strategy across environments

## Next Steps

- Explore adding more stacks (databases, applications) that depend on this VPC
- Try different parameter combinations
- Add custom tags for cost allocation or compliance
- Experiment with different AWS regions and accounts