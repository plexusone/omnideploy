# Container Runtime Adapter

Generic container deployment using the OmniDeploy config format.

## Overview

The Container adapter is a pass-through that reads OmniDeploy's native configuration format directly. Use this for any containerized application.

## Detection

Files are detected as container configs when:

1. Filename contains "deploy" or "omnideploy"
2. No application-specific patterns detected

## Configuration

```yaml
name: my-api
version: "1.0.0"
region: us-east-1

container:
  image: nginx:latest
  command: []
  args: []
  ports:
    - container_port: 80
      protocol: HTTP
      name: http
  health_check:
    path: /
    interval: 30s
    timeout: 5s
    healthy_threshold: 2
    unhealthy_threshold: 3

service:
  replicas: 1
  public: true

resources:
  size: micro

environment:
  LOG_LEVEL: info
  DATABASE_URL: ${DATABASE_URL}

tags:
  environment: production
  team: platform
```

## Deployment

```bash
# Auto-detect (default for deploy.yaml)
omnideploy up --config deploy.yaml

# Explicit adapter
omnideploy up --config myconfig.yaml --runtime container
```

## Examples

### Web Application

```yaml
name: web-app
region: us-east-1

container:
  image: ghcr.io/myorg/web-app:v1.2.3
  ports:
    - container_port: 3000
      protocol: HTTP
  health_check:
    path: /health

service:
  replicas: 2
  public: true

resources:
  size: small
```

### API Service

```yaml
name: api-service
region: us-west-2

container:
  image: ghcr.io/myorg/api:latest
  args:
    - --port=8080
    - --workers=4
  ports:
    - container_port: 8080
      protocol: HTTP
  health_check:
    path: /api/health
    interval: 15s
    timeout: 3s

service:
  replicas: 3
  public: true

resources:
  size: medium

environment:
  DATABASE_URL: ${DATABASE_URL}
  REDIS_URL: ${REDIS_URL}
  LOG_LEVEL: info
```

### Background Worker

```yaml
name: worker
region: us-east-1

container:
  image: ghcr.io/myorg/worker:latest
  args:
    - --queue=default
    - --concurrency=10
  ports:
    - container_port: 9090
      protocol: HTTP
      name: metrics
  health_check:
    path: /metrics

service:
  replicas: 2
  public: false  # Internal only

resources:
  size: small

environment:
  REDIS_URL: ${REDIS_URL}
```

## Validations

The container adapter validates:

- `name` is required
- `container.image` is required
- At least one port is required
- Port numbers are 1-65535
- Protocol is HTTP, HTTPS, TCP, or UDP

## Next Steps

- [OmniAgent Adapter](omniagent.md) - For OmniAgent applications
- [Configuration Schema](../configuration/schema.md) - Full config reference
