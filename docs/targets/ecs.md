# AWS ECS Target

!!! note "Coming Soon"
    The ECS target is planned for a future release.

## Overview

AWS ECS (Elastic Container Service) with Fargate provides:

- Serverless container execution
- Auto-scaling based on metrics
- Integration with ALB/NLB
- VPC networking
- EFS for persistent storage

## Planned Features

- ECS Fargate deployment
- Auto-scaling configuration
- ALB integration
- EFS volume mounts
- VPC configuration
- Service discovery

## Configuration Preview

```yaml
name: my-api
region: us-east-1

container:
  image: ghcr.io/myorg/my-api:latest
  ports:
    - container_port: 8080
      protocol: HTTP
  health_check:
    path: /health

service:
  replicas: 2
  public: true
  autoscaling:
    min: 2
    max: 10
    target_cpu: 70

resources:
  cpu: 512     # 0.5 vCPU
  memory: 1024 # 1 GB

# ECS-specific
ecs:
  cluster: my-cluster
  subnets:
    - subnet-123
    - subnet-456
  security_groups:
    - sg-789
```

## Comparison with LightSail

| Feature | LightSail | ECS Fargate |
|---------|-----------|-------------|
| Pricing | Fixed monthly | Pay per use |
| Auto-scaling | Manual | Automatic |
| Persistent storage | No | Yes (EFS) |
| Load balancer | Built-in | ALB/NLB |
| Complexity | Low | Medium |
| VPC required | No | Yes |

## When to Use

**Choose ECS when you need:**

- Auto-scaling for variable traffic
- Persistent storage
- Custom networking
- Fine-grained resource control
- Integration with other AWS services

**Choose LightSail when you need:**

- Simple, predictable pricing
- Quick setup
- Low operational overhead
