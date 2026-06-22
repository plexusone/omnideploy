# Local Deployment

Deploy to cloud infrastructure from your local machine.

## Prerequisites

1. [OmniDeploy installed](../getting-started/installation.md)
2. Cloud credentials configured
3. Container image built and pushed to a registry

## Workflow

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Build &    │ ──► │  Push to    │ ──► │  Deploy     │
│  Test       │     │  Registry   │     │  with       │
│  Locally    │     │  (GHCR,ECR) │     │  omnideploy │
└─────────────┘     └─────────────┘     └─────────────┘
```

## Step 1: Build Container Image

```bash
# Build image
docker build -t my-app:latest .

# Tag for registry
docker tag my-app:latest ghcr.io/myorg/my-app:latest

# Or with version
docker tag my-app:latest ghcr.io/myorg/my-app:v1.0.0
```

## Step 2: Push to Registry

=== "GitHub Container Registry"

    ```bash
    # Login to GHCR
    echo $GITHUB_TOKEN | docker login ghcr.io -u $GITHUB_USER --password-stdin

    # Push image
    docker push ghcr.io/myorg/my-app:latest
    ```

=== "Amazon ECR"

    ```bash
    # Login to ECR
    aws ecr get-login-password --region us-east-1 | \
      docker login --username AWS --password-stdin 123456789.dkr.ecr.us-east-1.amazonaws.com

    # Create repository (first time)
    aws ecr create-repository --repository-name my-app

    # Tag and push
    docker tag my-app:latest 123456789.dkr.ecr.us-east-1.amazonaws.com/my-app:latest
    docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/my-app:latest
    ```

=== "Docker Hub"

    ```bash
    # Login
    docker login -u $DOCKER_USER

    # Push
    docker push myorg/my-app:latest
    ```

## Step 3: Configure Deployment

Create `deploy.yaml`:

```yaml
name: my-app
region: us-east-1

container:
  image: ghcr.io/myorg/my-app:latest
  ports:
    - container_port: 8080
      protocol: HTTP
  health_check:
    path: /health
    interval: 30s

service:
  replicas: 1
  public: true

resources:
  size: small

environment:
  LOG_LEVEL: info
```

## Step 4: Set Cloud Credentials

=== "AWS"

    ```bash
    export AWS_ACCESS_KEY_ID=AKIA...
    export AWS_SECRET_ACCESS_KEY=...
    export AWS_REGION=us-east-1
    ```

=== "DigitalOcean"

    ```bash
    export DIGITALOCEAN_TOKEN=...
    ```

## Step 5: Preview Changes

Always preview before deploying:

```bash
omnideploy preview --config deploy.yaml
```

Output:

```
Previewing deployment to lightsail using pulumi...

Changes:
  + aws:lightsail:ContainerService my-app
  + aws:lightsail:ContainerServiceDeploymentVersion my-app-deployment

Changes: 2 create, 0 update, 0 delete
```

## Step 6: Deploy

```bash
# Deploy with confirmation prompt
omnideploy up --config deploy.yaml

# Deploy with auto-approve
omnideploy up --config deploy.yaml --yes

# Deploy to named stack
omnideploy up --config deploy.yaml --stack production
```

## Step 7: Verify Deployment

The output includes the service URL:

```
Outputs:
    service_name: "my-app"
    state       : "RUNNING"
    url         : "https://my-app.abc123.us-east-1.cs.amazonlightsail.com"
```

Test the endpoint:

```bash
curl https://my-app.abc123.us-east-1.cs.amazonlightsail.com/health
```

## Updating Deployments

To update a running deployment:

1. Build and push a new image:
   ```bash
   docker build -t ghcr.io/myorg/my-app:v1.1.0 .
   docker push ghcr.io/myorg/my-app:v1.1.0
   ```

2. Update `deploy.yaml` with new image tag:
   ```yaml
   container:
     image: ghcr.io/myorg/my-app:v1.1.0
   ```

3. Deploy the update:
   ```bash
   omnideploy up --config deploy.yaml --stack production
   ```

## Managing Secrets

### Option 1: Environment Variables

Pass secrets via environment when deploying:

```bash
ANTHROPIC_API_KEY=sk-... omnideploy up --config deploy.yaml
```

Reference in config:

```yaml
environment:
  ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
```

### Option 2: Pulumi Config

Store secrets in Pulumi config:

```bash
cd ~/.omnideploy/pulumi
pulumi config set --secret api_key sk-...
```

### Option 3: AWS Secrets Manager

Reference secrets from AWS:

```yaml
secrets:
  - name: ANTHROPIC_API_KEY
    source: secretsmanager:my-app/anthropic-api-key
```

## Multi-Environment Deployments

Create separate configs or use stacks:

### Option A: Separate Config Files

```bash
# deploy.staging.yaml
omnideploy up --config deploy.staging.yaml --stack staging

# deploy.prod.yaml
omnideploy up --config deploy.prod.yaml --stack production
```

### Option B: Same Config, Different Stacks

```bash
# Staging (smaller resources)
REPLICAS=1 SIZE=micro omnideploy up --config deploy.yaml --stack staging

# Production (larger resources)
REPLICAS=3 SIZE=medium omnideploy up --config deploy.yaml --stack production
```

## Troubleshooting

### View Pulumi State

```bash
cd ~/.omnideploy/pulumi
pulumi stack ls
pulumi stack select my-app
pulumi stack
```

### View Deployment Logs

For LightSail:

```bash
aws lightsail get-container-log \
  --service-name my-app \
  --container-name my-app
```

### Force Refresh State

```bash
omnideploy refresh --stack my-app
```

### Destroy and Recreate

```bash
omnideploy destroy --stack my-app --yes
omnideploy up --config deploy.yaml --stack my-app
```

## Next Steps

- [GitHub Actions](github-actions.md) - Automate deployments with CI/CD
- [Continuous Deployment](continuous.md) - Set up automatic deployments
