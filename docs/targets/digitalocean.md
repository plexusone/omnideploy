# DigitalOcean Target

!!! note "Coming Soon"
    The DigitalOcean target is planned for a future release.

## Overview

DigitalOcean App Platform provides:

- Simple PaaS experience
- Automatic builds from Git
- Free SSL certificates
- Global CDN
- Database integration

## Planned Features

- App Platform deployment
- Container registry integration
- Database provisioning
- Domain configuration
- Environment management

## Configuration Preview

```yaml
name: my-api
region: nyc1

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

resources:
  size: basic-xxs  # $5/month

# DigitalOcean-specific
digitalocean:
  instance_size: basic-xxs
  instance_count: 2
  http_port: 8080
  routes:
    - path: /
```

## Pricing

| Size | vCPU | Memory | Cost |
|------|------|--------|------|
| basic-xxs | 1 | 512MB | $5/mo |
| basic-xs | 1 | 1GB | $10/mo |
| basic-s | 1 | 2GB | $20/mo |
| basic-m | 2 | 4GB | $40/mo |

## When to Use

**Choose DigitalOcean when:**

- Want simple PaaS experience
- Cost-sensitive deployments
- Not locked into AWS
- Need quick global deployment
