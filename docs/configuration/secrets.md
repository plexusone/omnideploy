# Secrets Management

Securely manage sensitive values in your deployments.

## Overview

OmniDeploy supports multiple approaches for handling secrets:

1. **Environment Variables** - Pass at deploy time
2. **Pulumi Config** - Encrypted in Pulumi state
3. **AWS Secrets Manager** - Fetched at container runtime
4. **AWS SSM Parameter Store** - Fetched at container runtime

## Environment Variables

The simplest approach - pass secrets when deploying:

```yaml
# deploy.yaml
environment:
  API_KEY: ${API_KEY}
  DATABASE_URL: ${DATABASE_URL}
```

Deploy:

```bash
API_KEY=sk-... DATABASE_URL=postgres://... omnideploy up --config deploy.yaml
```

**Pros:**

- Simple, no setup required
- Works everywhere

**Cons:**

- Secrets visible in process list
- Not suitable for shared CI/CD
- No audit trail

## Pulumi Secrets

Store secrets encrypted in Pulumi state:

```bash
# Set secret
cd ~/.omnideploy/pulumi
pulumi stack select my-app
pulumi config set --secret api_key sk-...
pulumi config set --secret database_url postgres://...
```

Reference in deployment:

```yaml
# deploy.yaml
environment:
  API_KEY: ${pulumi:api_key}
  DATABASE_URL: ${pulumi:database_url}
```

**Pros:**

- Encrypted at rest
- Version controlled (with state)
- Team sharing via remote state

**Cons:**

- Requires Pulumi CLI for management
- Secrets in Pulumi state

## AWS Secrets Manager

Store secrets in AWS and fetch at container runtime:

### 1. Create Secret

```bash
aws secretsmanager create-secret \
  --name my-app/api-key \
  --secret-string "sk-..."

aws secretsmanager create-secret \
  --name my-app/database-url \
  --secret-string "postgres://..."
```

### 2. Reference in Config

```yaml
# deploy.yaml
secrets:
  - name: API_KEY
    source: secretsmanager:my-app/api-key

  - name: DATABASE_URL
    source: secretsmanager:my-app/database-url
```

### 3. Grant Access

Ensure your container has IAM permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue"
      ],
      "Resource": [
        "arn:aws:secretsmanager:*:*:secret:my-app/*"
      ]
    }
  ]
}
```

**Pros:**

- Secrets never in config or state
- Audit trail via CloudTrail
- Automatic rotation support
- Fine-grained access control

**Cons:**

- AWS-specific
- Additional cost (~$0.40/secret/month)
- Requires IAM setup

## AWS SSM Parameter Store

Similar to Secrets Manager, but simpler and cheaper:

### 1. Create Parameter

```bash
aws ssm put-parameter \
  --name /my-app/api-key \
  --value "sk-..." \
  --type SecureString

aws ssm put-parameter \
  --name /my-app/database-url \
  --value "postgres://..." \
  --type SecureString
```

### 2. Reference in Config

```yaml
# deploy.yaml
secrets:
  - name: API_KEY
    source: ssm:/my-app/api-key

  - name: DATABASE_URL
    source: ssm:/my-app/database-url
```

**Pros:**

- Free for standard parameters
- Simple API
- Integrated with AWS

**Cons:**

- Less features than Secrets Manager
- No automatic rotation

## Comparison

| Method | Security | Cost | Complexity | Audit |
|--------|----------|------|------------|-------|
| Environment Variables | Low | Free | Low | None |
| Pulumi Config | Medium | Free | Medium | Git history |
| AWS Secrets Manager | High | ~$0.40/mo | Medium | CloudTrail |
| AWS SSM | High | Free | Medium | CloudTrail |

## Best Practices

### 1. Never Commit Secrets

```bash
# .gitignore
.env
.env.*
*.pem
*.key
credentials.json
```

### 2. Use Different Secrets Per Environment

```bash
# Staging
aws secretsmanager create-secret --name staging/my-app/api-key ...

# Production
aws secretsmanager create-secret --name production/my-app/api-key ...
```

### 3. Rotate Secrets Regularly

Set up automatic rotation in AWS Secrets Manager.

### 4. Audit Access

Enable CloudTrail and monitor for unexpected secret access.

### 5. Least Privilege

Grant minimal IAM permissions:

```json
{
  "Effect": "Allow",
  "Action": ["secretsmanager:GetSecretValue"],
  "Resource": ["arn:aws:secretsmanager:*:*:secret:my-app/*"],
  "Condition": {
    "StringEquals": {
      "aws:RequestTag/environment": "production"
    }
  }
}
```

## CI/CD Integration

### GitHub Actions

```yaml
- name: Deploy
  env:
    AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
    AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
    # App secrets via AWS Secrets Manager - not in env
  run: omnideploy up --config deploy.yaml --yes
```

### With OIDC (Recommended)

```yaml
- name: Configure AWS credentials
  uses: aws-actions/configure-aws-credentials@v4
  with:
    role-to-assume: arn:aws:iam::123456789:role/github-actions
    aws-region: us-east-1

- name: Deploy
  run: omnideploy up --config deploy.yaml --yes
```
