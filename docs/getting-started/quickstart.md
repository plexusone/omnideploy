# Quick Start

Deploy your first container application to AWS LightSail in under 5 minutes.

## Prerequisites

- [OmniDeploy installed](installation.md)
- AWS credentials configured
- A container image (we'll use nginx for this example)

## Step 1: Create Configuration

Create a `deploy.yaml` file:

```yaml
name: hello-world
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
  size: micro  # ~$10/month
```

## Step 2: Preview Deployment

See what resources will be created:

```bash
omnideploy preview --config deploy.yaml
```

Output:

```
Previewing deployment to lightsail using pulumi...

Changes:
  + aws:lightsail:ContainerService hello-world
  + aws:lightsail:ContainerServiceDeploymentVersion hello-world-deployment

Changes: 2 create, 0 update, 0 delete
```

## Step 3: Deploy

Deploy to AWS LightSail:

```bash
omnideploy up --config deploy.yaml --yes
```

Output:

```
Deploying to lightsail using pulumi...

Updating (hello-world):
     Type                                              Name                    Status
 +   aws:lightsail:ContainerService                   hello-world             created
 +   aws:lightsail:ContainerServiceDeploymentVersion  hello-world-deployment  created

Outputs:
    service_name: "hello-world"
    state       : "RUNNING"
    url         : "https://hello-world.abc123.us-east-1.cs.amazonlightsail.com"

Resources:
    + 2 created

Deployment complete!
```

## Step 4: Access Your App

Open the URL from the output in your browser. You should see the nginx welcome page.

## Step 5: Clean Up

When you're done, destroy the resources:

```bash
omnideploy destroy --stack hello-world --yes
```

## Next: Deploy Your Own App

### Option A: Generic Container

Update the image to your own:

```yaml
name: my-api
region: us-east-1

container:
  image: ghcr.io/myorg/my-api:latest
  ports:
    - container_port: 8080
      protocol: HTTP
  health_check:
    path: /health
    interval: 30s
    timeout: 5s

service:
  replicas: 2
  public: true

resources:
  size: small

environment:
  LOG_LEVEL: info
  DATABASE_URL: ${DATABASE_URL}
```

### Option B: OmniAgent Application

If you have an OmniAgent-based app, omnideploy auto-detects the config:

```yaml
# omniagent.yaml
gateway:
  address: "0.0.0.0:18789"

agent:
  provider: anthropic
  model: claude-sonnet-4-20250514
  api_key: ${ANTHROPIC_API_KEY}

# Deployment settings
deploy:
  name: my-agent
  region: us-east-1
  image: ghcr.io/myorg/my-agent:latest
  replicas: 1
  resources:
    size: small
```

Deploy with:

```bash
omnideploy up --config omniagent.yaml --runtime omniagent
```

## Common Operations

```bash
# Preview changes
omnideploy preview --config deploy.yaml

# Deploy with auto-approve
omnideploy up --config deploy.yaml --yes

# Deploy to a named stack
omnideploy up --config deploy.yaml --stack production

# Use a different target
omnideploy up --config deploy.yaml --target ecs

# Use a different backend
omnideploy up --config deploy.yaml --backend terraform

# Destroy a stack
omnideploy destroy --stack my-app --yes
```

## Next Steps

- [Concepts](concepts.md) - Understand targets, backends, and adapters
- [Configuration Schema](../configuration/schema.md) - Full configuration reference
- [Targets](../targets/index.md) - Explore available deployment targets
