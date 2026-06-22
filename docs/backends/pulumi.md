# Pulumi Backend

Infrastructure as Code using Pulumi's Automation API.

## Overview

The Pulumi backend uses Pulumi's Go SDK and Automation API to:

- Create and manage cloud resources
- Track state changes
- Preview deployments
- Handle rollbacks

## Prerequisites

- AWS credentials configured
- (Optional) Pulumi CLI for advanced operations

## State Storage

### Local State (Default)

State is stored at `~/.omnideploy/pulumi/`.

```bash
ls ~/.omnideploy/pulumi/
# Pulumi.yaml, Pulumi.my-app.yaml, etc.
```

### AWS S3 Backend

For team collaboration:

```bash
export PULUMI_BACKEND_URL=s3://my-bucket/omnideploy-state

# Then deploy
omnideploy up --config deploy.yaml
```

### Pulumi Cloud

For managed state:

```bash
# Install Pulumi CLI
brew install pulumi

# Login to Pulumi Cloud
pulumi login

# Deploy (uses Pulumi Cloud for state)
omnideploy up --config deploy.yaml
```

## Usage

### Deploy

```bash
omnideploy up --config deploy.yaml --backend pulumi
```

### Preview

```bash
omnideploy preview --config deploy.yaml --backend pulumi
```

### Destroy

```bash
omnideploy destroy --stack my-app --backend pulumi
```

### Refresh State

Sync state with actual cloud resources:

```bash
omnideploy refresh --stack my-app
```

## Advanced Operations

For operations not available in omnideploy, use Pulumi CLI:

### View Stack State

```bash
cd ~/.omnideploy/pulumi
pulumi stack select my-app
pulumi stack
```

### Export State

```bash
pulumi stack export > state.json
```

### Import Resources

```bash
pulumi import aws:lightsail/containerService:ContainerService my-app my-app
```

### View History

```bash
pulumi history
```

## Configuration

### Stack Configuration

Set stack-specific config:

```bash
cd ~/.omnideploy/pulumi
pulumi stack select my-app
pulumi config set aws:region us-west-2
```

### Secrets

Store secrets securely:

```bash
pulumi config set --secret db_password mypassword
```

Access in code:

```go
password := cfg.RequireSecret("db_password")
```

## Troubleshooting

### State Lock

If deployment fails mid-way:

```bash
cd ~/.omnideploy/pulumi
pulumi cancel
```

### Corrupt State

Export, fix, and import:

```bash
pulumi stack export > state.json
# Edit state.json
pulumi stack import < state.json
```

### Resource Drift

Refresh to detect drift:

```bash
omnideploy refresh --stack my-app
```

Then re-deploy to fix:

```bash
omnideploy up --config deploy.yaml --stack my-app
```

## CI/CD Integration

### GitHub Actions

```yaml
- name: Setup Pulumi
  uses: pulumi/actions@v5

- name: Deploy
  env:
    PULUMI_ACCESS_TOKEN: ${{ secrets.PULUMI_ACCESS_TOKEN }}
    AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
    AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
  run: omnideploy up --config deploy.yaml --yes
```

### With S3 Backend

```yaml
- name: Deploy
  env:
    PULUMI_BACKEND_URL: s3://${{ secrets.STATE_BUCKET }}/omnideploy
    AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
    AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
  run: omnideploy up --config deploy.yaml --yes
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `PULUMI_BACKEND_URL` | Remote state URL (s3://, gs://, azblob://) |
| `PULUMI_ACCESS_TOKEN` | Pulumi Cloud access token |
| `PULUMI_CONFIG_PASSPHRASE` | Encryption passphrase for local secrets |

## Supported Targets

The Pulumi backend supports:

- ✓ AWS LightSail
- ◐ AWS ECS (planned)
- ◐ AWS AgentCore (planned)
- ◐ Kubernetes (planned)
- ◐ DigitalOcean (planned)

## Next Steps

- [Local Deployment](../deployment/local.md) - Deploy from your machine
- [GitHub Actions](../deployment/github-actions.md) - CI/CD setup
