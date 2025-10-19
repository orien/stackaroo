---
title: 🔧 Link Stack Dependencies
---

# 🔧 Link Stack Dependencies

Follow this guide when one stack needs outputs produced by another stack.

## 1. Declare the dependency

Add the upstream stack to `depends_on`:

```yaml
  - name: payment-app-service
    template: app.yaml
    depends_on:
      - payment-app-network
    parameters:
      DesiredCapacity: "2"
      VpcId:
        type: stack-output
        stack_name: payment-app-network
        output_key: VpcId
```

- Declaring the dependency is optional, but strongly recommended whenever you reference another stack’s outputs.
- Stackaroo uses it to deploy stacks in the correct order and wait for completion before resolving parameters.

## 2. Reference the output explicitly

Use `type: stack-output` for each value you need:

```yaml
parameters:
  VpcId:
    type: stack-output
    stack_name: payment-app-network
    output_key: VpcId
  PublicSubnetIds:
    - type: stack-output
      stack_name: payment-app-network
      output_key: PublicSubnetA
    - type: stack-output
      stack_name: payment-app-network
      output_key: PublicSubnetB
```

Tips:

- `stack_name` must match the CloudFormation stack name. For stacks defined in `stackaroo.yaml`, use the same value you set in the `name` field; you can also point at external stacks by specifying their deployed CloudFormation name.
- Treat list parameters as arrays—you can mix literals and output references inside the same list.
- Keep output keys consistent with the source template to avoid runtime errors.

## 3. Validate the wiring

After adding dependencies, run:

```bash
stackaroo diff development payment-app-service
```

If Stackaroo cannot resolve an output, the diff fails with the AWS error so you can correct the stack name or output key before deploying.
