# Go API Reference

Use OmniDeploy programmatically in Go applications.

## Installation

```bash
go get github.com/plexusone/omnideploy
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/plexusone/omnideploy/backend"
    "github.com/plexusone/omnideploy/backend/pulumi"
    "github.com/plexusone/omnideploy/config"
    "github.com/plexusone/omnideploy/deploy"
    "github.com/plexusone/omnideploy/target/lightsail"
)

func main() {
    ctx := context.Background()

    // Create deployer
    deployer := deploy.New(
        deploy.WithTarget(&lightsail.Target{}),
        deploy.WithBackend(pulumi.New()),
    )

    // Deploy from config file
    result, err := deployer.Apply(ctx, "deploy.yaml", backend.ApplyOptions{
        StackName:   "my-app",
        AutoApprove: true,
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Deployed: %s\n", result.Outputs["url"])
}
```

## Core Types

### config.DeployConfig

```go
type DeployConfig struct {
    Name        string
    Version     string
    Region      string
    Container   ContainerConfig
    Service     ServiceConfig
    Resources   ResourceConfig
    Environment map[string]string
    Secrets     []SecretRef
    Tags        map[string]string
}
```

### target.Target

```go
type Target interface {
    Name() string
    Description() string
    Validate(cfg *config.DeployConfig) error
    ResourceSpec(cfg *config.DeployConfig) (*ResourceSpec, error)
}
```

### backend.Backend

```go
type Backend interface {
    Name() string
    Description() string
    Apply(ctx context.Context, spec *target.ResourceSpec, opts ApplyOptions) (*Result, error)
    Preview(ctx context.Context, spec *target.ResourceSpec) (*PreviewResult, error)
    Destroy(ctx context.Context, stackName string, opts DestroyOptions) error
    Refresh(ctx context.Context, stackName string) error
}
```

### runtime.Adapter

```go
type Adapter interface {
    Name() string
    Description() string
    Detect(path string) bool
    Load(path string) (*config.DeployConfig, error)
}
```

## Examples

### Deploy from Config Struct

```go
cfg := &config.DeployConfig{
    Name:   "my-api",
    Region: "us-east-1",
    Container: config.ContainerConfig{
        Image: "nginx:latest",
        Ports: []config.PortMapping{
            {ContainerPort: 80, Protocol: "HTTP"},
        },
    },
    Service: config.ServiceConfig{
        Replicas: 1,
        Public:   true,
    },
    Resources: config.ResourceConfig{
        Size: "micro",
    },
}

cfg.Defaults()

result, err := deployer.DeployFromConfig(ctx, cfg, backend.ApplyOptions{})
```

### Preview Deployment

```go
preview, err := deployer.Preview(ctx, "deploy.yaml")
if err != nil {
    log.Fatal(err)
}

fmt.Println(preview.Summary)
for _, change := range preview.Changes {
    fmt.Printf("%s: %s\n", change.Type, change.ResourceName)
}
```

### Destroy Stack

```go
err := deployer.Destroy(ctx, "my-app", backend.DestroyOptions{
    AutoApprove: true,
})
```

### Custom Target

```go
type MyTarget struct{}

func (t *MyTarget) Name() string { return "mytarget" }

func (t *MyTarget) ResourceSpec(cfg *config.DeployConfig) (*target.ResourceSpec, error) {
    // Generate resources for your target
    return &target.ResourceSpec{
        StackName: cfg.Name,
        Resources: []target.Resource{...},
    }, nil
}

// Use
deployer := deploy.New(
    deploy.WithTarget(&MyTarget{}),
    deploy.WithBackend(pulumi.New()),
)
```

## Package Reference

| Package | Description |
|---------|-------------|
| `config` | Configuration types and loading |
| `target` | Target interface and registry |
| `target/lightsail` | LightSail implementation |
| `backend` | Backend interface and registry |
| `backend/pulumi` | Pulumi implementation |
| `runtime` | Runtime adapter interface |
| `runtime/omniagent` | OmniAgent adapter |
| `runtime/container` | Generic container adapter |
| `deploy` | Deployment orchestrator |
