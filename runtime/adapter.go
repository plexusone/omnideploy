// Package runtime defines runtime adapters that convert application configs to deployment configs.
package runtime

import (
	"fmt"

	"github.com/plexusone/omnideploy/config"
)

// Adapter converts application-specific configuration to universal DeployConfig.
type Adapter interface {
	// Name returns the adapter name (e.g., "omniagent", "agentkit", "container")
	Name() string

	// Description returns a human-readable description
	Description() string

	// Load loads and converts application config to DeployConfig
	Load(path string) (*config.DeployConfig, error)

	// Detect returns true if this adapter can handle the given config file
	Detect(path string) bool
}

// Registry holds registered adapters.
var registry = make(map[string]Adapter)

// Register registers an adapter.
func Register(a Adapter) {
	registry[a.Name()] = a
}

// Get returns a registered adapter by name.
func Get(name string) (Adapter, error) {
	a, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown runtime adapter: %s", name)
	}
	return a, nil
}

// List returns all registered adapter names.
func List() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// All returns all registered adapters.
func All() map[string]Adapter {
	result := make(map[string]Adapter, len(registry))
	for k, v := range registry {
		result[k] = v
	}
	return result
}

// Detect tries to detect the appropriate adapter for a config file.
func Detect(path string) (Adapter, error) {
	for _, a := range registry {
		if a.Detect(path) {
			return a, nil
		}
	}
	return nil, fmt.Errorf("no adapter detected for: %s", path)
}
