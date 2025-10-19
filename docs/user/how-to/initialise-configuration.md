---
title: 🔧 Initialise Configuration
---

# 🔧 Initialise Configuration

Set up the foundations of your Stackaroo project before adding stacks. This guide covers the top-level metadata, template directory, and environment contexts inside `stackaroo.yaml`.

## 1. Describe the project

Open `stackaroo.yaml` and populate the metadata block:

```yaml
project: payment-app
tags:
  Project: payment-app
  Owner: payments-team@example.com
templates:
  directory: templates
```

- `project` is a descriptive label you can reuse in dashboards or cost reports.
- `tags` apply globally; individual stacks can override specific keys later.
- `templates.directory` avoids repeating the folder path for every stack entry.

## 2. Define environment contexts

Contexts identify AWS accounts and regions for each environment:

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

Guidelines:

- Use 12-digit AWS account IDs and keep them in sync with your IAM roles or SSO assignments.
- Apply environment-specific tags (cost centre, owner, business unit) so they propagate to every stack automatically.
- Add staging, disaster recovery, or sandbox contexts using the same structure.

## 3. Sanity-check the configuration

Before moving on, ensure:

- The project tags match your organisation’s naming scheme.
- Context regions align with the templates you intend to deploy.
- Accounts are restricted to non-production while you test the setup.

Next, follow the remaining how-to guides to add stacks, link dependencies, and validate changes.
