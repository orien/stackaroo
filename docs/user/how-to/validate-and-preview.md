---
title: 🔧 Validate and Preview
---

# 🔧 Validate and Preview

Use this checklist to confirm configuration changes before deploying.

## 1. Run a diff for each affected stack

```bash
stackaroo diff development payment-app-network
stackaroo diff production payment-app-service
```

- Start with non-production contexts to verify credentials and permissions.
- Review the output for each stack—Stackaroo highlights template, parameter, and tag changes.

## 2. Investigate validation errors

Common causes:

- Missing template files or typos in the `template` path.
- Incorrect `stack_name` or `output_key` values when referencing stack outputs.
- Parameters that break CloudFormation type constraints (for example, malformed CIDR blocks).

Fix the configuration and rerun `stackaroo diff` until it completes without errors.

## 3. Capture a final review

- Share the diff output in pull requests or chat for team approval.
- Once satisfied, proceed with `stackaroo deploy <context> <stack>` to apply the changes.

Keeping this loop tight reduces surprises and keeps deployments predictable.
