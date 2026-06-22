# AgentKit Runtime Adapter

!!! note "Coming Soon"
    The AgentKit adapter is planned for a future release.

## Overview

The AgentKit adapter understands AgentKit agent configurations:

- Agent runtime settings
- Tool configurations
- LLM provider settings
- Deployment parameters

## Planned Detection

Files detected as AgentKit configs when:

1. Filename contains "agentkit"
2. File contains AgentKit-specific keys

## Configuration Preview

```yaml
# agentkit.yaml

# AgentKit configuration
runtime:
  name: my-agent
  version: "1.0.0"

llm:
  provider: anthropic
  model: claude-sonnet-4-20250514

tools:
  - name: web_search
    enabled: true
  - name: calculator
    enabled: true

# Deployment
deploy:
  name: my-agentkit-agent
  region: us-east-1
  image: ghcr.io/myorg/my-agent:latest
  replicas: 1
  resources:
    size: small
```

## What Will Be Extracted

| AgentKit Config | DeployConfig |
|-----------------|--------------|
| `deploy.name` | `name` |
| `deploy.image` | `container.image` |
| `deploy.region` | `region` |
| `llm.provider` | `environment.LLM_PROVIDER` |
| `llm.model` | `environment.LLM_MODEL` |
