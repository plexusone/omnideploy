# CLI Reference

Complete reference for omnideploy commands.

## Global Flags

These flags work with all commands:

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--config` | `-c` | Config file path | - |
| `--target` | `-t` | Deployment target | `lightsail` |
| `--backend` | `-b` | IaC backend | `pulumi` |
| `--runtime` | `-r` | Runtime adapter | auto-detect |
| `--stack` | `-s` | Stack name | config name |
| `--help` | `-h` | Show help | - |

## Commands

### omnideploy up

Deploy or update a stack.

```bash
omnideploy up [flags]
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--yes` | `-y` | Auto-approve changes |

**Examples:**

```bash
# Deploy with defaults (LightSail + Pulumi)
omnideploy up --config deploy.yaml

# Deploy to specific stack
omnideploy up --config deploy.yaml --stack production

# Auto-approve (no confirmation)
omnideploy up --config deploy.yaml --yes

# Use different target
omnideploy up --config deploy.yaml --target ecs

# Use different backend
omnideploy up --config deploy.yaml --backend terraform

# Specify runtime adapter
omnideploy up --config omniagent.yaml --runtime omniagent
```

---

### omnideploy preview

Preview changes without deploying.

```bash
omnideploy preview [flags]
```

**Examples:**

```bash
# Preview deployment
omnideploy preview --config deploy.yaml

# Preview for specific target
omnideploy preview --config deploy.yaml --target ecs
```

**Output:**

```
Previewing deployment to lightsail using pulumi...

Changes:
  + aws:lightsail:ContainerService my-app
  + aws:lightsail:ContainerServiceDeploymentVersion my-app-deployment

Changes: 2 create, 0 update, 0 delete
```

---

### omnideploy destroy

Destroy all resources in a stack.

```bash
omnideploy destroy [flags]
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--yes` | `-y` | Auto-approve destruction |

**Examples:**

```bash
# Destroy with confirmation prompt
omnideploy destroy --stack my-app

# Destroy without confirmation
omnideploy destroy --stack my-app --yes
```

---

### omnideploy targets

List available deployment targets.

```bash
omnideploy targets
```

**Output:**

```
Available targets:
  lightsail    AWS LightSail Container Service - simple, cost-effective container hosting
```

---

### omnideploy backends

List available IaC backends.

```bash
omnideploy backends
```

**Output:**

```
Available backends:
  pulumi       Pulumi - Infrastructure as Code using Go
```

---

### omnideploy runtimes

List available runtime adapters.

```bash
omnideploy runtimes
```

**Output:**

```
Available runtime adapters:
  omniagent    OmniAgent gateway with web UI
  container    Generic container deployment
```

---

### omnideploy version

Show version information.

```bash
omnideploy version
```

---

### omnideploy completion

Generate shell completion scripts.

```bash
# Bash
omnideploy completion bash > /etc/bash_completion.d/omnideploy

# Zsh
omnideploy completion zsh > "${fpath[1]}/_omnideploy"

# Fish
omnideploy completion fish > ~/.config/fish/completions/omnideploy.fish

# PowerShell
omnideploy completion powershell > omnideploy.ps1
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `AWS_ACCESS_KEY_ID` | AWS access key |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key |
| `AWS_REGION` | Default AWS region |
| `OMNIDEPLOY_WORK_DIR` | Working directory for state |
| `PULUMI_BACKEND_URL` | Pulumi remote state URL |
| `PULUMI_ACCESS_TOKEN` | Pulumi Cloud token |

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error |
| 3 | Deployment error |

## Configuration File Discovery

OmniDeploy looks for config files in this order:

1. Explicit `--config` flag
2. `deploy.yaml` in current directory
3. `deploy.yml` in current directory
4. `omnideploy.yaml` in current directory

## Runtime Auto-Detection

If `--runtime` is not specified, omnideploy tries to detect:

1. Check filename (e.g., `omniagent.yaml` → `omniagent` adapter)
2. Check file contents for known patterns
3. Fall back to `container` adapter

## Examples

### Full Deployment Workflow

```bash
# Preview
omnideploy preview --config deploy.yaml

# Deploy to staging
omnideploy up --config deploy.yaml --stack staging --yes

# Deploy to production
omnideploy up --config deploy.yaml --stack production --yes

# Destroy staging when done
omnideploy destroy --stack staging --yes
```

### Multi-Environment

```bash
# Development
omnideploy up --config deploy.dev.yaml --stack dev

# Staging
omnideploy up --config deploy.staging.yaml --stack staging

# Production
omnideploy up --config deploy.prod.yaml --stack production
```

### Different Targets

```bash
# Simple deployment to LightSail
omnideploy up --config deploy.yaml --target lightsail

# Production to ECS
omnideploy up --config deploy.yaml --target ecs

# Deploy to existing Kubernetes
omnideploy up --config deploy.yaml --target kubernetes
```
