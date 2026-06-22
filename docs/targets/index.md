# Deployment Targets

Targets define **where** your containers run.

## Available Targets

| Target | Status | Description | Best For |
|--------|--------|-------------|----------|
| [LightSail](lightsail.md) | ✓ Available | AWS LightSail Container Service | Simple apps, cost-effective |
| [ECS](ecs.md) | ◐ Planned | AWS ECS with Fargate | Production, auto-scaling |
| [AgentCore](agentcore.md) | ◐ Planned | AWS Bedrock AgentCore | AI agent deployments |
| [Kubernetes](kubernetes.md) | ◐ Planned | Any Kubernetes cluster | Complex orchestration |
| [DigitalOcean](digitalocean.md) | ◐ Planned | App Platform | Simple PaaS |

## Choosing a Target

### Cost Considerations

| Target | Starting Price | Scaling |
|--------|---------------|---------|
| LightSail | ~$7/month | Manual |
| ECS Fargate | ~$0.04/vCPU/hour | Auto |
| DigitalOcean | ~$5/month | Manual |

### Feature Comparison

| Feature | LightSail | ECS | Kubernetes |
|---------|-----------|-----|------------|
| Auto-scaling | ❌ | ✓ | ✓ |
| Load balancing | ✓ (built-in) | ✓ (ALB/NLB) | ✓ (Ingress) |
| Custom domains | ✓ | ✓ | ✓ |
| SSL/TLS | ✓ (auto) | ✓ (ACM) | ✓ (cert-manager) |
| Persistent storage | ❌ | ✓ (EFS) | ✓ (PVC) |
| GPU support | ❌ | ✓ | ✓ |
| Complexity | Low | Medium | High |

### Decision Guide

```
Start here:
    │
    ├─► Need simple, predictable pricing?
    │       └─► LightSail
    │
    ├─► Need auto-scaling for variable traffic?
    │       └─► ECS Fargate
    │
    ├─► Deploying AI agents to AWS Bedrock?
    │       └─► AgentCore
    │
    ├─► Have existing Kubernetes cluster?
    │       └─► Kubernetes
    │
    └─► Want simple PaaS outside AWS?
            └─► DigitalOcean
```

## Using Targets

Specify target with `--target` flag:

```bash
# LightSail (default)
omnideploy up --config deploy.yaml --target lightsail

# ECS
omnideploy up --config deploy.yaml --target ecs

# Kubernetes
omnideploy up --config deploy.yaml --target kubernetes
```

## Target-Specific Configuration

Some config fields are target-specific:

```yaml
# LightSail-specific
resources:
  size: micro  # nano, micro, small, medium, large, xlarge

# ECS-specific
resources:
  cpu: 256     # Fargate CPU units
  memory: 512  # Fargate memory MB

# Kubernetes-specific
resources:
  cpu: 250m    # Kubernetes CPU request
  memory: 256Mi # Kubernetes memory request
```

## List Available Targets

```bash
omnideploy targets
```

Output:

```
Available targets:
  lightsail    AWS LightSail Container Service - simple, cost-effective container hosting
```
