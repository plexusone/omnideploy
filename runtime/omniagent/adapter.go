// Package omniagent provides the OmniAgent runtime adapter.
package omniagent

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/plexusone/omnideploy/config"
	"github.com/plexusone/omnideploy/runtime"
)

func init() {
	runtime.Register(&Adapter{})
}

// Adapter converts OmniAgent configuration to DeployConfig.
type Adapter struct{}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "omniagent"
}

// Description returns a human-readable description.
func (a *Adapter) Description() string {
	return "OmniAgent gateway with web UI"
}

// Detect returns true if this looks like an OmniAgent config.
func (a *Adapter) Detect(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	if strings.Contains(base, "omniagent") {
		return true
	}

	// Check file contents for OmniAgent-specific keys
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return false
	}

	// OmniAgent configs typically have "gateway" or "agent" top-level keys
	_, hasGateway := raw["gateway"]
	_, hasAgent := raw["agent"]
	return hasGateway || hasAgent
}

// Load loads OmniAgent config and converts to DeployConfig.
func (a *Adapter) Load(path string) (*config.DeployConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var omniCfg OmniAgentConfig
	if err := yaml.Unmarshal(data, &omniCfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return a.convert(&omniCfg, path)
}

// convert converts OmniAgent config to DeployConfig.
func (a *Adapter) convert(cfg *OmniAgentConfig, path string) (*config.DeployConfig, error) {
	// Extract name from config or filename
	name := cfg.Deploy.Name
	if name == "" {
		base := filepath.Base(path)
		name = strings.TrimSuffix(base, filepath.Ext(base))
		if name == "omniagent" {
			name = "omniagent-gateway"
		}
	}

	// Determine image
	image := cfg.Deploy.Image
	if image == "" {
		image = "ghcr.io/plexusone/omniagent:latest"
	}

	// Determine port from gateway config
	port := 18789 // Default OmniAgent port
	if cfg.Gateway.Address != "" {
		// Parse port from address like "127.0.0.1:8080"
		if parts := strings.Split(cfg.Gateway.Address, ":"); len(parts) == 2 {
			if _, err := fmt.Sscanf(parts[1], "%d", &port); err != nil {
				slog.Warn("failed to parse port from gateway address, using default",
					"address", cfg.Gateway.Address,
					"default_port", port,
					"error", err)
			}
		}
	}

	// Build environment from config
	env := make(map[string]string)

	// Add LLM provider credentials
	if cfg.Agent.Provider != "" {
		env["OMNIAGENT_AGENT_PROVIDER"] = cfg.Agent.Provider
	}
	if cfg.Agent.Model != "" {
		env["OMNIAGENT_AGENT_MODEL"] = cfg.Agent.Model
	}

	// Add any explicitly set environment
	for k, v := range cfg.Deploy.Environment {
		env[k] = v
	}

	// Build secrets
	var secrets []config.SecretRef
	if cfg.Agent.APIKey != "" && strings.HasPrefix(cfg.Agent.APIKey, "${") {
		// Environment variable reference - convert to secret
		envVar := strings.TrimSuffix(strings.TrimPrefix(cfg.Agent.APIKey, "${"), "}")
		secrets = append(secrets, config.SecretRef{
			Name:   envVar,
			Source: "env:" + envVar, // Will be handled by deploy
		})
	}

	// Determine resources
	resources := config.ResourceConfig{
		Size: cfg.Deploy.Resources.Size,
	}
	if resources.Size == "" {
		resources.Size = "micro"
	}

	deployConfig := &config.DeployConfig{
		Name:    name,
		Version: cfg.Deploy.Version,
		Region:  cfg.Deploy.Region,
		Container: config.ContainerConfig{
			Image: image,
			Args:  []string{"gateway", "run"},
			Ports: []config.PortMapping{
				{ContainerPort: port, Protocol: "HTTP", Name: "http"},
			},
			HealthCheck: &config.HealthCheck{
				Path:               "/api/health",
				Port:               port,
				Interval:           30 * time.Second,
				Timeout:            5 * time.Second,
				HealthyThreshold:   2,
				UnhealthyThreshold: 3,
			},
		},
		Service: config.ServiceConfig{
			Replicas: cfg.Deploy.Replicas,
			Public:   true, // OmniAgent gateway is typically public
		},
		Resources:   resources,
		Environment: env,
		Secrets:     secrets,
		Tags: map[string]string{
			"app":        "omniagent",
			"managed-by": "omnideploy",
		},
	}

	deployConfig.Defaults()

	return deployConfig, nil
}

// OmniAgentConfig represents the OmniAgent configuration file structure.
type OmniAgentConfig struct {
	Gateway GatewayConfig `yaml:"gateway"`
	Agent   AgentConfig   `yaml:"agent"`
	Deploy  DeploySection `yaml:"deploy"`
}

// GatewayConfig is the gateway section.
type GatewayConfig struct {
	Address string   `yaml:"address"`
	APIKeys []string `yaml:"api_keys"`
}

// AgentConfig is the agent section.
type AgentConfig struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	APIKey   string `yaml:"api_key"`
}

// DeploySection is the deployment section (omnideploy-specific).
type DeploySection struct {
	Name        string            `yaml:"name"`
	Image       string            `yaml:"image"`
	Version     string            `yaml:"version"`
	Region      string            `yaml:"region"`
	Replicas    int               `yaml:"replicas"`
	Environment map[string]string `yaml:"environment"`
	Resources   struct {
		Size string `yaml:"size"`
	} `yaml:"resources"`
}
