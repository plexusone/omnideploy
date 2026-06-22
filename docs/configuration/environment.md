# Environment Variables

Configure OmniDeploy and your deployments using environment variables.

## OmniDeploy Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `OMNIDEPLOY_WORK_DIR` | Working directory for state | `~/.omnideploy` |

## Cloud Provider Credentials

### AWS

| Variable | Description |
|----------|-------------|
| `AWS_ACCESS_KEY_ID` | AWS access key ID |
| `AWS_SECRET_ACCESS_KEY` | AWS secret access key |
| `AWS_SESSION_TOKEN` | AWS session token (for temporary credentials) |
| `AWS_REGION` | Default AWS region |
| `AWS_PROFILE` | AWS CLI profile name |

### DigitalOcean

| Variable | Description |
|----------|-------------|
| `DIGITALOCEAN_TOKEN` | DigitalOcean API token |

## Backend Configuration

### Pulumi

| Variable | Description |
|----------|-------------|
| `PULUMI_BACKEND_URL` | Remote state URL (s3://, gs://, azblob://) |
| `PULUMI_ACCESS_TOKEN` | Pulumi Cloud access token |
| `PULUMI_CONFIG_PASSPHRASE` | Passphrase for local secrets encryption |

### Terraform

| Variable | Description |
|----------|-------------|
| `TF_VAR_*` | Terraform variable values |
| `TF_BACKEND_*` | Terraform backend configuration |

## Variable Expansion

Environment variables can be referenced in config files:

```yaml
environment:
  # Direct reference
  API_KEY: ${API_KEY}

  # With default value
  LOG_LEVEL: ${LOG_LEVEL:-info}

  # Nested reference
  DATABASE_URL: ${DATABASE_URL}
```

### Expansion Syntax

| Syntax | Description |
|--------|-------------|
| `${VAR}` | Required variable (error if missing) |
| `${VAR:-default}` | Default if variable is unset or empty |
| `${VAR-default}` | Default only if variable is unset |
| `$$` | Literal dollar sign |

### Example

```yaml
# deploy.yaml
name: my-app

environment:
  # Required - deployment fails if missing
  DATABASE_URL: ${DATABASE_URL}

  # Optional with defaults
  LOG_LEVEL: ${LOG_LEVEL:-info}
  PORT: ${PORT:-8080}

  # Literal dollar sign
  PRICE: $$100
```

Deploy with:

```bash
DATABASE_URL=postgres://... LOG_LEVEL=debug omnideploy up --config deploy.yaml
```

## Setting Variables

### Shell Export

```bash
export AWS_ACCESS_KEY_ID=AKIA...
export AWS_SECRET_ACCESS_KEY=...
export DATABASE_URL=postgres://...

omnideploy up --config deploy.yaml
```

### Inline

```bash
DATABASE_URL=postgres://... omnideploy up --config deploy.yaml
```

### dotenv File

Create `.env`:

```bash
AWS_ACCESS_KEY_ID=AKIA...
AWS_SECRET_ACCESS_KEY=...
DATABASE_URL=postgres://...
```

Load and deploy:

```bash
export $(cat .env | xargs) && omnideploy up --config deploy.yaml
```

### direnv

Create `.envrc`:

```bash
export AWS_ACCESS_KEY_ID=AKIA...
export AWS_SECRET_ACCESS_KEY=...
export DATABASE_URL=postgres://...
```

Allow and use:

```bash
direnv allow
omnideploy up --config deploy.yaml
```

## CI/CD Variables

### GitHub Actions

```yaml
- name: Deploy
  env:
    AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
    AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
    DATABASE_URL: ${{ secrets.DATABASE_URL }}
  run: omnideploy up --config deploy.yaml --yes
```

### GitLab CI

```yaml
deploy:
  variables:
    AWS_ACCESS_KEY_ID: $AWS_ACCESS_KEY_ID
    AWS_SECRET_ACCESS_KEY: $AWS_SECRET_ACCESS_KEY
    DATABASE_URL: $DATABASE_URL
  script:
    - omnideploy up --config deploy.yaml --yes
```

## Security Best Practices

1. **Never commit secrets**: Add `.env` to `.gitignore`
2. **Use secret managers**: AWS Secrets Manager, HashiCorp Vault
3. **Rotate credentials**: Regularly rotate API keys and passwords
4. **Least privilege**: Use IAM roles with minimal permissions
5. **Audit access**: Enable CloudTrail for AWS API logging
