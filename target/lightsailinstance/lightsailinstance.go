// Package lightsailinstance implements the AWS Lightsail Instance target:
// a pre-built binary deployed to a persistent Lightsail VM as a systemd
// service, backed by the instance's included disk. An alternative to the
// lightsail (Container Service) target for deployments that need cheap,
// capped-cost local persistence rather than a container's ephemeral
// filesystem — see docs/specs/{PRD,TRD}.md.
package lightsailinstance

import (
	"fmt"

	"github.com/plexusone/omnideploy/config"
	"github.com/plexusone/omnideploy/target"
)

func init() {
	target.Register(&Target{})
}

// Target implements the AWS Lightsail Instance target.
type Target struct{}

// Name returns the target name.
func (t *Target) Name() string {
	return "lightsail-instance"
}

// Description returns a human-readable description.
func (t *Target) Description() string {
	return "AWS Lightsail Instance - a persistent VM running a pre-built binary as a systemd service, for deployments needing cheap local disk persistence"
}

// Validate validates the configuration for a Lightsail Instance
// deployment. Deliberately does not enumerate valid Blueprint/Bundle
// values (unlike the lightsail container target's fixed power-size set,
// Lightsail's instance bundle catalog is large, versioned, and changes
// over time — AWS's own API is the source of truth for whether a given
// blueprint/bundle/region combination is valid, not a hardcoded list
// here that would go stale).
func (t *Target) Validate(cfg *config.DeployConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if cfg.Instance == nil {
		return fmt.Errorf("lightsail-instance target requires config.instance to be set")
	}
	return nil
}

// ResourceSpec generates the resource specification for a Lightsail
// Instance deployment. The abstract Resources/Outputs below describe the
// deployment for informational purposes (e.g. `omnideploy plan`-style
// output); the actual Pulumi resource graph is built directly from
// spec.Config by the backend, mirroring the lightsail (container)
// target's existing division of responsibility.
func (t *Target) ResourceSpec(cfg *config.DeployConfig) (*target.ResourceSpec, error) {
	if err := t.Validate(cfg); err != nil {
		return nil, err
	}

	inst := cfg.Instance
	resources := []target.Resource{
		{
			Type: "lightsail_key_pair",
			Name: cfg.Name + "-keypair",
			Properties: map[string]any{
				"name": cfg.Name,
			},
		},
		{
			Type: "lightsail_instance",
			Name: cfg.Name,
			Properties: map[string]any{
				"name":      cfg.Name,
				"blueprint": inst.Blueprint,
				"bundle":    inst.Bundle,
				"key_pair":  cfg.Name,
				"tags":      cfg.Tags,
			},
		},
		{
			Type: "lightsail_instance_public_ports",
			Name: cfg.Name + "-firewall",
			Properties: map[string]any{
				"instance_name": cfg.Name,
				"ssh_only":      inst.HealthCheck == nil || inst.HealthCheck.Kind != "http",
				"app_port":      healthCheckPort(inst),
			},
		},
		{
			Type: "remote_copy_to_remote",
			Name: cfg.Name + "-artifact",
			Properties: map[string]any{
				"local_path":  inst.BinaryPath,
				"remote_path": inst.RemotePath,
			},
		},
		{
			Type: "remote_command",
			Name: cfg.Name + "-service-install",
			Properties: map[string]any{
				"service_name": inst.ServiceName,
				"remote_path":  inst.RemotePath,
			},
		},
	}

	outputs := []target.Output{
		{Name: "public_ip", Description: "Instance public IP address"},
		{Name: "state", Description: "Instance state"},
	}

	return &target.ResourceSpec{
		StackName: cfg.Name,
		Region:    cfg.Region,
		Target:    t.Name(),
		Config:    cfg,
		Resources: resources,
		Outputs:   outputs,
	}, nil
}

// healthCheckPort returns the port to open in the instance firewall
// alongside SSH, or 0 if no HTTP health check (and therefore no app
// port) is configured — keeping the default firewall posture SSH-only.
func healthCheckPort(inst *config.InstanceConfig) int {
	if inst.HealthCheck != nil && inst.HealthCheck.Kind == "http" {
		return inst.HealthCheck.Port
	}
	return 0
}
