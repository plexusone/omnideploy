# AWS LightSail Target

Simple, cost-effective container hosting on AWS LightSail Container Service.

## Overview

LightSail Container Service provides:

- Fixed monthly pricing (no surprise bills)
- Built-in load balancing and HTTPS
- Simple scaling (1-20 containers)
- Automatic health checks
- Public endpoints with custom domains

## When to Use

**Good for:**

- Development and staging environments
- Small production workloads
- Predictable traffic patterns
- Cost-sensitive deployments
- Simple applications without complex scaling needs

**Not ideal for:**

- High-traffic applications requiring auto-scaling
- Applications needing persistent storage
- GPU workloads
- Complex networking requirements

## Resource Sizes

| Size | vCPU | Memory | Est. Monthly Cost |
|------|------|--------|-------------------|
| `nano` | 0.25 | 512 MB | ~$7 |
| `micro` | 0.5 | 1 GB | ~$10 |
| `small` | 1 | 2 GB | ~$25 |
| `medium` | 2 | 4 GB | ~$50 |
| `large` | 4 | 8 GB | ~$100 |
| `xlarge` | 8 | 16 GB | ~$200 |

*Prices are approximate and may vary by region.*

## Configuration

### Basic Example

```yaml
name: my-api
region: us-east-1

container:
  image: ghcr.io/myorg/my-api:latest
  ports:
    - container_port: 8080
      protocol: HTTP

service:
  replicas: 1
  public: true

resources:
  size: micro
```

### Full Example

```yaml
name: my-api
region: us-east-1

container:
  image: ghcr.io/myorg/my-api:v1.2.3
  args:
    - --port=8080
    - --log-level=info
  ports:
    - container_port: 8080
      protocol: HTTP
      name: api
  health_check:
    path: /health
    interval: 30s
    timeout: 5s
    healthy_threshold: 2
    unhealthy_threshold: 3

service:
  replicas: 2
  public: true

resources:
  size: small

environment:
  LOG_LEVEL: info
  DATABASE_URL: ${DATABASE_URL}

tags:
  environment: production
  team: platform
```

## Deployment

```bash
# Deploy
omnideploy up --config deploy.yaml --target lightsail

# Preview changes
omnideploy preview --config deploy.yaml --target lightsail

# Destroy
omnideploy destroy --stack my-api
```

## Outputs

After deployment, you'll receive:

| Output | Description |
|--------|-------------|
| `url` | Public HTTPS URL |
| `service_name` | LightSail service name |
| `state` | Deployment state (RUNNING, etc.) |

Example:

```
Outputs:
    service_name: "my-api"
    state       : "RUNNING"
    url         : "https://my-api.abc123.us-east-1.cs.amazonlightsail.com"
```

## Limitations

### No Persistent Storage

LightSail Container Service doesn't support persistent volumes. Container storage is ephemeral.

**Workarounds:**

- Use external databases (RDS, DynamoDB)
- Use S3 for file storage
- Use external caching (ElastiCache)

### Fixed Scaling

Scaling is manual (set `replicas` in config). No auto-scaling based on metrics.

### Region Availability

Available in most AWS regions. Check [AWS documentation](https://docs.aws.amazon.com/lightsail/latest/userguide/amazon-lightsail-regions.html) for current availability.

### Container Limits

- Maximum 20 replicas per service
- Single container per deployment (no sidecars)
- Public endpoint required for health checks

## Health Checks

LightSail requires health checks for public endpoints:

```yaml
container:
  health_check:
    path: /health          # Must return 200-399
    interval: 30s          # 5-300 seconds
    timeout: 5s            # 2-60 seconds
    healthy_threshold: 2   # 2-10
    unhealthy_threshold: 3 # 2-10
```

Ensure your application has a health endpoint that:

- Returns HTTP 200-399 when healthy
- Responds within the timeout
- Is lightweight (doesn't perform heavy operations)

## Custom Domains

LightSail provides automatic HTTPS. For custom domains:

1. Deploy first to get the LightSail URL
2. Create a CNAME record pointing to the LightSail URL
3. LightSail automatically provisions an SSL certificate

## Monitoring

View logs via AWS CLI:

```bash
aws lightsail get-container-log \
  --service-name my-api \
  --container-name my-api
```

View service status:

```bash
aws lightsail get-container-services --service-name my-api
```

## Cost Optimization

1. **Right-size resources**: Start with `nano` or `micro`, scale up as needed
2. **Use single replica for dev/staging**: Multiple replicas only for production
3. **Clean up unused services**: Destroy services when not needed

```bash
# List all services
aws lightsail get-container-services

# Destroy unused
omnideploy destroy --stack old-service
```

## Troubleshooting

### Deployment Stuck

If deployment takes too long:

1. Check container logs for startup errors
2. Verify health check endpoint is accessible
3. Ensure image is publicly accessible or credentials are correct

### Container Keeps Restarting

1. Check health check configuration
2. Verify the health endpoint returns 2xx/3xx
3. Check container logs for errors

### Cannot Pull Image

1. For private registries, ensure credentials are configured
2. Verify image URL is correct
3. Check network connectivity from LightSail

## Next Steps

- [ECS Target](ecs.md) - For auto-scaling workloads
- [Configuration Schema](../configuration/schema.md) - Full configuration reference
