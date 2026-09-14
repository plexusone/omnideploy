# OmniDeploy

**Universal deployment tool for container applications.**

Deploy to any cloud provider using any Infrastructure as Code tool.

## Overview

OmniDeploy separates **where** you deploy (targets) from **how** you provision (backends), giving you flexibility to choose the best combination for your needs.

| Target (WHERE) | Pulumi | CDK | Terraform |
|---|:---:|:---:|:---:|
| LightSail | ✓ | ◐ | ◐ |
| LightSail Instance | ✓ | ◐ | ◐ |
| ECS | ◐ | ◐ | ◐ |
| AgentCore | ◐ | ◐ | ◐ |
| Kubernetes | ◐ | - | ◐ |
| DigitalOcean | ◐ | - | ◐ |

✓ = Available &nbsp;&nbsp; ◐ = Planned &nbsp;&nbsp; - = Not applicable

## Key Features

- **Multi-Target**: Deploy to AWS LightSail (container or persistent VM instance), ECS, Kubernetes, DigitalOcean, and more
- **Multi-Backend**: Use Pulumi, AWS CDK, or Terraform as your IaC tool
- **Runtime Adapters**: Auto-detect OmniAgent, AgentKit, or generic container configs
- **Simple CLI**: One command to deploy, preview, or destroy
- **CI/CD Ready**: Built-in support for GitHub Actions and GitLab CI

## Quick Start

```bash
# Install
go install github.com/plexusone/omnideploy/cmd/omnideploy@latest

# Deploy to AWS LightSail using Pulumi
omnideploy up --config deploy.yaml --target lightsail --backend pulumi

# Preview changes
omnideploy preview --config deploy.yaml

# Destroy
omnideploy destroy --stack my-app
```

## Example Configuration

```yaml
name: my-api
region: us-east-1

container:
  image: nginx:latest
  ports:
    - container_port: 80
      protocol: HTTP

service:
  replicas: 1
  public: true

resources:
  size: micro
```

## Architecture

OmniDeploy uses a pluggable architecture with three main components:

1. **Targets** - Define where to deploy (LightSail, ECS, Kubernetes)
2. **Backends** - Define how to provision (Pulumi, CDK, Terraform)
3. **Runtime Adapters** - Translate app configs to deployment configs

```
┌─────────────────────────────────────────────────────┐
│                 Your Application                    │
│         (OmniAgent, AgentKit, Container)            │
└─────────────────────────┬───────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│              Runtime Adapter                        │
│    Converts app config → deployment config          │
└─────────────────────────┬───────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│              Deployment Config                      │
│     (containers, ports, resources, health)          │
└─────────────────────────┬───────────────────────────┘
                          │
            ┌─────────────┼─────────────┐
            ▼             ▼             ▼
      ┌──────────┐  ┌──────────┐  ┌──────────┐
      │ LightSail│  │   ECS    │  │   K8s    │
      │  Target  │  │  Target  │  │  Target  │
      └────┬─────┘  └────┬─────┘  └────┬─────┘
           │             │             │
           └─────────────┼─────────────┘
                         ▼
            ┌─────────────────────────┐
            │      IaC Backend        │
            │  (Pulumi, CDK, TF)      │
            └─────────────────────────┘
```

## Next Steps

- [Installation](getting-started/installation.md) - Install omnideploy
- [Quick Start](getting-started/quickstart.md) - Deploy your first app
- [Concepts](getting-started/concepts.md) - Understand the architecture
- [Configuration](configuration/schema.md) - Full configuration reference
