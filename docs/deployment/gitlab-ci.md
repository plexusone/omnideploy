# GitLab CI Deployment

Deploy with GitLab CI/CD pipelines.

## Basic Pipeline

Create `.gitlab-ci.yml`:

```yaml
stages:
  - build
  - deploy

variables:
  REGISTRY: registry.gitlab.com
  IMAGE_NAME: $CI_PROJECT_PATH

build:
  stage: build
  image: docker:24
  services:
    - docker:24-dind
  script:
    - docker login -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD $CI_REGISTRY
    - docker build -t $REGISTRY/$IMAGE_NAME:$CI_COMMIT_SHA .
    - docker push $REGISTRY/$IMAGE_NAME:$CI_COMMIT_SHA
  only:
    - main
    - tags

deploy:
  stage: deploy
  image: golang:1.24
  before_script:
    - go install github.com/plexusone/omnideploy/cmd/omnideploy@latest
  script:
    - omnideploy up --config deploy.yaml --stack $CI_ENVIRONMENT_SLUG --yes
  environment:
    name: production
    url: https://my-app.example.com
  only:
    - main
```

## Multi-Environment

```yaml
stages:
  - build
  - deploy-staging
  - deploy-production

build:
  stage: build
  # ... build steps

deploy-staging:
  stage: deploy-staging
  image: golang:1.24
  before_script:
    - go install github.com/plexusone/omnideploy/cmd/omnideploy@latest
  script:
    - omnideploy up --config deploy.yaml --stack staging --yes
  environment:
    name: staging
  only:
    - main

deploy-production:
  stage: deploy-production
  image: golang:1.24
  before_script:
    - go install github.com/plexusone/omnideploy/cmd/omnideploy@latest
  script:
    - omnideploy up --config deploy.yaml --stack production --yes
  environment:
    name: production
  when: manual
  only:
    - tags
```

## Variables

Configure in **Settings → CI/CD → Variables**:

| Variable | Description | Protected | Masked |
|----------|-------------|-----------|--------|
| `AWS_ACCESS_KEY_ID` | AWS access key | Yes | Yes |
| `AWS_SECRET_ACCESS_KEY` | AWS secret | Yes | Yes |
| `ANTHROPIC_API_KEY` | App secret | Yes | Yes |

## With GitLab Container Registry

```yaml
build:
  stage: build
  image: docker:24
  services:
    - docker:24-dind
  script:
    - docker login -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD $CI_REGISTRY
    - docker build -t $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA .
    - docker build -t $CI_REGISTRY_IMAGE:latest .
    - docker push $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA
    - docker push $CI_REGISTRY_IMAGE:latest
```

## Preview on Merge Requests

```yaml
preview:
  stage: deploy
  image: golang:1.24
  before_script:
    - go install github.com/plexusone/omnideploy/cmd/omnideploy@latest
  script:
    - omnideploy preview --config deploy.yaml > preview.txt
    - cat preview.txt
  artifacts:
    paths:
      - preview.txt
  only:
    - merge_requests
```
