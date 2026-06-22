// Package target defines deployment target abstractions.
package target

import (
	"fmt"

	"github.com/plexusone/omnideploy/config"
)

// Target represents a deployment target (where to deploy).
type Target interface {
	// Name returns the target name (e.g., "lightsail", "ecs", "agentcore")
	Name() string

	// Description returns a human-readable description
	Description() string

	// Validate validates the configuration for this target
	Validate(cfg *config.DeployConfig) error

	// ResourceSpec generates the resource specification for this target
	ResourceSpec(cfg *config.DeployConfig) (*ResourceSpec, error)
}

// ResourceSpec defines the resources needed for a deployment.
// This is the bridge between Target and Backend.
type ResourceSpec struct {
	// StackName is the unique name for this deployment stack
	StackName string

	// Region is the deployment region
	Region string

	// Target is the target name
	Target string

	// Config is the original deployment config
	Config *config.DeployConfig

	// Resources are the abstract resource definitions
	Resources []Resource

	// Outputs are the expected outputs after deployment
	Outputs []Output
}

// Resource represents an abstract cloud resource.
type Resource struct {
	// Type is the resource type (e.g., "container_service", "load_balancer")
	Type string

	// Name is the resource name
	Name string

	// Properties are resource-specific properties
	Properties map[string]any
}

// Output represents an expected deployment output.
type Output struct {
	// Name is the output name (e.g., "url", "ip_address")
	Name string

	// Description describes the output
	Description string
}

// Registry holds registered targets.
var registry = make(map[string]Target)

// Register registers a target.
func Register(t Target) {
	registry[t.Name()] = t
}

// Get returns a registered target by name.
func Get(name string) (Target, error) {
	t, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown target: %s", name)
	}
	return t, nil
}

// List returns all registered target names.
func List() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// All returns all registered targets.
func All() map[string]Target {
	result := make(map[string]Target, len(registry))
	for k, v := range registry {
		result[k] = v
	}
	return result
}
