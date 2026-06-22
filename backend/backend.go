// Package backend defines IaC backend abstractions.
package backend

import (
	"context"
	"fmt"

	"github.com/plexusone/omnideploy/target"
)

// Backend represents an IaC backend (how to provision).
type Backend interface {
	// Name returns the backend name (e.g., "pulumi", "cdk", "terraform")
	Name() string

	// Description returns a human-readable description
	Description() string

	// Apply provisions the resources defined in the spec
	Apply(ctx context.Context, spec *target.ResourceSpec, opts ApplyOptions) (*Result, error)

	// Preview shows what would be provisioned without making changes
	Preview(ctx context.Context, spec *target.ResourceSpec) (*PreviewResult, error)

	// Destroy removes all resources in the stack
	Destroy(ctx context.Context, stackName string, opts DestroyOptions) error

	// Refresh refreshes the state from the cloud provider
	Refresh(ctx context.Context, stackName string) error
}

// ApplyOptions configures the Apply operation.
type ApplyOptions struct {
	// StackName overrides the default stack name
	StackName string

	// DryRun shows what would be done without making changes
	DryRun bool

	// AutoApprove skips confirmation prompts
	AutoApprove bool

	// Parallel sets the number of parallel operations
	Parallel int

	// OnOutput is called with output messages
	OnOutput func(string)
}

// DestroyOptions configures the Destroy operation.
type DestroyOptions struct {
	// AutoApprove skips confirmation prompts
	AutoApprove bool

	// OnOutput is called with output messages
	OnOutput func(string)
}

// Result contains the result of an Apply operation.
type Result struct {
	// StackName is the stack that was deployed
	StackName string

	// Outputs are the deployment outputs
	Outputs map[string]string

	// ResourcesCreated is the number of resources created
	ResourcesCreated int

	// ResourcesUpdated is the number of resources updated
	ResourcesUpdated int

	// ResourcesDeleted is the number of resources deleted
	ResourcesDeleted int

	// Duration is how long the deployment took
	Duration string
}

// PreviewResult contains the result of a Preview operation.
type PreviewResult struct {
	// StackName is the stack that would be deployed
	StackName string

	// Changes describes what would change
	Changes []Change

	// Summary is a human-readable summary
	Summary string
}

// Change represents a resource change.
type Change struct {
	// Type is the change type (create, update, delete, replace)
	Type string

	// ResourceType is the type of resource
	ResourceType string

	// ResourceName is the name of the resource
	ResourceName string

	// Details provides additional change details
	Details string
}

// Registry holds registered backends.
var registry = make(map[string]Backend)

// Register registers a backend.
func Register(b Backend) {
	registry[b.Name()] = b
}

// Get returns a registered backend by name.
func Get(name string) (Backend, error) {
	b, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown backend: %s", name)
	}
	return b, nil
}

// List returns all registered backend names.
func List() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// All returns all registered backends.
func All() map[string]Backend {
	result := make(map[string]Backend, len(registry))
	for k, v := range registry {
		result[k] = v
	}
	return result
}
