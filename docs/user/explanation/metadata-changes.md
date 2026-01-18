# Metadata-Only Template Changes

## Overview

CloudFormation templates contain both infrastructure-defining elements and metadata elements. When you modify only metadata elements, CloudFormation treats this as "no infrastructure changes" because no actual AWS resources are affected.

## What Are Metadata-Only Changes?

Metadata-only changes include modifications to:

- **Description field** - The template-level description
- **Metadata section** - Custom metadata for documentation or tooling
- **Comments** - YAML or JSON comments (stripped during processing)
- **Whitespace and formatting** - Indentation, line breaks, etc.

These changes do not affect:
- Resource definitions
- Resource properties
- Parameters
- Outputs
- Conditions
- Mappings

## How Stackaroo Handles This

### During `diff`

When you run `stackaroo diff` and only metadata has changed:

```bash
stackaroo diff production my-stack
```

You'll see output like:

```
my-stack - production

Changes Detected
Your local configuration differs from the deployed stack.

TEMPLATE

  @@ -1,4 +1,4 @@
  -Description: IAM configuration for human access
  +Description: IAM configuration for human access - updated

   Resources:
     UserGroup:

PLAN

No Infrastructure Changes

The template changes shown above are metadata-only and do not affect infrastructure.

Examples of metadata-only changes:
  • Template Description field
  • Metadata section
  • Comments or formatting

No deployment is required for these changes.
```

**Key points:**
- The template diff is shown (you can see what changed)
- CloudFormation reports "no infrastructure changes"
- The command exits successfully (exit code 0)
- This is informational, not an error

### During `deploy`

When you run `stackaroo deploy` with metadata-only changes:

```bash
stackaroo deploy production my-stack
```

The output will be:

```
my-stack - production

Changes Detected
Your local configuration differs from the deployed stack.

TEMPLATE

  @@ -1,4 +1,4 @@
  -Description: IAM configuration for human access
  +Description: IAM configuration for human access - updated

   Resources:
     UserGroup:

PLAN

No Infrastructure Changes

The template changes shown above are metadata-only and do not affect infrastructure.

Examples of metadata-only changes:
  • Template Description field
  • Metadata section
  • Comments or formatting

No deployment is required for these changes.


No infrastructure changes for stack my-stack (metadata-only changes detected)
```

**Behaviour:**
- The command shows the diff
- No deployment is executed
- Exits successfully (exit code 0)
- Treated the same as "no changes detected"

## Why This Happens

CloudFormation uses ChangeSets to preview infrastructure changes. When you create a ChangeSet with only metadata changes, AWS returns:

> "The submitted information didn't contain changes. Submit different information to create a change set."

This is not an error—it's CloudFormation's way of saying "these changes don't affect your infrastructure."

## When Is This Useful?

Metadata-only changes are useful for:

1. **Documentation updates** - Updating template descriptions without triggering deployments
2. **Version comments** - Adding version or change log information to templates
3. **Formatting standardisation** - Reformatting templates for consistency
4. **Tooling metadata** - Adding metadata for other tools to consume

## Combining Metadata with Real Changes

If you change both metadata AND infrastructure-defining elements (like adding a resource or changing a property), CloudFormation will:

1. Detect the infrastructure changes
2. Create a ChangeSet showing resource-level impacts
3. Ignore the metadata changes in the ChangeSet
4. Proceed with deployment if approved

The metadata changes will be included in the template update, but they won't appear in the ChangeSet details.

## Example Scenario

You want to update documentation in your template without triggering a deployment:

**Before:**
```yaml
Description: VPC configuration

Resources:
  VPC:
    Type: AWS::EC2::VPC
    Properties:
      CidrBlock: 10.0.0.0/16
```

**After:**
```yaml
Description: VPC configuration for production environment

Metadata:
  Documentation:
    LastUpdated: 2025-01-18
    Owner: Infrastructure Team

Resources:
  VPC:
    Type: AWS::EC2::VPC
    Properties:
      CidrBlock: 10.0.0.0/16
```

Running `stackaroo diff` or `stackaroo deploy` will show the changes but report "No Infrastructure Changes" and won't execute a deployment.

## Troubleshooting

### I expected changes but got "No Infrastructure Changes"

If you believe you made infrastructure changes but Stackaroo reports "No Infrastructure Changes":

1. **Check your changes** - Verify you modified resources, properties, or parameters
2. **Review the template diff** - Look at what actually changed
3. **Check parameter values** - Ensure parameter changes affect resources
4. **Validate template** - Run `stackaroo validate` to check for syntax errors

### Common mistakes:

- Changing only the Description when you meant to change a resource
- Modifying Metadata instead of resource Properties
- Adding comments but not actual configuration changes
- Formatting changes without functional changes

## Related Commands

- `stackaroo diff` - Preview changes including metadata-only detection
- `stackaroo deploy` - Deploy changes (skips deployment for metadata-only)
- `stackaroo validate` - Validate template syntax

## Technical Details

Stackaroo detects metadata-only changes by:

1. Creating an AWS ChangeSet for the proposed template
2. AWS evaluates the ChangeSet and returns status
3. If status is FAILED with reason containing "didn't contain changes"
4. Stackaroo recognises this as metadata-only changes
5. Handles it gracefully (no error, informational message)

This detection happens automatically—you don't need to specify any flags.