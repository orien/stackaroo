---
title: 🔧 Configure Stacks
---

# 🔧 Configure Stacks

This guide shows the practical steps for managing Stackaroo configuration files. Follow it when you want to add a new stack, adjust parameters, or tailor contexts without digging through architectural background.

We start from an existing workspace containing a `stackaroo.yaml` file and CloudFormation templates.

## 1. Review project metadata

Open `stackaroo.yaml` and confirm the top-level metadata describes your project:

```yaml
project: payment-app
tags:
  Project: payment-app
  Owner: payments-team@example.com
templates:
  directory: templates
```

- `project` lets you capture whichever label your organisation uses for the deployment (for example, cost centre or product line). Stackaroo does not reference it directly, but you can surface it in reports or dashboards generated from your configuration.
- `tags` apply to every stack unless overridden later.
- `templates.directory` lets you omit the directory prefix in each stack entry.

## 2. Define contexts

Contexts identify AWS accounts and regions for each environment. Add or edit entries under `contexts`:

```yaml
contexts:
  development:
    account: "123456789012"
    region: ap-southeast-4
    tags:
      Environment: development
      CostCentre: dev-payments
  production:
    account: "210987654321"
    region: eu-west-1
    tags:
      Environment: production
      CostCentre: prod-payments
```

Tips:

- Use IAM roles or AWS Organisations to map accounts cleanly; the value must be a 12-digit AWS account ID.
- Apply environment-specific tags here so every stack inherits them automatically (for example, the development context sets `CostCentre: dev-payments`).

## 3. Add a stack

Stacks describe the templates and parameters you want to deploy. Append a new block to the `stacks` list:

```yaml
stacks:
  - name: payment-app-network
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

- `name` becomes the CloudFormation stack name; make it unique within the region.
- `template` points to a local file relative to `templates.directory`.
- `parameters` accept literal values, nested lists, or stack output references (see [List Parameters example](https://github.com/orien/stackaroo/tree/main/examples/list-parameters)).
- `tags` override or extend inherited tags for this stack only—in this case we label the network as a shared foundation service.

## 4. Reference other stacks

Use `depends_on` to guarantee deployment order. Anytime a stack references outputs from another stack, add the dependency here so Stackaroo knows to deploy and resolve in the correct sequence (it is optional, but omitting it can lead to race conditions):

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

Stackaroo deploys stacks in dependency order and resolves outputs at runtime. Explicit dependencies and parameter references mean `payment-app-service` waits for `payment-app-network` to finish before resolving `VpcId`.

## 5. Override per context

Add a `contexts` section inside a stack to change parameters or tags for a specific environment:

```yaml
  - name: payment-app-service
    template: app.yaml
    contexts:
      production:
        parameters:
          DesiredCapacity: "6"
        tags:
          CostCentre: prod-payments
          Tier: critical
```

Overrides merge with defaults:

- Literal fields such as `DesiredCapacity` replace the base value (production scales from 2 to 6 instances).
- Lists are replaced entirely; re-specify all items if you need partial differences.
- Tags merge with the base set—use the same key to override a value.

## 6. Validate and preview

Run Stackaroo commands during edits to catch mistakes:

```bash
stackaroo diff development payment-app-network
stackaroo diff production payment-app-service
```

Fix any validation errors before deploying. Common issues include missing templates, misspelled stack names in `depends_on`, or parameters that fail CloudFormation type checks.

## 7. Keep configuration readable

As the file grows:

- Group related stacks (networking, security, application) together.
- Prefer descriptive comments over abbreviations.
- Extract large parameters (for example, inline scripts) into separate template files or use templating features when necessary.

## Next steps

- Learn the configuration internals in the forthcoming [💡 Explanations](/explanation/).
- Follow the [🎯 Tutorial](/tutorials/first-stack-deployment) to see these concepts in action.
- Contribute additional how-to guides as you codify repeatable workflows in your organisation.
