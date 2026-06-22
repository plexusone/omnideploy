# Configuration Examples

Real-world configuration examples for common scenarios.

## Web Applications

### Static Site (Nginx)

```yaml
name: my-website
region: us-east-1

container:
  image: nginx:alpine
  ports:
    - container_port: 80
      protocol: HTTP
  health_check:
    path: /

service:
  replicas: 1
  public: true

resources:
  size: nano
```

### Node.js API

```yaml
name: node-api
region: us-east-1

container:
  image: ghcr.io/myorg/node-api:latest
  ports:
    - container_port: 3000
      protocol: HTTP
  health_check:
    path: /health
    interval: 15s

service:
  replicas: 2
  public: true

resources:
  size: small

environment:
  NODE_ENV: production
  PORT: "3000"
  DATABASE_URL: ${DATABASE_URL}
```

### Python Flask App

```yaml
name: flask-app
region: us-west-2

container:
  image: ghcr.io/myorg/flask-app:v1.0.0
  args:
    - gunicorn
    - --bind=0.0.0.0:8000
    - app:app
  ports:
    - container_port: 8000
      protocol: HTTP
  health_check:
    path: /health

service:
  replicas: 2
  public: true

resources:
  size: small

environment:
  FLASK_ENV: production
  SECRET_KEY: ${SECRET_KEY}
  DATABASE_URL: ${DATABASE_URL}
```

## AI/ML Applications

### OmniAgent Gateway

```yaml
name: my-agent
region: us-east-1

container:
  image: ghcr.io/myorg/my-agent:latest
  args:
    - gateway
    - run
  ports:
    - container_port: 18789
      protocol: HTTP
  health_check:
    path: /api/health

service:
  replicas: 1
  public: true

resources:
  size: small

environment:
  OMNIAGENT_AGENT_PROVIDER: anthropic
  OMNIAGENT_AGENT_MODEL: claude-sonnet-4-20250514
  ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
  LOG_LEVEL: info

tags:
  app: omniagent
  type: ai-gateway
```

### LLM Proxy

```yaml
name: llm-proxy
region: us-east-1

container:
  image: ghcr.io/myorg/llm-proxy:latest
  ports:
    - container_port: 8080
      protocol: HTTP
  health_check:
    path: /health
    interval: 10s
    timeout: 3s

service:
  replicas: 3
  public: true

resources:
  size: medium

environment:
  OPENAI_API_KEY: ${OPENAI_API_KEY}
  ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
  RATE_LIMIT: "100"
  CACHE_TTL: "3600"
```

## Microservices

### API Gateway

```yaml
name: api-gateway
region: us-east-1

container:
  image: ghcr.io/myorg/gateway:latest
  ports:
    - container_port: 8080
      protocol: HTTP
  health_check:
    path: /health

service:
  replicas: 2
  public: true

resources:
  size: small

environment:
  AUTH_SERVICE_URL: http://auth-service:8080
  USER_SERVICE_URL: http://user-service:8080
  JWT_SECRET: ${JWT_SECRET}
```

### Background Worker

```yaml
name: queue-worker
region: us-east-1

container:
  image: ghcr.io/myorg/worker:latest
  args:
    - --queue=high,default,low
    - --concurrency=20
  ports:
    - container_port: 9090
      protocol: HTTP
      name: metrics
  health_check:
    path: /metrics

service:
  replicas: 3
  public: false

resources:
  size: medium

environment:
  REDIS_URL: ${REDIS_URL}
  DATABASE_URL: ${DATABASE_URL}
```

## Full-Stack Applications

### Next.js with API

```yaml
name: nextjs-app
region: us-east-1

container:
  image: ghcr.io/myorg/nextjs-app:latest
  ports:
    - container_port: 3000
      protocol: HTTP
  health_check:
    path: /api/health

service:
  replicas: 2
  public: true

resources:
  size: small

environment:
  NODE_ENV: production
  NEXT_PUBLIC_API_URL: https://api.example.com
  DATABASE_URL: ${DATABASE_URL}
  NEXTAUTH_SECRET: ${NEXTAUTH_SECRET}
  NEXTAUTH_URL: https://app.example.com
```

## Multi-Environment Setup

### Development

```yaml
# deploy.dev.yaml
name: my-app-dev
region: us-east-1

container:
  image: ghcr.io/myorg/my-app:dev
  ports:
    - container_port: 8080
      protocol: HTTP
  health_check:
    path: /health

service:
  replicas: 1
  public: true

resources:
  size: nano

environment:
  LOG_LEVEL: debug
  DATABASE_URL: ${DEV_DATABASE_URL}
```

### Production

```yaml
# deploy.prod.yaml
name: my-app-prod
region: us-east-1

container:
  image: ghcr.io/myorg/my-app:v1.2.3
  ports:
    - container_port: 8080
      protocol: HTTP
  health_check:
    path: /health
    interval: 15s
    timeout: 3s
    healthy_threshold: 3
    unhealthy_threshold: 2

service:
  replicas: 3
  public: true

resources:
  size: medium

environment:
  LOG_LEVEL: info
  DATABASE_URL: ${PROD_DATABASE_URL}

tags:
  environment: production
  cost-center: engineering
```
