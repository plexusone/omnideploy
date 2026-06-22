# AWS AgentCore Target

!!! note "Coming Soon"
    The AgentCore target is planned for a future release.

## Overview

AWS Bedrock AgentCore provides managed infrastructure for AI agents:

- Optimized for agent workloads
- Built-in observability
- Agent-specific scaling
- Integration with Bedrock models

## Planned Features

- AgentCore runtime deployment
- Agent configuration
- Tool registration
- Session management
- Observability integration

## Configuration Preview

```yaml
name: my-agent
region: us-east-1

container:
  image: ghcr.io/myorg/my-agent:latest
  ports:
    - container_port: 8080
      protocol: HTTP

service:
  replicas: 1
  public: true

# AgentCore-specific
agentcore:
  runtime: python3.11
  memory: 2048
  timeout: 300
  tools:
    - web_search
    - calculator
  observability:
    tracing: true
    metrics: true
```

## When to Use

**Choose AgentCore when:**

- Deploying AI agents to AWS
- Need agent-specific scaling
- Want managed agent infrastructure
- Using Bedrock models
