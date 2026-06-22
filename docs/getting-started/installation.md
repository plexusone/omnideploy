# Installation

## Prerequisites

- **Go 1.24+** (for installation from source)
- **AWS credentials** (for AWS targets)
- **Pulumi CLI** (optional, for advanced Pulumi operations)

## Install from Source

```bash
go install github.com/plexusone/omnideploy/cmd/omnideploy@latest
```

Verify installation:

```bash
omnideploy --help
```

## Install from Binary

Download the latest release from [GitHub Releases](https://github.com/plexusone/omnideploy/releases):

=== "macOS (Apple Silicon)"

    ```bash
    curl -L https://github.com/plexusone/omnideploy/releases/latest/download/omnideploy-darwin-arm64.tar.gz | tar xz
    sudo mv omnideploy /usr/local/bin/
    ```

=== "macOS (Intel)"

    ```bash
    curl -L https://github.com/plexusone/omnideploy/releases/latest/download/omnideploy-darwin-amd64.tar.gz | tar xz
    sudo mv omnideploy /usr/local/bin/
    ```

=== "Linux (x86_64)"

    ```bash
    curl -L https://github.com/plexusone/omnideploy/releases/latest/download/omnideploy-linux-amd64.tar.gz | tar xz
    sudo mv omnideploy /usr/local/bin/
    ```

## AWS Credentials

OmniDeploy needs AWS credentials for AWS targets (LightSail, ECS, AgentCore).

### Option 1: Environment Variables

```bash
export AWS_ACCESS_KEY_ID=AKIA...
export AWS_SECRET_ACCESS_KEY=...
export AWS_REGION=us-east-1
```

### Option 2: AWS CLI Profile

```bash
aws configure
# or for a named profile
aws configure --profile omnideploy
export AWS_PROFILE=omnideploy
```

### Option 3: IAM Role (EC2/ECS)

If running on AWS infrastructure, use an IAM instance role or task role.

## Pulumi Setup (Optional)

For the Pulumi backend, you can optionally install the Pulumi CLI:

```bash
# macOS
brew install pulumi

# Linux
curl -fsSL https://get.pulumi.com | sh
```

OmniDeploy uses Pulumi's Automation API and doesn't require the CLI for basic operations, but the CLI is useful for:

- Viewing stack state: `pulumi stack`
- Manual state management: `pulumi state`
- Stack export/import

### Pulumi State Storage

By default, OmniDeploy stores Pulumi state locally at `~/.omnideploy/pulumi/`.

For team collaboration, configure a remote backend:

```bash
# AWS S3
export PULUMI_BACKEND_URL=s3://my-bucket/pulumi-state

# Pulumi Cloud
pulumi login
```

## Verify Installation

Check that everything is working:

```bash
# Show version
omnideploy version

# List available targets
omnideploy targets

# List available backends
omnideploy backends

# List runtime adapters
omnideploy runtimes
```

Expected output:

```
Available targets:
  lightsail    AWS LightSail Container Service - simple, cost-effective container hosting

Available backends:
  pulumi       Pulumi - Infrastructure as Code using Go

Available runtime adapters:
  omniagent    OmniAgent gateway with web UI
  container    Generic container deployment
```

## Next Steps

- [Quick Start](quickstart.md) - Deploy your first application
- [Concepts](concepts.md) - Understand how OmniDeploy works
