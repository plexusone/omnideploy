// Package pulumi implements the Pulumi IaC backend.
package pulumi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lightsail"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optrefresh"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/plexusone/omnideploy/backend"
	"github.com/plexusone/omnideploy/config"
	"github.com/plexusone/omnideploy/target"
)

func init() {
	backend.Register(New())
}

// Backend implements the Pulumi IaC backend.
type Backend struct {
	// WorkDir is the working directory for Pulumi state
	WorkDir string
}

// New creates a new Pulumi backend.
func New() *Backend {
	workDir := os.Getenv("OMNIDEPLOY_WORK_DIR")
	if workDir == "" {
		home, _ := os.UserHomeDir()
		workDir = filepath.Join(home, ".omnideploy", "pulumi")
	}
	return &Backend{WorkDir: workDir}
}

// Name returns the backend name.
func (b *Backend) Name() string {
	return "pulumi"
}

// Description returns a human-readable description.
func (b *Backend) Description() string {
	return "Pulumi - Infrastructure as Code using Go"
}

// Apply provisions resources using Pulumi.
func (b *Backend) Apply(ctx context.Context, spec *target.ResourceSpec, opts backend.ApplyOptions) (*backend.Result, error) {
	stackName := opts.StackName
	if stackName == "" {
		stackName = spec.StackName
	}

	// Create the Pulumi program
	program := b.createProgram(spec)

	// Create or select the stack
	stack, err := b.getOrCreateStack(ctx, stackName, spec.Config.Name, program)
	if err != nil {
		return nil, fmt.Errorf("creating stack: %w", err)
	}

	// Set AWS region
	if err := stack.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: spec.Region}); err != nil {
		return nil, fmt.Errorf("setting region: %w", err)
	}

	// Run update
	var upOpts []optup.Option
	if opts.OnOutput != nil {
		upOpts = append(upOpts, optup.ProgressStreams(writerFunc(opts.OnOutput)))
	}

	result, err := stack.Up(ctx, upOpts...)
	if err != nil {
		return nil, fmt.Errorf("deploying: %w", err)
	}

	// Extract outputs
	outputs := make(map[string]string)
	for k, v := range result.Outputs {
		if s, ok := v.Value.(string); ok {
			outputs[k] = s
		}
	}

	// Extract resource changes
	var created, updated, deleted int
	if result.Summary.ResourceChanges != nil {
		changes := *result.Summary.ResourceChanges
		created = changes["create"]
		updated = changes["update"]
		deleted = changes["delete"]
	}

	return &backend.Result{
		StackName:        stackName,
		Outputs:          outputs,
		ResourcesCreated: created,
		ResourcesUpdated: updated,
		ResourcesDeleted: deleted,
	}, nil
}

// Preview shows what would be provisioned.
func (b *Backend) Preview(ctx context.Context, spec *target.ResourceSpec) (*backend.PreviewResult, error) {
	program := b.createProgram(spec)

	stack, err := b.getOrCreateStack(ctx, spec.StackName, spec.Config.Name, program)
	if err != nil {
		return nil, fmt.Errorf("creating stack: %w", err)
	}

	if err := stack.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: spec.Region}); err != nil {
		return nil, fmt.Errorf("setting region: %w", err)
	}

	result, err := stack.Preview(ctx, optpreview.ProgressStreams(os.Stdout))
	if err != nil {
		return nil, fmt.Errorf("previewing: %w", err)
	}

	// Build summary from ChangeSummary
	var created, updated, deleted int
	if result.ChangeSummary != nil {
		created = result.ChangeSummary["create"]
		updated = result.ChangeSummary["update"]
		deleted = result.ChangeSummary["delete"]
	}

	return &backend.PreviewResult{
		StackName: spec.StackName,
		Changes:   []backend.Change{}, // Detailed steps not available in this API version
		Summary:   fmt.Sprintf("Changes: %d create, %d update, %d delete", created, updated, deleted),
	}, nil
}

// Destroy removes all resources.
func (b *Backend) Destroy(ctx context.Context, stackName string, opts backend.DestroyOptions) error {
	stack, err := auto.SelectStackLocalSource(ctx, stackName, b.WorkDir)
	if err != nil {
		return fmt.Errorf("selecting stack: %w", err)
	}

	var destroyOpts []optdestroy.Option
	if opts.OnOutput != nil {
		destroyOpts = append(destroyOpts, optdestroy.ProgressStreams(writerFunc(opts.OnOutput)))
	}

	_, err = stack.Destroy(ctx, destroyOpts...)
	if err != nil {
		return fmt.Errorf("destroying: %w", err)
	}

	return nil
}

// Refresh refreshes the state.
func (b *Backend) Refresh(ctx context.Context, stackName string) error {
	stack, err := auto.SelectStackLocalSource(ctx, stackName, b.WorkDir)
	if err != nil {
		return fmt.Errorf("selecting stack: %w", err)
	}

	_, err = stack.Refresh(ctx, optrefresh.ProgressStreams(os.Stdout))
	if err != nil {
		return fmt.Errorf("refreshing: %w", err)
	}

	return nil
}

// getOrCreateStack creates or selects a Pulumi stack.
func (b *Backend) getOrCreateStack(ctx context.Context, stackName, projectName string, program pulumi.RunFunc) (auto.Stack, error) {
	// Ensure work directory exists
	if err := os.MkdirAll(b.WorkDir, 0o755); err != nil {
		return auto.Stack{}, fmt.Errorf("creating work dir: %w", err)
	}

	// Try to create a new stack, or select existing
	stack, err := auto.UpsertStackInlineSource(ctx, stackName, projectName, program,
		auto.WorkDir(b.WorkDir),
	)
	if err != nil {
		return auto.Stack{}, err
	}

	return stack, nil
}

// createProgram creates the Pulumi program for the given spec.
func (b *Backend) createProgram(spec *target.ResourceSpec) pulumi.RunFunc {
	return func(ctx *pulumi.Context) error {
		switch spec.Target {
		case "lightsail":
			return b.deployLightsail(ctx, spec)
		default:
			return fmt.Errorf("unsupported target: %s", spec.Target)
		}
	}
}

// deployLightsail deploys to AWS LightSail.
func (b *Backend) deployLightsail(ctx *pulumi.Context, spec *target.ResourceSpec) error {
	cfg := spec.Config

	// Create container service
	service, err := lightsail.NewContainerService(ctx, cfg.Name, &lightsail.ContainerServiceArgs{
		Name:       pulumi.String(cfg.Name),
		Power:      pulumi.String(sizeToPower(cfg.Resources.Size)),
		Scale:      pulumi.Int(cfg.Service.Replicas),
		IsDisabled: pulumi.Bool(false),
		Tags:       pulumi.ToStringMap(cfg.Tags),
	})
	if err != nil {
		return fmt.Errorf("creating container service: %w", err)
	}

	// Build container definition
	containers := lightsail.ContainerServiceDeploymentVersionContainerArray{
		&lightsail.ContainerServiceDeploymentVersionContainerArgs{
			ContainerName: pulumi.String(cfg.Name),
			Image:         pulumi.String(cfg.Container.Image),
			Commands:      pulumi.ToStringArray(cfg.Container.Args),
			Environment:   pulumi.ToStringMap(cfg.Environment),
			Ports:         buildPortMap(cfg.Container.Ports),
		},
	}

	// Build public endpoint
	var publicEndpoint *lightsail.ContainerServiceDeploymentVersionPublicEndpointArgs
	if cfg.Service.Public && len(cfg.Container.Ports) > 0 {
		endpointArgs := &lightsail.ContainerServiceDeploymentVersionPublicEndpointArgs{
			ContainerName: pulumi.String(cfg.Name),
			ContainerPort: pulumi.Int(cfg.Container.Ports[0].ContainerPort),
		}

		if cfg.Container.HealthCheck != nil && cfg.Container.HealthCheck.Path != "" {
			endpointArgs.HealthCheck = &lightsail.ContainerServiceDeploymentVersionPublicEndpointHealthCheckArgs{
				Path:               pulumi.String(cfg.Container.HealthCheck.Path),
				IntervalSeconds:    pulumi.Int(int(cfg.Container.HealthCheck.Interval.Seconds())),
				TimeoutSeconds:     pulumi.Int(int(cfg.Container.HealthCheck.Timeout.Seconds())),
				HealthyThreshold:   pulumi.Int(cfg.Container.HealthCheck.HealthyThreshold),
				UnhealthyThreshold: pulumi.Int(cfg.Container.HealthCheck.UnhealthyThreshold),
			}
		}

		publicEndpoint = endpointArgs
	}

	// Create deployment
	deployment, err := lightsail.NewContainerServiceDeploymentVersion(ctx, cfg.Name+"-deployment", &lightsail.ContainerServiceDeploymentVersionArgs{
		ServiceName:    service.Name,
		Containers:     containers,
		PublicEndpoint: publicEndpoint,
	})
	if err != nil {
		return fmt.Errorf("creating deployment: %w", err)
	}

	// Export outputs
	ctx.Export("url", service.Url)
	ctx.Export("state", deployment.State)
	ctx.Export("service_name", service.Name)

	return nil
}

// sizeToPower maps size to LightSail power.
func sizeToPower(size string) string {
	switch size {
	case "nano":
		return "nano"
	case "micro", "":
		return "micro"
	case "small":
		return "small"
	case "medium":
		return "medium"
	case "large":
		return "large"
	case "xlarge":
		return "xlarge"
	default:
		return "micro"
	}
}

// buildPortMap converts ports to Pulumi string map (port -> protocol).
func buildPortMap(ports []config.PortMapping) pulumi.StringMap {
	result := pulumi.StringMap{}
	for _, p := range ports {
		key := fmt.Sprintf("%d", p.ContainerPort)
		result[key] = pulumi.String(p.Protocol)
	}
	return result
}

// writerFunc adapts a callback to io.Writer.
type writerFunc func(string)

func (f writerFunc) Write(p []byte) (n int, err error) {
	f(string(p))
	return len(p), nil
}
