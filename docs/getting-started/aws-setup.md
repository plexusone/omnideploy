# AWS Setup

Configure AWS credentials and permissions for OmniDeploy.

## Quick Start with Bootstrap Command

The easiest way to set up AWS IAM resources is using the `omnideploy bootstrap` command:

```bash
# Using admin credentials, create policy and group
omnideploy bootstrap run

# Or create a user with access keys
omnideploy bootstrap run --user deployer --create-key
```

This creates:

- **OmniDeployPolicy** - IAM policy with required permissions
- **omnideploy-users** - IAM group with the policy attached
- Optionally, an IAM user with access keys

See [Bootstrap Command](#bootstrap-command) for details.

## Required Permissions

OmniDeploy needs permissions for the deployment target and container registry.

### LightSail Target

| Permission | Purpose |
|------------|---------|
| `lightsail:CreateContainerService` | Create container service |
| `lightsail:CreateContainerServiceDeployment` | Deploy containers |
| `lightsail:GetContainerServices` | Check service status |
| `lightsail:DeleteContainerService` | Destroy resources |
| `lightsail:UpdateContainerService` | Update configuration |

### ECR (Container Registry)

| Permission | Purpose |
|------------|---------|
| `ecr:GetAuthorizationToken` | Authenticate Docker |
| `ecr:CreateRepository` | Create image repository |
| `ecr:BatchCheckLayerAvailability` | Push images |
| `ecr:PutImage` | Push images |
| `ecr:BatchGetImage` | Pull images |

### SSM (Secrets)

| Permission | Purpose |
|------------|---------|
| `ssm:GetParameter` | Read secrets |
| `ssm:PutParameter` | Store secrets (optional) |

## Creating an IAM User

### Option 1: Managed Policies (Recommended)

Attach AWS-managed policies for simplicity:

1. Go to **IAM Console** → **Users** → **Create user**

2. Enter username (e.g., `omnideploy`)

3. Select **Attach policies directly**

4. Search and attach these managed policies:

   | Policy | Permissions |
   |--------|-------------|
   | `AmazonLightsailFullAccess` | Full LightSail access |
   | `AmazonEC2ContainerRegistryFullAccess` | Full ECR access |
   | `AmazonSSMReadOnlyAccess` | Read SSM parameters |

5. Click **Create user**

6. Go to **Security credentials** → **Create access key**

7. Select **Command Line Interface (CLI)**

8. Download or copy the access key and secret

### Option 2: Custom Policy (Least Privilege)

For production, create a custom policy with minimal permissions:

1. Go to **IAM Console** → **Policies** → **Create policy**

2. Use this JSON:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "LightSailContainers",
      "Effect": "Allow",
      "Action": [
        "lightsail:CreateContainerService",
        "lightsail:CreateContainerServiceDeployment",
        "lightsail:CreateContainerServiceRegistryLogin",
        "lightsail:DeleteContainerService",
        "lightsail:GetContainerServiceDeployments",
        "lightsail:GetContainerServices",
        "lightsail:GetContainerLog",
        "lightsail:RegisterContainerImage",
        "lightsail:UpdateContainerService"
      ],
      "Resource": "*"
    },
    {
      "Sid": "ECR",
      "Effect": "Allow",
      "Action": [
        "ecr:GetAuthorizationToken",
        "ecr:BatchCheckLayerAvailability",
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage",
        "ecr:PutImage",
        "ecr:InitiateLayerUpload",
        "ecr:UploadLayerPart",
        "ecr:CompleteLayerUpload",
        "ecr:CreateRepository",
        "ecr:DescribeRepositories"
      ],
      "Resource": "*"
    },
    {
      "Sid": "SSMSecrets",
      "Effect": "Allow",
      "Action": [
        "ssm:GetParameter",
        "ssm:GetParameters"
      ],
      "Resource": "arn:aws:ssm:*:*:parameter/*"
    }
  ]
}
```

3. Name it `OmniDeployPolicy`

4. Create IAM user and attach this policy

### Option 3: IAM Group (Teams)

For teams, use groups:

1. Create IAM group `omnideploy-users`
2. Attach policies to the group
3. Add users to the group

## Configuring Credentials

### Environment Variables

```bash
export AWS_ACCESS_KEY_ID="AKIA..."
export AWS_SECRET_ACCESS_KEY="..."
export AWS_REGION="us-east-1"
```

### AWS CLI Profile

```bash
aws configure --profile omnideploy
# Enter access key, secret, region

export AWS_PROFILE="omnideploy"
```

### Shared Credentials File

Add to `~/.aws/credentials`:

```ini
[omnideploy]
aws_access_key_id = AKIA...
aws_secret_access_key = ...
```

Add to `~/.aws/config`:

```ini
[profile omnideploy]
region = us-east-1
output = json
```

## Verify Setup

```bash
# Check identity
aws sts get-caller-identity

# Expected output:
{
    "UserId": "AIDA...",
    "Account": "123456789012",
    "Arn": "arn:aws:iam::123456789012:user/omnideploy"
}
```

## Setting Up ECR

Create a repository for your container images:

```bash
# Create repository
aws ecr create-repository \
    --repository-name my-app \
    --region us-east-1

# Get repository URI
aws ecr describe-repositories \
    --repository-names my-app \
    --query 'repositories[0].repositoryUri' \
    --output text
# Output: 123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app
```

### Push Image to ECR

```bash
# Login to ECR
aws ecr get-login-password --region us-east-1 | \
    docker login --username AWS --password-stdin \
    123456789012.dkr.ecr.us-east-1.amazonaws.com

# Tag image
docker tag my-app:latest \
    123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app:latest

# Push
docker push 123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app:latest
```

### Update deploy.yaml

```yaml
container:
  image: 123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app:latest
```

## Using GHCR with Private Repos

If using GitHub Container Registry with private images, configure registry credentials:

```yaml
container:
  image: ghcr.io/owner/repo:latest
  registry:
    server: ghcr.io
    username: github-username
    password_env: GITHUB_TOKEN
```

Create a GitHub token with `read:packages` scope and set:

```bash
export GITHUB_TOKEN="ghp_..."
```

## IAM Roles for CI/CD

For GitHub Actions, use OIDC instead of access keys:

1. Create IAM Identity Provider for GitHub
2. Create IAM Role with trust policy for GitHub
3. Attach OmniDeploy permissions to the role

See [GitHub Actions deployment](../deployment/github-actions.md) for details.

## Troubleshooting

### "Access Denied" Errors

Check that your IAM user has the required policies attached:

```bash
aws iam list-attached-user-policies --user-name omnideploy
```

### "Token Expired" Errors

Refresh credentials:

```bash
# For SSO
aws sso login --profile your-profile

# For access keys, create new ones in IAM Console
```

### ECR Login Fails

Ensure you have `ecr:GetAuthorizationToken` permission:

```bash
aws ecr get-login-password --region us-east-1
```

If this fails, check IAM permissions.

## Bootstrap Command

The `omnideploy bootstrap` command automates IAM setup.

### Commands

```bash
# Create policy and group (requires admin credentials)
omnideploy bootstrap run

# Create policy, group, and user with access keys
omnideploy bootstrap run --user deployer --create-key

# Check current status
omnideploy bootstrap status

# View the IAM policy document
omnideploy bootstrap policy
```

### Workflow

1. **Get admin credentials** - Use AWS Console or existing admin profile

2. **Run bootstrap**:
   ```bash
   export AWS_PROFILE=admin  # or export AWS_ACCESS_KEY_ID/SECRET
   omnideploy bootstrap run --user deployer --create-key
   ```

3. **Save the output** - Copy the access key and secret

4. **Use new credentials for deployments**:
   ```bash
   export AWS_ACCESS_KEY_ID="AKIA..."
   export AWS_SECRET_ACCESS_KEY="..."
   export AWS_REGION="us-east-1"

   omnideploy up --config deploy.yaml
   ```

### What Bootstrap Creates

| Resource | Name | Description |
|----------|------|-------------|
| IAM Policy | `OmniDeployPolicy` | Permissions for LightSail, ECR, SSM |
| IAM Group | `omnideploy-users` | Group with policy attached |
| IAM User | (optional) | User in the group |
| Access Key | (optional) | Credentials for the user |

### Idempotent

The bootstrap command is safe to run multiple times. It checks for existing resources and skips creation if they already exist.
