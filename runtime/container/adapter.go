// Package container provides a generic container runtime adapter.
package container

import (
	"path/filepath"
	"strings"

	"github.com/plexusone/omnideploy/config"
	"github.com/plexusone/omnideploy/runtime"
)

func init() {
	runtime.Register(&Adapter{})
}

// Adapter handles generic container deployments using omnideploy config format.
type Adapter struct{}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "container"
}

// Description returns a human-readable description.
func (a *Adapter) Description() string {
	return "Generic container deployment"
}

// Detect returns true if this is a generic omnideploy config.
func (a *Adapter) Detect(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.Contains(base, "deploy") || strings.Contains(base, "omnideploy")
}

// Load loads a generic deploy config.
func (a *Adapter) Load(path string) (*config.DeployConfig, error) {
	return config.Load(path)
}
