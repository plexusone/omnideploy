// Package lightsail implements the AWS LightSail Container Service target.
package lightsail

import (
	"fmt"

	"github.com/plexusone/omnideploy/config"
	"github.com/plexusone/omnideploy/target"
)

func init() {
	target.Register(&Target{})
}

// Target implements the AWS LightSail Container Service target.
type Target struct{}

// Name returns the target name.
func (t *Target) Name() string {
	return "lightsail"
}

// Description returns a human-readable description.
func (t *Target) Description() string {
	return "AWS LightSail Container Service - simple, cost-effective container hosting"
}

// Validate validates the configuration for LightSail.
func (t *Target) Validate(cfg *config.DeployConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	// LightSail-specific validation
	size := cfg.Resources.Size
	if !isValidPower(size) {
		return fmt.Errorf("invalid resource size for LightSail: %s (valid: nano, micro, small, medium, large, xlarge)", size)
	}

	// LightSail supports 1-20 replicas
	if cfg.Service.Replicas > 20 {
		return fmt.Errorf("LightSail supports max 20 replicas, got %d", cfg.Service.Replicas)
	}

	return nil
}

// ResourceSpec generates the resource specification for LightSail.
func (t *Target) ResourceSpec(cfg *config.DeployConfig) (*target.ResourceSpec, error) {
	if err := t.Validate(cfg); err != nil {
		return nil, err
	}

	// Map size to LightSail power
	power := sizeToPower(cfg.Resources.Size)

	// Build container definition
	containerDef := map[string]any{
		"image":       cfg.Container.Image,
		"command":     cfg.Container.Args,
		"environment": cfg.Environment,
		"ports":       buildPorts(cfg.Container.Ports),
	}

	// Build public endpoint if service is public
	var publicEndpoint map[string]any
	if cfg.Service.Public && len(cfg.Container.Ports) > 0 {
		publicEndpoint = map[string]any{
			"container_name": cfg.Name,
			"container_port": cfg.Container.Ports[0].ContainerPort,
		}
		if cfg.Container.HealthCheck != nil {
			publicEndpoint["health_check"] = map[string]any{
				"path":                cfg.Container.HealthCheck.Path,
				"interval_seconds":    int(cfg.Container.HealthCheck.Interval.Seconds()),
				"timeout_seconds":     int(cfg.Container.HealthCheck.Timeout.Seconds()),
				"healthy_threshold":   cfg.Container.HealthCheck.HealthyThreshold,
				"unhealthy_threshold": cfg.Container.HealthCheck.UnhealthyThreshold,
			}
		}
	}

	resources := []target.Resource{
		{
			Type: "lightsail_container_service",
			Name: cfg.Name,
			Properties: map[string]any{
				"name":        cfg.Name,
				"power":       power,
				"scale":       cfg.Service.Replicas,
				"is_disabled": false,
				"tags":        cfg.Tags,
			},
		},
		{
			Type: "lightsail_container_deployment",
			Name: cfg.Name + "-deployment",
			Properties: map[string]any{
				"service_name": cfg.Name,
				"containers": map[string]any{
					cfg.Name: containerDef,
				},
				"public_endpoint": publicEndpoint,
			},
		},
	}

	outputs := []target.Output{
		{Name: "url", Description: "Service URL"},
		{Name: "state", Description: "Service state"},
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

// sizeToPower maps resource size to LightSail power.
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

// isValidPower checks if a size is valid for LightSail.
func isValidPower(size string) bool {
	switch size {
	case "", "nano", "micro", "small", "medium", "large", "xlarge":
		return true
	default:
		return false
	}
}

// buildPorts converts port mappings to LightSail format.
func buildPorts(ports []config.PortMapping) map[string]string {
	result := make(map[string]string)
	for _, p := range ports {
		key := fmt.Sprintf("%d", p.ContainerPort)
		result[key] = p.Protocol
	}
	return result
}
