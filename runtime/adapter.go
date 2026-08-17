// Package runtime defines runtime adapters that convert application configs to deployment configs.
package runtime

import (
	"fmt"
	"sort"

	"github.com/plexusone/omnideploy/config"
)

// genericAdapterName is the catch-all adapter ("Generic container
// deployment") whose Detect heuristic is intentionally broad (e.g. any
// filename containing "deploy"). Detect checks it last, after every
// other (more specific) adapter, so a config that also matches a
// specialized adapter's content-based signal — e.g. an omniagent config
// named "deploy.yaml" — resolves to the specialized adapter instead of
// colliding with the generic one.
const genericAdapterName = "container"

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
//
// Registered adapters are checked in a deterministic order — sorted by
// name, with genericAdapterName always last — rather than Go's randomized
// map iteration order, which previously made the outcome nondeterministic
// (varying from run to run) whenever more than one adapter's Detect
// matched the same file.
func Detect(path string) (Adapter, error) {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if names[i] == genericAdapterName {
			return false
		}
		if names[j] == genericAdapterName {
			return true
		}
		return names[i] < names[j]
	})

	for _, name := range names {
		if a := registry[name]; a.Detect(path) {
			return a, nil
		}
	}
	return nil, fmt.Errorf("no adapter detected for: %s", path)
}
