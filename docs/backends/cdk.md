# AWS CDK Backend

!!! note "Coming Soon"
    The CDK backend is planned for a future release.

## Overview

AWS Cloud Development Kit provides:

- Native AWS integration
- CloudFormation under the hood
- Strong typing with Go
- Construct libraries
- CDK Pipelines for CI/CD

## Planned Features

- CDK stack generation
- CloudFormation deployment
- CDK Pipelines integration
- Cross-stack references
- Custom constructs support

## When to Use

**Choose CDK when:**

- AWS-only deployments
- Team familiar with CloudFormation
- Want tight AWS integration
- Using CDK Pipelines

## Comparison with Pulumi

| Feature | CDK | Pulumi |
|---------|-----|--------|
| State | CloudFormation | Local/Cloud |
| Multi-cloud | AWS only | Yes |
| Language | Go, TS, Python | Go, TS, Python |
| Deployment | CloudFormation | Direct API |
