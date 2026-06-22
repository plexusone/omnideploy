# Runtime Adapters

Runtime adapters translate application-specific configs to deployment configs.

## Available Adapters

| Adapter | Status | Description |
|---------|--------|-------------|
| [OmniAgent](omniagent.md) | ✓ Available | OmniAgent gateway applications |
| [AgentKit](agentkit.md) | ◐ Planned | AgentKit-based agents |
| [Container](container.md) | ✓ Available | Generic container deployments |

## How Adapters Work

```
┌─────────────────────────────────────────────┐
│          Your Config File                    │
│                                             │
│  ┌─────────────┐  ┌─────────────────────┐  │
│  │ omniagent.  │  │ deploy.yaml         │  │
│  │ yaml        │  │ (generic)           │  │
│  └──────┬──────┘  └──────────┬──────────┘  │
│         │                    │              │
└─────────┼────────────────────┼──────────────┘
          │                    │
          ▼                    ▼
   ┌─────────────┐      ┌─────────────┐
   │  OmniAgent  │      │  Container  │
   │   Adapter   │      │   Adapter   │
   └──────┬──────┘      └──────┬──────┘
          │                    │
          └────────┬───────────┘
                   │
                   ▼
          ┌─────────────────┐
          │  DeployConfig   │
          │  (universal)    │
          └─────────────────┘
```

## Using Adapters

### Auto-Detection

OmniDeploy auto-detects the appropriate adapter:

```bash
# Detected as omniagent (filename)
omnideploy up --config omniagent.yaml

# Detected as omniagent (content)
omnideploy up --config my-agent.yaml

# Detected as container (generic deploy format)
omnideploy up --config deploy.yaml
```

### Explicit Selection

Specify adapter with `--runtime`:

```bash
omnideploy up --config config.yaml --runtime omniagent
omnideploy up --config config.yaml --runtime container
```

## List Available Adapters

```bash
omnideploy runtimes
```

Output:

```
Available runtime adapters:
  omniagent    OmniAgent gateway with web UI
  container    Generic container deployment
```

## Creating Custom Adapters

See [Extending OmniDeploy](../reference/extending.md) for creating custom runtime adapters.
