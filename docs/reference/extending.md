# Extending OmniDeploy

Add custom targets, backends, and runtime adapters.

## Architecture

OmniDeploy uses a pluggable architecture with three extension points:

1. **Targets** - Where to deploy
2. **Backends** - How to provision
3. **Runtime Adapters** - What to deploy

All extensions use Go's `init()` function for auto-registration.

## Adding a Target

Create a new target to support a different cloud platform.

### Interface

```go
// target/target.go
type Target interface {
    Name() string
    Description() string
    Validate(cfg *config.DeployConfig) error
    ResourceSpec(cfg *config.DeployConfig) (*ResourceSpec, error)
}
```

### Example: Custom Target

```go
// target/mytarget/mytarget.go
package mytarget

import (
    "github.com/plexusone/omnideploy/config"
    "github.com/plexusone/omnideploy/target"
)

func init() {
    target.Register(&Target{})
}

type Target struct{}

func (t *Target) Name() string {
    return "mytarget"
}

func (t *Target) Description() string {
    return "My custom deployment target"
}

func (t *Target) Validate(cfg *config.DeployConfig) error {
    if err := cfg.Validate(); err != nil {
        return err
    }
    // Target-specific validation
    return nil
}

func (t *Target) ResourceSpec(cfg *config.DeployConfig) (*target.ResourceSpec, error) {
    if err := t.Validate(cfg); err != nil {
        return nil, err
    }

    resources := []target.Resource{
        {
            Type: "my_resource_type",
            Name: cfg.Name,
            Properties: map[string]any{
                "image":    cfg.Container.Image,
                "replicas": cfg.Service.Replicas,
            },
        },
    }

    return &target.ResourceSpec{
        StackName: cfg.Name,
        Region:    cfg.Region,
        Target:    t.Name(),
        Config:    cfg,
        Resources: resources,
        Outputs: []target.Output{
            {Name: "url", Description: "Service URL"},
        },
    }, nil
}
```

### Register the Target

Import in `cmd/omnideploy/main.go`:

```go
import (
    _ "github.com/plexusone/omnideploy/target/mytarget"
)
```

## Adding a Backend

Create a new backend to support a different IaC tool.

### Interface

```go
// backend/backend.go
type Backend interface {
    Name() string
    Description() string
    Apply(ctx context.Context, spec *target.ResourceSpec, opts ApplyOptions) (*Result, error)
    Preview(ctx context.Context, spec *target.ResourceSpec) (*PreviewResult, error)
    Destroy(ctx context.Context, stackName string, opts DestroyOptions) error
    Refresh(ctx context.Context, stackName string) error
}
```

### Example: Custom Backend

```go
// backend/mybackend/mybackend.go
package mybackend

import (
    "context"

    "github.com/plexusone/omnideploy/backend"
    "github.com/plexusone/omnideploy/target"
)

func init() {
    backend.Register(&Backend{})
}

type Backend struct{}

func (b *Backend) Name() string {
    return "mybackend"
}

func (b *Backend) Description() string {
    return "My custom IaC backend"
}

func (b *Backend) Apply(ctx context.Context, spec *target.ResourceSpec, opts backend.ApplyOptions) (*backend.Result, error) {
    // Provision resources based on spec
    for _, res := range spec.Resources {
        // Create resource using your IaC tool
    }

    return &backend.Result{
        StackName: spec.StackName,
        Outputs:   map[string]string{"url": "https://..."},
    }, nil
}

func (b *Backend) Preview(ctx context.Context, spec *target.ResourceSpec) (*backend.PreviewResult, error) {
    // Show what would change
    return &backend.PreviewResult{
        StackName: spec.StackName,
        Summary:   "1 create, 0 update, 0 delete",
    }, nil
}

func (b *Backend) Destroy(ctx context.Context, stackName string, opts backend.DestroyOptions) error {
    // Destroy resources
    return nil
}

func (b *Backend) Refresh(ctx context.Context, stackName string) error {
    // Refresh state
    return nil
}
```

## Adding a Runtime Adapter

Create a new adapter to support a different application format.

### Interface

```go
// runtime/adapter.go
type Adapter interface {
    Name() string
    Description() string
    Detect(path string) bool
    Load(path string) (*config.DeployConfig, error)
}
```

### Example: Custom Adapter

```go
// runtime/myapp/adapter.go
package myapp

import (
    "os"
    "path/filepath"
    "strings"

    "gopkg.in/yaml.v3"

    "github.com/plexusone/omnideploy/config"
    "github.com/plexusone/omnideploy/runtime"
)

func init() {
    runtime.Register(&Adapter{})
}

type Adapter struct{}

func (a *Adapter) Name() string {
    return "myapp"
}

func (a *Adapter) Description() string {
    return "MyApp framework configuration"
}

func (a *Adapter) Detect(path string) bool {
    base := strings.ToLower(filepath.Base(path))
    return strings.Contains(base, "myapp")
}

func (a *Adapter) Load(path string) (*config.DeployConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var myConfig MyAppConfig
    if err := yaml.Unmarshal(data, &myConfig); err != nil {
        return nil, err
    }

    // Convert to DeployConfig
    deployConfig := &config.DeployConfig{
        Name:   myConfig.AppName,
        Region: myConfig.Deploy.Region,
        Container: config.ContainerConfig{
            Image: myConfig.Deploy.Image,
            Ports: []config.PortMapping{
                {ContainerPort: myConfig.Port, Protocol: "HTTP"},
            },
        },
        Service: config.ServiceConfig{
            Replicas: myConfig.Deploy.Replicas,
            Public:   true,
        },
        Resources: config.ResourceConfig{
            Size: myConfig.Deploy.Size,
        },
        Environment: myConfig.Env,
    }

    deployConfig.Defaults()
    return deployConfig, nil
}

type MyAppConfig struct {
    AppName string            `yaml:"app_name"`
    Port    int               `yaml:"port"`
    Env     map[string]string `yaml:"env"`
    Deploy  struct {
        Image    string `yaml:"image"`
        Region   string `yaml:"region"`
        Replicas int    `yaml:"replicas"`
        Size     string `yaml:"size"`
    } `yaml:"deploy"`
}
```

## Testing Extensions

### Unit Tests

```go
func TestMyTarget_Validate(t *testing.T) {
    target := &mytarget.Target{}

    cfg := &config.DeployConfig{
        Name: "test",
        Container: config.ContainerConfig{
            Image: "nginx:latest",
            Ports: []config.PortMapping{{ContainerPort: 80}},
        },
    }

    err := target.Validate(cfg)
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
}
```

### Integration Tests

```go
func TestMyTarget_Deploy(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()
    deployer := deploy.New(
        deploy.WithTarget(&mytarget.Target{}),
        deploy.WithBackend(pulumi.New()),
    )

    cfg := &config.DeployConfig{...}
    result, err := deployer.DeployFromConfig(ctx, cfg, backend.ApplyOptions{})
    if err != nil {
        t.Fatal(err)
    }

    // Verify deployment
    if result.Outputs["url"] == "" {
        t.Error("expected URL output")
    }

    // Cleanup
    deployer.Destroy(ctx, cfg.Name, backend.DestroyOptions{})
}
```

## Best Practices

1. **Auto-register with init()**: Use init() for automatic registration
2. **Validate early**: Validate configuration before provisioning
3. **Handle errors gracefully**: Return meaningful error messages
4. **Support preview**: Implement Preview() for dry-run support
5. **Document requirements**: Document any external dependencies
6. **Test thoroughly**: Include unit and integration tests
