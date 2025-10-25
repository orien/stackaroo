---
title: 🔧 Add a Stack
---

# 🔧 Add a Stack

Use this guide when you need to introduce a new CloudFormation stack to `stackaroo.yaml`.

## 1. Create the stack entry

Add an entry to the `stacks` map:

```yaml
stacks:
  payment-app-network:
    template: network.yaml
    parameters:
      VpcCidr: "10.0.0.0/16"
      AvailabilityZones:
        - ap-southeast-4a
        - ap-southeast-4b
    tags:
      CostCentre: shared-services
      Tier: shared-network
```

- The key (e.g., `payment-app-network`) becomes the CloudFormation stack name. Keep it unique per region.
- `template` resolves relative to `templates.directory`.
- `parameters` accept literal values, nested lists, or stack-output references.
- `tags` override the project defaults for this stack only.

## 2. Override for specific contexts

Tailor parameters or tags per environment by nesting a `contexts` block:

```yaml
  payment-app-service:
    template: app.yaml
    parameters:
      DesiredCapacity: "2"
      VpcId:
        type: stack-output
        stack_name: payment-app-network
        output_key: VpcId
    contexts:
      production:
        parameters:
          DesiredCapacity: "6"
        tags:
          CostCentre: prod-payments
          Tier: critical
```

Rules to remember:

- Literal keys replace the base value (`DesiredCapacity` increases from 2 to 6 in production).
- Lists replace the entire list—restate every item if only one value changes.
- Tags merge with the base set; reuse the same key to override a value.

## 3. Keep stacks readable

- Group related stacks (networking, security, application) together.
- Add comments for non-obvious parameter choices.
- Move large inline assets (user data scripts, policy documents) into separate template files or use template helpers.

Once the stack definition is in place, see the dependency and validation guides to wire it into the rest of the project safely.
