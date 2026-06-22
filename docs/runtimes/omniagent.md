# OmniAgent Runtime Adapter

Deploy OmniAgent gateway applications.

## Overview

The OmniAgent adapter understands OmniAgent configuration files and extracts:

- Gateway port configuration
- Agent provider and model settings
- Deployment parameters

## Detection

Files are detected as OmniAgent configs when:

1. Filename contains "omniagent"
2. File contains `gateway:` or `agent:` top-level keys

## Configuration

### OmniAgent Config with Deploy Section

```yaml
# omniagent.yaml

# OmniAgent configuration
gateway:
  address: "0.0.0.0:18789"
  api_keys:
    - ${GATEWAY_API_KEY}

agent:
  provider: anthropic
  model: claude-sonnet-4-20250514
  api_key: ${ANTHROPIC_API_KEY}

# Deployment configuration (omnideploy-specific)
deploy:
  name: my-omniagent
  region: us-east-1
  image: ghcr.io/myorg/my-omniagent:latest
  replicas: 1
  resources:
    size: small
  environment:
    LOG_LEVEL: info
```

### What Gets Extracted

| OmniAgent Config | DeployConfig |
|------------------|--------------|
| `gateway.address` port | `container.ports[0].container_port` |
| `deploy.name` | `name` |
| `deploy.image` | `container.image` |
| `deploy.region` | `region` |
| `deploy.replicas` | `service.replicas` |
| `deploy.resources.size` | `resources.size` |
| `deploy.environment` | `environment` |
| `agent.provider` | `environment.OMNIAGENT_AGENT_PROVIDER` |
| `agent.model` | `environment.OMNIAGENT_AGENT_MODEL` |

## Deployment

```bash
# Auto-detect adapter
omnideploy up --config omniagent.yaml

# Explicit adapter
omnideploy up --config my-config.yaml --runtime omniagent
```

## Defaults

If not specified:

| Field | Default |
|-------|---------|
| `deploy.name` | Filename without extension |
| `deploy.image` | `ghcr.io/plexusone/omniagent:latest` |
| `deploy.region` | `us-east-1` |
| `deploy.replicas` | `1` |
| `deploy.resources.size` | `micro` |
| Container port | `18789` (OmniAgent default) |

## Health Check

Auto-configured to:

```yaml
health_check:
  path: /api/health
  port: 18789
  interval: 30s
  timeout: 5s
  healthy_threshold: 2
  unhealthy_threshold: 3
```

## Environment Variables

The adapter automatically sets:

```yaml
environment:
  OMNIAGENT_AGENT_PROVIDER: anthropic  # from agent.provider
  OMNIAGENT_AGENT_MODEL: claude-...    # from agent.model
```

## Secrets Handling

API keys referenced with `${VAR}` are converted to secrets:

```yaml
# In omniagent.yaml
agent:
  api_key: ${ANTHROPIC_API_KEY}

# Results in DeployConfig
environment:
  ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}  # Resolved at deploy time
```

Pass secrets when deploying:

```bash
ANTHROPIC_API_KEY=sk-... omnideploy up --config omniagent.yaml
```

## Example: grokify-omniagent

```yaml
# omniagent.yaml
gateway:
  address: "0.0.0.0:8080"

agent:
  provider: anthropic
  model: claude-sonnet-4-20250514
  api_key: ${ANTHROPIC_API_KEY}

deploy:
  name: grokify-omniagent
  region: us-east-1
  image: ghcr.io/grokify/grokify-omniagent:latest
  replicas: 1
  resources:
    size: small
  environment:
    LOG_LEVEL: info
    TWILIO_ACCOUNT_SID: ${TWILIO_ACCOUNT_SID}
    TWILIO_AUTH_TOKEN: ${TWILIO_AUTH_TOKEN}
```

Deploy:

```bash
export ANTHROPIC_API_KEY=sk-...
export TWILIO_ACCOUNT_SID=AC...
export TWILIO_AUTH_TOKEN=...

omnideploy up --config omniagent.yaml --stack production
```

## Next Steps

- [Container Adapter](container.md) - Generic container deployment
- [Configuration Schema](../configuration/schema.md) - Full config reference
