# IaC Backends

Backends define **how** cloud resources are provisioned.

## Available Backends

| Backend | Status | Description | State Storage |
|---------|--------|-------------|---------------|
| [Pulumi](pulumi.md) | ✓ Available | Pulumi Automation API | Local or cloud |
| [CDK](cdk.md) | ◐ Planned | AWS Cloud Development Kit | CloudFormation |
| [Terraform](terraform.md) | ◐ Planned | Terraform HCL generation | Local or remote |

## Choosing a Backend

### Pulumi

**Best for:**

- Go developers (native Go SDK)
- Teams already using Pulumi
- Flexibility in state storage

**State options:** Local filesystem, S3, Pulumi Cloud

### AWS CDK

**Best for:**

- AWS-only deployments
- Teams familiar with CloudFormation
- Tight AWS integration

**State:** Managed by CloudFormation

### Terraform

**Best for:**

- Teams with Terraform expertise
- Multi-cloud deployments
- Existing Terraform workflows

**State options:** Local, S3, Terraform Cloud

## Backend Comparison

| Feature | Pulumi | CDK | Terraform |
|---------|--------|-----|-----------|
| Language | Go | Go/TS/Py | HCL |
| State management | Flexible | CloudFormation | Flexible |
| Preview/Plan | ✓ | ✓ | ✓ |
| Drift detection | ✓ | Limited | ✓ |
| Multi-cloud | ✓ | AWS only | ✓ |
| Learning curve | Medium | Medium | Medium |

## Using Backends

Specify backend with `--backend` flag:

```bash
# Pulumi (default)
omnideploy up --config deploy.yaml --backend pulumi

# CDK
omnideploy up --config deploy.yaml --backend cdk

# Terraform
omnideploy up --config deploy.yaml --backend terraform
```

## State Management

Each backend handles state differently:

### Pulumi

Default: `~/.omnideploy/pulumi/`

Configure remote:

```bash
export PULUMI_BACKEND_URL=s3://my-bucket/pulumi-state
```

### CDK

State managed by AWS CloudFormation. No local state files.

### Terraform

Default: `~/.omnideploy/terraform/`

Configure remote:

```bash
export TF_BACKEND_TYPE=s3
export TF_BACKEND_BUCKET=my-bucket
export TF_BACKEND_KEY=omnideploy/terraform.tfstate
```

## Switching Backends

Backends are independent. Switching backends creates new resources (doesn't migrate).

To migrate:

1. Deploy with new backend
2. Verify deployment works
3. Destroy old backend's resources

```bash
# Deploy with new backend
omnideploy up --config deploy.yaml --backend terraform --stack my-app-tf

# Test
curl https://new-url/health

# Destroy old
omnideploy destroy --stack my-app --backend pulumi
```

## List Available Backends

```bash
omnideploy backends
```

Output:

```
Available backends:
  pulumi       Pulumi - Infrastructure as Code using Go
```
