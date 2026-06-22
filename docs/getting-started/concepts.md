# Concepts

Understanding how OmniDeploy works.

## The Three Pillars

OmniDeploy separates deployment into three independent concerns:

### 1. Targets (WHERE to deploy)

A **target** defines the cloud platform and service where your container runs.

| Target | Description | Best For |
|--------|-------------|----------|
| `lightsail` | AWS LightSail Container Service | Simple apps, cost-effective |
| `ecs` | AWS ECS with Fargate | Production workloads, auto-scaling |
| `agentcore` | AWS Bedrock AgentCore | AI agent deployments |
| `kubernetes` | Any Kubernetes cluster | Complex orchestration |
| `digitalocean` | DigitalOcean App Platform | Simple PaaS experience |

Each target understands how to translate your deployment config into the specific resources needed.

### 2. Backends (HOW to provision)

A **backend** is the Infrastructure as Code (IaC) tool that creates and manages cloud resources.

| Backend | Description | State Storage |
|---------|-------------|---------------|
| `pulumi` | Pulumi Automation API | Local or cloud |
| `cdk` | AWS Cloud Development Kit | CloudFormation |
| `terraform` | Terraform HCL generation | Local or remote |

Backends are interchangeable. You can switch backends without changing your config.

### 3. Runtime Adapters (WHAT to deploy)

A **runtime adapter** converts application-specific configuration to the universal deployment format.

| Adapter | Detects | Extracts |
|---------|---------|----------|
| `omniagent` | OmniAgent configs | Gateway port, agent settings |
| `agentkit` | AgentKit configs | Agent runtime settings |
| `container` | Generic deploy configs | Direct pass-through |

## Configuration Flow

```
┌─────────────────────────────────────────────┐
│          Your Config File                    │
│  (omniagent.yaml, deploy.yaml, etc.)        │
└─────────────────────┬───────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────┐
│          Runtime Adapter                     │
│  Converts to universal DeployConfig         │
└─────────────────────┬───────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────┐
│          DeployConfig                        │
│  name, container, service, resources        │
└─────────────────────┬───────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────┐
│          Target                              │
│  Generates ResourceSpec for the target      │
└─────────────────────┬───────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────┐
│          Backend                             │
│  Provisions resources using IaC             │
└─────────────────────────────────────────────┘
```

## DeployConfig Schema

The universal deployment configuration that all targets understand:

```yaml
# Identity
name: my-app              # Required: deployment name
version: "1.0.0"          # Optional: version tag
region: us-east-1         # Cloud region

# Container settings
container:
  image: nginx:latest     # Container image
  command: []             # Entrypoint override
  args: []                # Command arguments
  ports:                  # Port mappings
    - container_port: 80
      protocol: HTTP
  health_check:           # Health check config
    path: /health
    interval: 30s
    timeout: 5s

# Service settings
service:
  replicas: 1             # Instance count
  public: true            # Public access
  domains: []             # Custom domains

# Resource allocation
resources:
  size: micro             # Preset size

# Environment
environment:              # Key-value pairs
  LOG_LEVEL: info

# Tags
tags:                     # Resource tags
  team: platform
```

## Stacks

A **stack** is an isolated deployment instance. You can have multiple stacks from the same config:

```bash
# Deploy to staging
omnideploy up --config deploy.yaml --stack staging

# Deploy to production
omnideploy up --config deploy.yaml --stack production
```

Each stack maintains its own:

- Cloud resources
- State (tracked by the backend)
- Outputs (URLs, IPs, etc.)

## State Management

OmniDeploy uses the backend's state management:

| Backend | Default State Location | Remote Options |
|---------|----------------------|----------------|
| Pulumi | `~/.omnideploy/pulumi/` | S3, Pulumi Cloud |
| CDK | CloudFormation | N/A (always in AWS) |
| Terraform | `~/.omnideploy/terraform/` | S3, Terraform Cloud |

## Resource Lifecycle

```
preview → up → (running) → destroy
   │       │                  │
   │       │                  └── Removes all resources
   │       └── Creates/updates resources
   └── Shows what would change (dry run)
```

## Best Practices

### 1. Use Named Stacks

Always use explicit stack names for clarity:

```bash
omnideploy up --config deploy.yaml --stack my-app-prod
```

### 2. Preview Before Apply

Always preview changes before deploying:

```bash
omnideploy preview --config deploy.yaml
```

### 3. Use Environment Variables for Secrets

Don't put secrets in config files:

```yaml
environment:
  API_KEY: ${API_KEY}  # Resolved from environment
```

### 4. Tag Resources

Add tags for cost tracking and organization:

```yaml
tags:
  environment: production
  team: platform
  cost-center: engineering
```

### 5. Use Health Checks

Always configure health checks for reliable deployments:

```yaml
container:
  health_check:
    path: /health
    interval: 30s
    timeout: 5s
    healthy_threshold: 2
    unhealthy_threshold: 3
```

## Internal Architecture

OmniDeploy uses different libraries for different operations:

```
┌─────────────────────────────────────────────────────────────────┐
│                        omnideploy CLI                           │
├─────────────────────────────┬───────────────────────────────────┤
│      Bootstrap Commands     │       Deployment Commands         │
│  (bootstrap run/status)     │    (up, preview, destroy)         │
├─────────────────────────────┼───────────────────────────────────┤
│                             │                                   │
│      AWS SDK v2 (Go)        │      Pulumi Automation API        │
│                             │              │                    │
│    ┌─────────────────┐      │    ┌─────────┴─────────┐          │
│    │   IAM Service   │      │    │  Pulumi AWS SDK   │          │
│    └─────────────────┘      │    └─────────┬─────────┘          │
│                             │              │                    │
│                             │      AWS SDK (internal)           │
└─────────────────────────────┴───────────────────────────────────┘
                              │
                              ▼
                         AWS APIs
```

### Why Two Libraries?

| Command | Library | Reason |
|---------|---------|--------|
| `bootstrap` | **AWS SDK v2** | One-time IAM setup, no state needed |
| `up/preview/destroy` | **Pulumi** | IaC with state management, drift detection, rollback |

### Bootstrap (AWS SDK v2)

The `bootstrap` command uses AWS SDK v2 directly because:

- **Simple operations** - Just create IAM policy, group, user
- **No state needed** - Idempotent, can run multiple times
- **No Pulumi dependency** - Works without Pulumi installed

### Deployments (Pulumi Automation API)

Deployment commands use Pulumi because:

- **State management** - Tracks what resources exist
- **Dependency ordering** - Creates resources in correct order
- **Drift detection** - Knows if resources changed externally
- **Updates** - Can modify existing resources safely
- **Rollback** - Can undo failed deployments

### Credentials Flow

Both libraries read AWS credentials from:

1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. Shared credentials file (`~/.aws/credentials`)
3. AWS config profiles (`AWS_PROFILE`)

## Next Steps

- [Configuration Schema](../configuration/schema.md) - Full configuration reference
- [Targets](../targets/index.md) - Explore deployment targets
- [Backends](../backends/index.md) - Learn about IaC backends
