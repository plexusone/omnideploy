package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load loads a deployment configuration from a file.
func Load(path string) (*DeployConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	return Parse(data, filepath.Ext(path))
}

// Parse parses deployment configuration from bytes.
func Parse(data []byte, format string) (*DeployConfig, error) {
	var cfg DeployConfig

	format = strings.TrimPrefix(strings.ToLower(format), ".")

	switch format {
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing YAML: %w", err)
		}
	case "json":
		if err := yaml.Unmarshal(data, &cfg); err != nil { // yaml.v3 handles JSON
			return nil, fmt.Errorf("parsing JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	cfg.Defaults()

	return &cfg, nil
}

// Validate validates the deployment configuration.
func (c *DeployConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.Container.Image == "" {
		return fmt.Errorf("container.image is required")
	}
	if len(c.Container.Ports) == 0 {
		return fmt.Errorf("at least one container port is required")
	}
	for i, port := range c.Container.Ports {
		if port.ContainerPort <= 0 || port.ContainerPort > 65535 {
			return fmt.Errorf("container.ports[%d].container_port must be between 1 and 65535", i)
		}
	}
	if c.Service.Replicas < 0 {
		return fmt.Errorf("service.replicas cannot be negative")
	}
	return nil
}
