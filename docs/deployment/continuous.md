# Continuous Deployment

Strategies for automated deployments.

## Deployment Strategies

### 1. Trunk-Based Development

Deploy on every push to main:

```yaml
# .github/workflows/deploy.yaml
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: go install github.com/plexusone/omnideploy/cmd/omnideploy@latest
      - run: omnideploy up --config deploy.yaml --yes
```

### 2. Release-Based

Deploy only on version tags:

```yaml
on:
  push:
    tags: ['v*']

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: omnideploy up --config deploy.yaml --stack production --yes
```

### 3. Environment Promotion

```
main → staging → production
```

```yaml
on:
  push:
    branches: [main]

jobs:
  deploy-staging:
    runs-on: ubuntu-latest
    environment: staging
    steps:
      - run: omnideploy up --config deploy.yaml --stack staging --yes

  deploy-production:
    needs: deploy-staging
    runs-on: ubuntu-latest
    environment: production
    steps:
      - run: omnideploy up --config deploy.yaml --stack production --yes
```

## Rollback Strategies

### Manual Rollback

```bash
# Deploy previous version
omnideploy up --config deploy.yaml --stack production --yes
# (with previous image tag in config)
```

### Automated Rollback

```yaml
deploy:
  runs-on: ubuntu-latest
  steps:
    - name: Deploy
      id: deploy
      run: omnideploy up --config deploy.yaml --yes

    - name: Health check
      id: health
      run: curl -f https://my-app.example.com/health

    - name: Rollback on failure
      if: failure() && steps.deploy.outcome == 'success'
      run: |
        git checkout HEAD~1 -- deploy.yaml
        omnideploy up --config deploy.yaml --yes
```

## Blue-Green Deployments

Deploy new version alongside old, then switch:

```bash
# Deploy new version to blue stack
omnideploy up --config deploy.yaml --stack blue --yes

# Test blue
curl https://blue.example.com/health

# Switch traffic (DNS or load balancer)
# ...

# Destroy old green
omnideploy destroy --stack green --yes
```

## Canary Deployments

Not directly supported by LightSail. Use ECS or Kubernetes targets for canary deployments with traffic splitting.

## Monitoring Deployments

### Deployment Notifications

```yaml
- name: Notify Slack
  if: always()
  uses: slackapi/slack-github-action@v1
  with:
    payload: |
      {
        "text": "Deployment ${{ job.status }}: ${{ github.repository }}"
      }
  env:
    SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK }}
```

### Health Checks

```yaml
- name: Verify deployment
  run: |
    for i in {1..30}; do
      if curl -sf https://my-app.example.com/health; then
        echo "Deployment healthy"
        exit 0
      fi
      sleep 10
    done
    echo "Health check failed"
    exit 1
```

## Best Practices

1. **Always preview first** in CI before deploying
2. **Use environments** for approval gates
3. **Monitor after deploy** with health checks
4. **Have rollback plan** ready
5. **Keep deployments small** and frequent
6. **Use semantic versioning** for releases
