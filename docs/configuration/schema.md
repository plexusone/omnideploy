# Configuration Schema

Complete reference for OmniDeploy configuration files.

## Overview

OmniDeploy uses YAML or JSON configuration files. The schema is designed to be cloud-agnostic while supporting target-specific features.

## Full Schema

```yaml
# =============================================================================
# IDENTITY
# =============================================================================

# Required: Unique name for this deployment
name: my-app

# Optional: Version tag (used for image tagging and tracking)
version: "1.0.0"

# Optional: Cloud region (default: us-east-1)
region: us-east-1

# =============================================================================
# CONTAINER
# =============================================================================

container:
  # Required: Container image URL
  image: nginx:latest

  # Optional: Override container entrypoint
  command:
    - /bin/sh
    - -c

  # Optional: Arguments to entrypoint
  args:
    - "nginx -g 'daemon off;'"

  # Optional: Working directory inside container
  working_dir: /app

  # Required: At least one port
  ports:
    - container_port: 80      # Required: Port number (1-65535)
      protocol: HTTP          # HTTP, HTTPS, TCP, UDP (default: TCP)
      name: http              # Optional: Port name

  # Optional: Health check configuration
  health_check:
    path: /health             # HTTP path to check
    port: 80                  # Port to check (default: first port)
    interval: 30s             # Time between checks (default: 30s)
    timeout: 5s               # Timeout per check (default: 5s)
    healthy_threshold: 2      # Consecutive successes (default: 2)
    unhealthy_threshold: 3    # Consecutive failures (default: 3)

# =============================================================================
# SERVICE
# =============================================================================

service:
  # Optional: Number of container instances (default: 1)
  replicas: 1

  # Optional: Make service publicly accessible (default: false)
  public: true

  # Optional: Custom domain names
  domains:
    - api.example.com
    - www.example.com

  # Optional: TLS/HTTPS configuration
  tls:
    enabled: true
    certificate_arn: arn:aws:acm:...  # AWS-specific
    auto_cert: false                   # Auto-provision certificate

# =============================================================================
# RESOURCES
# =============================================================================

resources:
  # Optional: Preset size (target-specific)
  # LightSail: nano, micro, small, medium, large, xlarge
  # ECS: Defined by cpu/memory below
  size: micro

  # Optional: CPU in millicores (e.g., 256 = 0.25 vCPU)
  cpu: 256

  # Optional: Memory in MB
  memory: 512

# =============================================================================
# ENVIRONMENT
# =============================================================================

# Optional: Environment variables (key-value pairs)
environment:
  LOG_LEVEL: info
  DATABASE_URL: postgres://...

  # Environment variable expansion
  API_KEY: ${API_KEY}           # From shell environment
  HOME_DIR: ${HOME:-/root}      # With default value

# =============================================================================
# SECRETS
# =============================================================================

# Optional: Secret references (target-specific)
secrets:
  - name: DATABASE_PASSWORD     # Env var name in container
    source: ssm:/my-app/db-pass # Source (ssm:, secretsmanager:, vault:)

  - name: API_KEY
    source: secretsmanager:my-app/api-key

# =============================================================================
# TAGS
# =============================================================================

# Optional: Resource tags (for cost tracking, organization)
tags:
  environment: production
  team: platform
  cost-center: engineering
  managed-by: omnideploy
```

## Field Reference

### Top-Level Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | - | Deployment name (alphanumeric, hyphens) |
| `version` | string | No | - | Version string for tracking |
| `region` | string | No | `us-east-1` | Cloud region |
| `container` | object | Yes | - | Container configuration |
| `service` | object | No | - | Service configuration |
| `resources` | object | No | - | Resource allocation |
| `environment` | map | No | - | Environment variables |
| `secrets` | array | No | - | Secret references |
| `tags` | map | No | - | Resource tags |

### Container Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `image` | string | Yes | - | Container image URL |
| `command` | array | No | - | Entrypoint override |
| `args` | array | No | - | Command arguments |
| `working_dir` | string | No | - | Working directory |
| `ports` | array | Yes | - | Port mappings (at least one) |
| `health_check` | object | No | - | Health check config |

### Port Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `container_port` | int | Yes | - | Port number (1-65535) |
| `protocol` | string | No | `TCP` | HTTP, HTTPS, TCP, UDP |
| `name` | string | No | - | Port name for reference |

### Health Check Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `path` | string | No | - | HTTP path (e.g., `/health`) |
| `port` | int | No | First port | Port to check |
| `interval` | duration | No | `30s` | Time between checks |
| `timeout` | duration | No | `5s` | Check timeout |
| `healthy_threshold` | int | No | `2` | Consecutive successes |
| `unhealthy_threshold` | int | No | `3` | Consecutive failures |

### Service Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `replicas` | int | No | `1` | Number of instances |
| `public` | bool | No | `false` | Public accessibility |
| `domains` | array | No | - | Custom domain names |
| `tls` | object | No | - | TLS configuration |

### Resource Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `size` | string | No | `micro` | Preset size |
| `cpu` | int | No | - | CPU in millicores |
| `memory` | int | No | - | Memory in MB |

## Environment Variable Expansion

Environment variables support shell-style expansion:

```yaml
environment:
  # Direct reference
  API_KEY: ${API_KEY}

  # With default value
  LOG_LEVEL: ${LOG_LEVEL:-info}

  # Literal dollar sign (escape with double)
  PRICE: $$100
```

## Duration Format

Duration fields accept Go-style duration strings:

- `30s` - 30 seconds
- `5m` - 5 minutes
- `1h` - 1 hour
- `1h30m` - 1 hour 30 minutes

## Target-Specific Sizes

### AWS LightSail

| Size | vCPU | Memory | Est. Cost |
|------|------|--------|-----------|
| `nano` | 0.25 | 512 MB | ~$7/mo |
| `micro` | 0.5 | 1 GB | ~$10/mo |
| `small` | 1 | 2 GB | ~$25/mo |
| `medium` | 2 | 4 GB | ~$50/mo |
| `large` | 4 | 8 GB | ~$100/mo |
| `xlarge` | 8 | 16 GB | ~$200/mo |

### AWS ECS

Use `cpu` and `memory` fields directly:

```yaml
resources:
  cpu: 256    # 0.25 vCPU
  memory: 512 # 512 MB
```

## Validation

OmniDeploy validates configurations before deployment:

- `name` must be alphanumeric with hyphens
- `container.image` is required
- At least one `container.ports` entry is required
- `container_port` must be 1-65535
- `service.replicas` must be non-negative

Use `omnideploy preview` to catch validation errors before deploying.
