# GitHub Actions Deployment

Automate deployments with GitHub Actions CI/CD.

## Overview

Set up a complete CI/CD pipeline that:

1. Builds your container image on push
2. Pushes to GitHub Container Registry
3. Deploys to your cloud provider

## Basic Workflow

Create `.github/workflows/deploy.yaml`:

```yaml
name: Build and Deploy

on:
  push:
    branches: [main]
    tags: ['v*']
  pull_request:
    branches: [main]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    outputs:
      image_tag: ${{ steps.meta.outputs.tags }}

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to Container Registry
        if: github.event_name != 'pull_request'
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=ref,event=branch
            type=ref,event=pr
            type=semver,pattern={{version}}
            type=sha

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          push: ${{ github.event_name != 'pull_request' }}
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

  deploy:
    needs: build
    runs-on: ubuntu-latest
    if: github.event_name != 'pull_request'
    environment: production

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install omnideploy
        run: go install github.com/plexusone/omnideploy/cmd/omnideploy@latest

      - name: Deploy
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          AWS_REGION: us-east-1
        run: |
          omnideploy up \
            --config deploy.yaml \
            --target lightsail \
            --backend pulumi \
            --stack production \
            --yes
```

## Environment-Based Deployments

Deploy to different environments based on branch:

```yaml
name: Deploy

on:
  push:
    branches: [main, develop]

jobs:
  deploy:
    runs-on: ubuntu-latest
    environment: ${{ github.ref == 'refs/heads/main' && 'production' || 'staging' }}

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install omnideploy
        run: go install github.com/plexusone/omnideploy/cmd/omnideploy@latest

      - name: Set environment
        id: env
        run: |
          if [ "${{ github.ref }}" == "refs/heads/main" ]; then
            echo "stack=production" >> $GITHUB_OUTPUT
            echo "config=deploy.prod.yaml" >> $GITHUB_OUTPUT
          else
            echo "stack=staging" >> $GITHUB_OUTPUT
            echo "config=deploy.staging.yaml" >> $GITHUB_OUTPUT
          fi

      - name: Deploy
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          AWS_REGION: us-east-1
        run: |
          omnideploy up \
            --config ${{ steps.env.outputs.config }} \
            --stack ${{ steps.env.outputs.stack }} \
            --yes
```

## Tag-Based Releases

Deploy only on version tags:

```yaml
name: Release

on:
  push:
    tags: ['v*']

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    environment: production

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Get version
        id: version
        run: echo "version=${GITHUB_REF#refs/tags/v}" >> $GITHUB_OUTPUT

      - name: Build and push image
        # ... build steps ...

      - name: Update deploy config
        run: |
          sed -i "s|image:.*|image: ghcr.io/${{ github.repository }}:${{ steps.version.outputs.version }}|" deploy.yaml

      - name: Deploy
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
        run: |
          omnideploy up --config deploy.yaml --stack production --yes
```

## Preview on Pull Requests

Show deployment preview on PRs:

```yaml
name: Preview

on:
  pull_request:
    branches: [main]

jobs:
  preview:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install omnideploy
        run: go install github.com/plexusone/omnideploy/cmd/omnideploy@latest

      - name: Preview deployment
        id: preview
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
        run: |
          OUTPUT=$(omnideploy preview --config deploy.yaml 2>&1)
          echo "preview<<EOF" >> $GITHUB_OUTPUT
          echo "$OUTPUT" >> $GITHUB_OUTPUT
          echo "EOF" >> $GITHUB_OUTPUT

      - name: Comment on PR
        uses: actions/github-script@v7
        with:
          script: |
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: `## Deployment Preview\n\n\`\`\`\n${{ steps.preview.outputs.preview }}\n\`\`\``
            })
```

## Secrets Configuration

Configure these secrets in your GitHub repository:

| Secret | Description |
|--------|-------------|
| `AWS_ACCESS_KEY_ID` | AWS access key |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key |
| `ANTHROPIC_API_KEY` | (Optional) For OmniAgent apps |
| `TWILIO_ACCOUNT_SID` | (Optional) For voice/SMS |
| `TWILIO_AUTH_TOKEN` | (Optional) For voice/SMS |

### Setting Secrets

1. Go to repository **Settings** → **Secrets and variables** → **Actions**
2. Click **New repository secret**
3. Add each secret

### Using Environments

For production deployments, use GitHub Environments for approval gates:

1. Go to **Settings** → **Environments**
2. Create `production` environment
3. Enable **Required reviewers**
4. Add protection rules

## Pulumi State Management

For team deployments, use remote state storage:

### Option 1: Pulumi Cloud

```yaml
- name: Setup Pulumi
  uses: pulumi/actions@v5

- name: Login to Pulumi
  run: pulumi login
  env:
    PULUMI_ACCESS_TOKEN: ${{ secrets.PULUMI_ACCESS_TOKEN }}
```

### Option 2: AWS S3 Backend

```yaml
- name: Deploy
  env:
    PULUMI_BACKEND_URL: s3://${{ secrets.PULUMI_STATE_BUCKET }}/omnideploy
    AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
    AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
  run: omnideploy up --config deploy.yaml --yes
```

## Complete Example

Full workflow for an OmniAgent application:

```yaml
name: CI/CD

on:
  push:
    branches: [main]
    tags: ['v*']
  pull_request:
    branches: [main]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: go test ./...

  build:
    needs: test
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    outputs:
      image: ${{ steps.meta.outputs.tags }}
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        if: github.event_name != 'pull_request'
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/metadata-action@v5
        id: meta
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
      - uses: docker/build-push-action@v5
        with:
          context: .
          push: ${{ github.event_name != 'pull_request' }}
          tags: ${{ steps.meta.outputs.tags }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

  deploy-staging:
    needs: build
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    environment: staging
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: go install github.com/plexusone/omnideploy/cmd/omnideploy@latest
      - name: Deploy to staging
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: |
          omnideploy up \
            --config deploy.yaml \
            --stack staging \
            --yes

  deploy-production:
    needs: [build, deploy-staging]
    if: startsWith(github.ref, 'refs/tags/v')
    runs-on: ubuntu-latest
    environment: production
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: go install github.com/plexusone/omnideploy/cmd/omnideploy@latest
      - name: Deploy to production
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: |
          omnideploy up \
            --config deploy.yaml \
            --stack production \
            --yes
```

## Next Steps

- [GitLab CI](gitlab-ci.md) - Deploy with GitLab CI/CD
- [Continuous Deployment](continuous.md) - Automated deployment strategies
