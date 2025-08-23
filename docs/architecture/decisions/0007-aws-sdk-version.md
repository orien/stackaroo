# 7. AWS SDK version choice

Date: 2025-08-23

## Status

Accepted

## Context

We need to choose an AWS SDK for Go to interact with CloudFormation and other AWS services. There are two major versions of the AWS SDK for Go available:

- **AWS SDK for Go v1**: The original SDK, mature and widely used
- **AWS SDK for Go v2**: The newer SDK with improved performance, modularity, and API design

Key considerations:
- Performance and memory efficiency
- Modularity and dependency management
- API design and developer experience
- Long-term support and maintenance
- Compatibility with modern Go practices
- CloudFormation service support quality

The main differences:
- **SDK v1**: Monolithic package, all AWS services included, synchronous APIs, older design patterns
- **SDK v2**: Modular packages, import only what you need, context-aware APIs, better error handling, improved performance

## Decision

We will use **AWS SDK for Go v2** for all AWS interactions in Stackaroo.

Specific packages adopted:
- `github.com/aws/aws-sdk-go-v2` - Core SDK
- `github.com/aws/aws-sdk-go-v2/config` - Configuration and credential loading
- `github.com/aws/aws-sdk-go-v2/service/cloudformation` - CloudFormation service client

Key factors in this decision:
- Modular design allows importing only required services, reducing binary size
- Better performance and lower memory usage
- Context-aware APIs align with modern Go practices
- Improved error handling and type safety
- Active development and long-term AWS support
- Better integration with AWS authentication mechanisms
- Cleaner API design for CloudFormation operations

## Consequences

**Positive:**
- Smaller binary size due to modular imports
- Better performance and resource efficiency
- Modern Go idioms (context, structured errors)
- Future-proof choice with ongoing AWS investment
- Cleaner, more maintainable code
- Better debugging and observability support

**Negative:**
- Less mature ecosystem compared to v1 (fewer third-party examples)
- Some learning curve for developers familiar with v1
- Potential breaking changes as v2 continues to evolve
- More verbose import statements for multiple services

We accept these trade-offs in favour of better performance, maintainability, and alignment with modern Go and AWS best practices.
