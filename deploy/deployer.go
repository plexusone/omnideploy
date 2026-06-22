// Package deploy provides the deployment orchestrator.
package deploy

import (
	"context"
	"fmt"

	"github.com/plexusone/omnideploy/backend"
	"github.com/plexusone/omnideploy/config"
	"github.com/plexusone/omnideploy/runtime"
	"github.com/plexusone/omnideploy/target"
)

// Deployer orchestrates deployments.
type Deployer struct {
	runtime runtime.Adapter
	target  target.Target
	backend backend.Backend
}

// Option configures a Deployer.
type Option func(*Deployer)

// WithRuntime sets the runtime adapter.
func WithRuntime(r runtime.Adapter) Option {
	return func(d *Deployer) {
		d.runtime = r
	}
}

// WithTarget sets the deployment target.
func WithTarget(t target.Target) Option {
	return func(d *Deployer) {
		d.target = t
	}
}

// WithBackend sets the IaC backend.
func WithBackend(b backend.Backend) Option {
	return func(d *Deployer) {
		d.backend = b
	}
}

// New creates a new Deployer.
func New(opts ...Option) *Deployer {
	d := &Deployer{}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Plan generates a deployment plan from a config file.
func (d *Deployer) Plan(ctx context.Context, configPath string) (*target.ResourceSpec, error) {
	// Load config using runtime adapter
	cfg, err := d.loadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// Generate resource spec using target
	spec, err := d.target.ResourceSpec(cfg)
	if err != nil {
		return nil, fmt.Errorf("generating resource spec: %w", err)
	}

	return spec, nil
}

// Apply deploys using the backend.
func (d *Deployer) Apply(ctx context.Context, configPath string, opts backend.ApplyOptions) (*backend.Result, error) {
	spec, err := d.Plan(ctx, configPath)
	if err != nil {
		return nil, err
	}

	return d.backend.Apply(ctx, spec, opts)
}

// Preview shows what would be deployed.
func (d *Deployer) Preview(ctx context.Context, configPath string) (*backend.PreviewResult, error) {
	spec, err := d.Plan(ctx, configPath)
	if err != nil {
		return nil, err
	}

	return d.backend.Preview(ctx, spec)
}

// Destroy removes a deployment.
func (d *Deployer) Destroy(ctx context.Context, stackName string, opts backend.DestroyOptions) error {
	return d.backend.Destroy(ctx, stackName, opts)
}

// loadConfig loads config using the runtime adapter.
func (d *Deployer) loadConfig(path string) (*config.DeployConfig, error) {
	if d.runtime != nil {
		return d.runtime.Load(path)
	}

	// Try to auto-detect runtime
	adapter, err := runtime.Detect(path)
	if err != nil {
		// Fall back to generic config
		return config.Load(path)
	}

	return adapter.Load(path)
}

// DeployFromConfig deploys directly from a DeployConfig.
func (d *Deployer) DeployFromConfig(ctx context.Context, cfg *config.DeployConfig, opts backend.ApplyOptions) (*backend.Result, error) {
	spec, err := d.target.ResourceSpec(cfg)
	if err != nil {
		return nil, fmt.Errorf("generating resource spec: %w", err)
	}

	return d.backend.Apply(ctx, spec, opts)
}
