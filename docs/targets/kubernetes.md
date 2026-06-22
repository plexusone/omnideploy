# Kubernetes Target

!!! note "Coming Soon"
    The Kubernetes target is planned for a future release.

## Overview

Deploy to any Kubernetes cluster:

- EKS, GKE, AKS, or self-managed
- Helm chart generation
- Ingress configuration
- Horizontal Pod Autoscaling
- Persistent Volume Claims

## Planned Features

- Deployment resource generation
- Service and Ingress configuration
- ConfigMap and Secret management
- HPA configuration
- PVC for persistent storage
- Helm chart output

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
  replicas: 3
  public: true

resources:
  cpu: 500m
  memory: 512Mi

# Kubernetes-specific
kubernetes:
  namespace: production
  ingress:
    class: nginx
    host: api.example.com
    tls:
      secret: api-tls
  autoscaling:
    min: 3
    max: 10
    target_cpu: 70
  volumes:
    - name: data
      size: 10Gi
      mount: /data
```

## When to Use

**Choose Kubernetes when:**

- You have existing K8s infrastructure
- Need complex orchestration
- Want cloud-agnostic deployments
- Require advanced networking
