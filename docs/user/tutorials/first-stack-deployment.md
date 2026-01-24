---
title: 🎯 First Stack Deployment
---

# 🎯 First Stack Deployment

This tutorial walks through a complete Stackaroo journey—from a clean workspace to a successfully deployed CloudFormation stack. By the end, you will have:

- Installed the Stackaroo CLI from official binaries.
- Authenticated with AWS using a non-production account.
- Defined a minimal `stackaroo.yaml` configuration and accompanying template.
- Run a safe dry-run and then deployed the stack.

::: info Assumptions
You are comfortable with the AWS Console and CloudFormation, and you have basic command line fluency.
:::

## Prerequisites

- macOS, Linux, or Windows terminal with `curl` (or a browser) and ability to extract `.tar.gz` archives.
- AWS credentials pointing at a development account with CloudFormation permissions. Use an IAM user or role that cannot touch production resources.
- An empty or disposable S3 bucket for template uploads (Stackaroo will create one if configured to do so).

Make sure AWS credentials for a development account are configured. For example, you can set `AWS_PROFILE` to point at a non-production profile before starting, or rely on environment credentials from your shell, SSO session, or EC2/ECS role.

## Step 1 – Install the Stackaroo CLI

Download the latest release for your operating system from the [Stackaroo Releases](https://codeberg.org/orien/stackaroo/releases) page and place the binary on your `PATH`. On macOS or Linux you can fetch and extract it like this:

```bash
VERSION=1.0.0
ARCH=darwin-arm64  # Replace with linux-arm64, linux-amd64, etc.
URL="https://codeberg.org/orien/stackaroo/releases/download/v${VERSION}/stackaroo-${VERSION}-${ARCH}.tar.gz"
DIR="stackaroo-${VERSION}-${ARCH}"

curl -sL "$URL" | tar -xz
sudo mv "${DIR}/stackaroo" /usr/local/bin/
rm -rf "${DIR}"
```

> 💡 Prefer installing from source? Use `go install codeberg.org/orien/stackaroo@latest` instead.

Verify the installation:

```bash
stackaroo --version
```

## Step 2 – Prepare a working directory

Create a new project directory and initialise the expected layout:

```bash
mkdir -p stackaroo-tutorial/templates
cd stackaroo-tutorial
```

## Step 3 – Create a starter template

For the tutorial we will provision a lightweight CloudFormation stack that creates a simple S3 bucket. Add the template under `templates/tutorial-bucket.yaml`:

```yaml
AWSTemplateFormatVersion: '2010-09-09'
Description: Tutorial bucket provisioned by Stackaroo.

Resources:
  TutorialBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Sub stackaroo-tutorial-${AWS::AccountId}-${AWS::Region}

Outputs:
  BucketName:
    Description: Name of the tutorial bucket.
    Value: !Ref TutorialBucket
```

## Step 4 – Author `stackaroo.yaml`

Create a `stackaroo.yaml` file alongside your template:

```yaml
project: stackaroo-tutorial
tags:
  Project: stackaroo-tutorial
  Owner: your.name@example.com

templates:
  directory: templates

contexts:
  development:
    account: "123456789012" # Replace with your dev AWS account ID
    region: ap-southeast-4 # Replace with your preferred AWS region
    tags:
      Environment: development

stacks:
  stackaroo-tutorial:
    template: tutorial-bucket.yaml
    parameters: {}
```

Key points:

- Replace the `region` value with your preferred target region.
- Update the `account` value with your 12-digit AWS Account ID.
- The `templates.directory` block lets you reference `tutorial-bucket.yaml` without repeating the folder name.
- The stack key (`stackaroo-tutorial`) sets the CloudFormation stack name. Use a unique name if you expect to run the tutorial repeatedly.

## Step 5 – Preview the deployment

Use the diff command to inspect the CloudFormation change set without applying it:

```bash
stackaroo diff development stackaroo-tutorial
```

The output lists resources that will be created, updated, or deleted. Confirm the plan only includes the S3 bucket you expect.

## Step 6 – Deploy the stack

When you are satisfied with the diff, ship the change:

```bash
stackaroo deploy development stackaroo-tutorial
```

Stackaroo waits for CloudFormation to complete and prints stack outputs at the end. Navigate to the AWS Console → CloudFormation to confirm the stack status is `CREATE_COMPLETE`.

## Step 7 – Verify the resource

List S3 buckets to ensure the tutorial bucket exists:

```bash
aws s3 ls | grep stackaroo-tutorial
```

Retrieve detailed stack information if you need to script follow-up tasks:

```bash
stackaroo describe development stackaroo-tutorial
```

## Step 8 – Clean up (optional)

When you no longer need the tutorial stack, delete it to avoid ongoing charges:

```bash
stackaroo delete development stackaroo-tutorial
```

Confirm that CloudFormation reports the stack as deleted and that the S3 bucket has been removed. Remember that buckets containing objects cannot be deleted automatically—empty the bucket first if necessary.

## What next?

- Dive into targeted tasks in the [🔧 How-to Guides](/how-to/).
- Learn the reasoning behind Stackaroo’s configuration model in the forthcoming [💡 Explanations](/explanation/).
- Consult the [📘 Reference](/reference/) section for exhaustive CLI options and configuration schemas.
