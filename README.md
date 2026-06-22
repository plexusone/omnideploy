# OmniDeploy

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Docs][docs-mkdoc-svg]][docs-mkdoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/plexusone/omnideploy/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/plexusone/omnideploy/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/plexusone/omnideploy/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/plexusone/omnideploy/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/plexusone/omnideploy/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/plexusone/omnideploy/actions/workflows/go-sast-codeql.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/plexusone/omnideploy
 [goreport-url]: https://goreportcard.com/report/github.com/plexusone/omnideploy
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/omnideploy
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/omnideploy
 [docs-mkdoc-svg]: https://img.shields.io/badge/Go-dev%20guide-blue.svg
 [docs-mkdoc-url]: https://plexusone.dev/omnideploy
 [viz-svg]: https://img.shields.io/badge/Go-visualizaton-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=plexusone%2Fomnideploy
 [loc-svg]: https://tokei.rs/b1/github/plexusone/omnideploy
 [repo-url]: https://github.com/plexusone/omnideploy
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/omnideploy/blob/main/LICENSE

Universal deployment tool for container applications. Deploy to any cloud provider using any IaC tool.

## Overview

OmniDeploy separates **where** you deploy (targets) from **how** you provision (backends):

```
                    Backend (HOW to provision)
                    ┌─────────┬─────────┬───────────┐
                    │ Pulumi  │   CDK   │ Terraform │
        ┌───────────┼─────────┼─────────┼───────────┤
        │ LightSail │    ✓    │    -    │     -     │
Target  │ ECS       │    -    │    -    │     -     │
(WHERE) │ AgentCore │    -    │    -    │     -     │
        │ Kubernetes│    -    │    -    │     -     │
        └───────────┴─────────┴─────────┴───────────┘
```

## Installation

```bash
go install github.com/plexusone/omnideploy/cmd/omnideploy@latest
```

## Quick Start

### Deploy OmniAgent to AWS LightSail

```bash
# Using OmniAgent config
omnideploy up \
  --config omniagent.yaml \
  --target lightsail \
  --backend pulumi \
  --stack prod

# Preview changes first
omnideploy preview --config omniagent.yaml

# Destroy when done
omnideploy destroy --stack prod --yes
```

### Generic Container Deployment

Create a `deploy.yaml`:

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
  size: micro  # nano, micro, small, medium, large, xlarge
```

Deploy:

```bash
omnideploy up --config deploy.yaml
```

## Configuration

### OmniAgent Configuration

OmniDeploy auto-detects OmniAgent configurations and extracts deployment settings:

```yaml
# omniagent.yaml
gateway:
  address: "0.0.0.0:18789"

agent:
  provider: anthropic
  model: claude-sonnet-4-20250514
  api_key: ${ANTHROPIC_API_KEY}

# Deployment settings (omnideploy-specific)
deploy:
  name: omniagent-prod
  region: us-east-1
  image: ghcr.io/plexusone/omniagent:latest
  replicas: 1
  resources:
    size: micro
  environment:
    LOG_LEVEL: info
```

### Generic Configuration

Full configuration reference:

```yaml
name: my-service           # Required: deployment name
version: "1.0.0"           # Optional: version tag
region: us-east-1          # AWS region

container:
  image: nginx:latest      # Required: container image
  command: []              # Optional: entrypoint override
  args: []                 # Optional: command arguments
  ports:
    - container_port: 80   # Required: port number
      protocol: HTTP       # HTTP, HTTPS, TCP, UDP
      name: http           # Optional: port name
  health_check:
    path: /health          # Health check endpoint
    interval: 30s          # Check interval
    timeout: 5s            # Check timeout
    healthy_threshold: 2   # Consecutive successes needed
    unhealthy_threshold: 3 # Consecutive failures needed

service:
  replicas: 1              # Number of instances
  public: true             # Publicly accessible
  domains:                 # Custom domains
    - api.example.com

resources:
  size: micro              # nano, micro, small, medium, large, xlarge

environment:               # Environment variables
  LOG_LEVEL: info
  DATABASE_URL: ${DB_URL}

tags:                      # Resource tags
  environment: production
  team: platform
```

## CLI Reference

### Commands

```bash
# Deploy or update
omnideploy up --config <file> [--target <target>] [--backend <backend>] [--stack <name>] [--yes]

# Preview changes
omnideploy preview --config <file> [--target <target>] [--backend <backend>]

# Destroy stack
omnideploy destroy --stack <name> [--yes]

# List available components
omnideploy targets    # List deployment targets
omnideploy backends   # List IaC backends
omnideploy runtimes   # List runtime adapters
```

### Global Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--config` | `-c` | Config file path | - |
| `--target` | `-t` | Deployment target | `lightsail` |
| `--backend` | `-b` | IaC backend | `pulumi` |
| `--runtime` | `-r` | Runtime adapter (auto-detected) | - |
| `--stack` | `-s` | Stack name | config name |

## Targets

### AWS LightSail

Cost-effective container hosting with simple scaling.

**Resource Sizes:**

| Size | vCPU | Memory | Price/month |
|------|------|--------|-------------|
| nano | 0.25 | 512 MB | ~$7 |
| micro | 0.5 | 1 GB | ~$10 |
| small | 1 | 2 GB | ~$25 |
| medium | 2 | 4 GB | ~$50 |
| large | 4 | 8 GB | ~$100 |
| xlarge | 8 | 16 GB | ~$200 |

## Backends

### Pulumi

Uses Pulumi Automation API with local state storage.

**Requirements:**
- AWS credentials configured (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
- Pulumi CLI installed (optional, for advanced operations)

**State Location:** `~/.omnideploy/pulumi/`

## Runtime Adapters

### omniagent

Auto-detects OmniAgent configurations and maps:
- Gateway port → container port
- Agent config → environment variables
- Deploy section → resources and replicas

### container

Generic container deployments using the full OmniDeploy config schema.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `AWS_ACCESS_KEY_ID` | AWS access key |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key |
| `AWS_REGION` | Default AWS region |
| `OMNIDEPLOY_WORK_DIR` | Pulumi state directory |

## Extending OmniDeploy

### Adding a Target

```go
package mytarget

import "github.com/plexusone/omnideploy/target"

func init() {
    target.Register(&MyTarget{})
}

type MyTarget struct{}

func (t *MyTarget) Name() string { return "mytarget" }
func (t *MyTarget) Description() string { return "My custom target" }
func (t *MyTarget) Validate(cfg *config.DeployConfig) error { ... }
func (t *MyTarget) ResourceSpec(cfg *config.DeployConfig) (*target.ResourceSpec, error) { ... }
```

### Adding a Backend

```go
package mybackend

import "github.com/plexusone/omnideploy/backend"

func init() {
    backend.Register(&MyBackend{})
}

type MyBackend struct{}

func (b *MyBackend) Name() string { return "mybackend" }
func (b *MyBackend) Apply(ctx context.Context, spec *target.ResourceSpec, opts backend.ApplyOptions) (*backend.Result, error) { ... }
```

## License

MIT
