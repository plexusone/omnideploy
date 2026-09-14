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
# CONTAINER (exactly one of `container` or `instance` — see below)
# =============================================================================

# Used by container-based targets (lightsail, ecs, kubernetes, ...).
# Omit entirely when using `instance` (the lightsail-instance target).
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
# INSTANCE (exactly one of `container` or `instance` — see below)
# =============================================================================

# Only for the lightsail-instance target — a persistent VM instead of a
# container. Omit entirely when using `container`.
instance:
  # Required: Lightsail blueprint ID, e.g. "ubuntu_22_04"
  blueprint: ubuntu_22_04

  # Required: Lightsail bundle ID, e.g. "nano_3_0" — not validated
  # locally (the bundle catalog changes over time); an invalid value
  # surfaces as an AWS API error during `omnideploy up`.
  bundle: nano_3_0

  # Required: local path to the pre-built binary (cross-compiled
  # beforehand, e.g. CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build)
  binary_path: ./bin/my-app

  # Required: path on the instance where the binary is installed and run
  remote_path: /opt/my-app/my-app

  # Required: systemd service name
  service_name: my-app

  # Optional: override the generated systemd unit entirely. The default
  # unit grants ReadWritePaths only on remote_path's own directory —
  # override this to add e.g. a separate /data volume.
  systemd_unit: null

  # Optional: post-deploy health verification. Omit to skip verification
  # entirely (useful for outbound-only workloads like Discord bots).
  health_check:
    kind: http           # "http" or "systemd"
    path: /health         # required when kind is http
    port: 8080            # required when kind is http; also opens this
                           # port in the instance firewall (SSH-only by
                           # default)

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
| `container` | object | Exactly one of `container`/`instance` | - | Container configuration |
| `instance` | object | Exactly one of `container`/`instance` | - | VM instance configuration (lightsail-instance target only) |
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

### Instance Fields

*(lightsail-instance target only)*

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `blueprint` | string | Yes | - | Lightsail blueprint ID (e.g. `ubuntu_22_04`) |
| `bundle` | string | Yes | - | Lightsail bundle ID (e.g. `nano_3_0`); not validated locally |
| `binary_path` | string | Yes | - | Local path to the pre-built binary |
| `remote_path` | string | Yes | - | Path on the instance where the binary is installed |
| `service_name` | string | Yes | - | systemd service name |
| `systemd_unit` | string | No | Generated | Override the generated systemd unit |
| `health_check` | object | No | - | VM health check config |

### VM Health Check Fields

*(lightsail-instance target only — distinct from the container target's `health_check`)*

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `kind` | string | Yes | - | `http` or `systemd` |
| `path` | string | Required when `kind` is `http` | - | HTTP path to probe |
| `port` | int | Required when `kind` is `http` | - | Port to probe; also opened in the instance firewall |

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
- Exactly one of `container`/`instance` must be set
- When using `container`: `container.image` is required, at least one `container.ports` entry is required, `container_port` must be 1-65535, `service.replicas` must be non-negative
- When using `instance`: `blueprint`, `bundle`, `binary_path`, `remote_path`, and `service_name` are all required; if `health_check` is set, `kind` must be `http` or `systemd`, and `http` additionally requires `path` and a valid `port`

Use `omnideploy preview` to catch validation errors before deploying.
