// Package config defines the universal deployment configuration schema.
package config

import "time"

// DeployConfig is the universal deployment configuration that all targets understand.
type DeployConfig struct {
	// Name is the deployment/stack name
	Name string `yaml:"name" json:"name"`

	// Version is the deployment version (used for tagging)
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// Region is the cloud region to deploy to
	Region string `yaml:"region,omitempty" json:"region,omitempty"`

	// Container defines the container configuration. Exactly one of
	// Container/Instance is expected — Validate rejects both-set or
	// neither-set. Container.Image=="" (its zero value) is how a
	// lightsail-instance deploy leaves this unset.
	Container ContainerConfig `yaml:"container,omitempty" json:"container,omitempty"`

	// Instance defines a persistent-VM deployment (e.g. the
	// lightsail-instance target) as an alternative to Container — a
	// pre-built binary running as a systemd service, backed by the
	// target's included persistent disk rather than a container's
	// ephemeral filesystem.
	Instance *InstanceConfig `yaml:"instance,omitempty" json:"instance,omitempty"`

	// Service defines the service configuration
	Service ServiceConfig `yaml:"service,omitempty" json:"service,omitempty"`

	// Resources defines compute resource requirements
	Resources ResourceConfig `yaml:"resources,omitempty" json:"resources,omitempty"`

	// Environment variables
	Environment map[string]string `yaml:"environment,omitempty" json:"environment,omitempty"`

	// Secrets references (provider-specific)
	Secrets []SecretRef `yaml:"secrets,omitempty" json:"secrets,omitempty"`

	// Tags for resource tagging
	Tags map[string]string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// ContainerConfig defines container settings.
type ContainerConfig struct {
	// Image is the container image (e.g., "ghcr.io/plexusone/omniagent:latest")
	Image string `yaml:"image" json:"image"`

	// Command overrides the container entrypoint
	Command []string `yaml:"command,omitempty" json:"command,omitempty"`

	// Args are arguments to the entrypoint
	Args []string `yaml:"args,omitempty" json:"args,omitempty"`

	// Ports to expose
	Ports []PortMapping `yaml:"ports,omitempty" json:"ports,omitempty"`

	// HealthCheck configuration
	HealthCheck *HealthCheck `yaml:"health_check,omitempty" json:"health_check,omitempty"`

	// WorkingDir sets the working directory
	WorkingDir string `yaml:"working_dir,omitempty" json:"working_dir,omitempty"`
}

// PortMapping defines a port exposure.
type PortMapping struct {
	// ContainerPort is the port inside the container
	ContainerPort int `yaml:"container_port" json:"container_port"`

	// Protocol is TCP or UDP (default: TCP)
	Protocol string `yaml:"protocol,omitempty" json:"protocol,omitempty"`

	// Name is an optional name for the port
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
}

// HealthCheck defines health check configuration.
type HealthCheck struct {
	// Path is the HTTP path to check (e.g., "/health")
	Path string `yaml:"path,omitempty" json:"path,omitempty"`

	// Port is the port to check (defaults to first exposed port)
	Port int `yaml:"port,omitempty" json:"port,omitempty"`

	// Interval between checks
	Interval time.Duration `yaml:"interval,omitempty" json:"interval,omitempty"`

	// Timeout for each check
	Timeout time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`

	// HealthyThreshold is the number of consecutive successes required
	HealthyThreshold int `yaml:"healthy_threshold,omitempty" json:"healthy_threshold,omitempty"`

	// UnhealthyThreshold is the number of consecutive failures required
	UnhealthyThreshold int `yaml:"unhealthy_threshold,omitempty" json:"unhealthy_threshold,omitempty"`
}

// InstanceConfig defines a persistent-VM deployment: a pre-built binary
// installed as a systemd service on the target's instance, with the
// instance's included disk as the storage layer.
type InstanceConfig struct {
	// Blueprint is the Lightsail OS image ID (e.g. "ubuntu_22_04"). See
	// `aws lightsail get-blueprints`.
	Blueprint string `yaml:"blueprint" json:"blueprint"`

	// Bundle is the Lightsail instance bundle ID (e.g. "nano_3_0"),
	// determining vCPU/memory/disk/price. See `aws lightsail
	// get-bundles`. Deliberately has no default — silently picking a
	// bundle size (unlike a container's Resources.Size) risks choosing
	// one invalid for the given Blueprint.
	Bundle string `yaml:"bundle" json:"bundle"`

	// BinaryPath is the local path to the pre-built binary to deploy
	// (cross-compiled by the caller, e.g. CGO_ENABLED=0 GOOS=linux
	// GOARCH=amd64 go build) — the target ships it via SSH, it does not
	// build it.
	BinaryPath string `yaml:"binary_path" json:"binary_path"`

	// RemotePath is the directory on the instance the binary (and its
	// config/.env) are installed to, e.g. "/opt/omniagent".
	RemotePath string `yaml:"remote_path" json:"remote_path"`

	// ServiceName is the systemd unit name (without ".service").
	ServiceName string `yaml:"service_name" json:"service_name"`

	// SystemdUnit optionally overrides the generated systemd unit file
	// content. Leave empty to use the generated default (binary at
	// RemotePath, EnvironmentFile=RemotePath/.env, Restart=always).
	SystemdUnit string `yaml:"systemd_unit,omitempty" json:"systemd_unit,omitempty"`

	// HealthCheck configures post-deploy verification. Nil skips
	// verification entirely — the right choice for a tool with no HTTP
	// surface and no meaningful systemd-active signal beyond "it ran."
	HealthCheck *VMHealthCheck `yaml:"health_check,omitempty" json:"health_check,omitempty"`
}

// VMHealthCheck configures post-deploy verification for an Instance
// deployment. Exactly one Kind is used.
type VMHealthCheck struct {
	// Kind selects the verification method: "http" (probe an HTTP
	// endpoint the app serves) or "systemd" (confirm the unit is active
	// over SSH, for tools with no HTTP surface).
	Kind string `yaml:"kind" json:"kind"`

	// Path is the HTTP path to probe. Only used when Kind == "http".
	Path string `yaml:"path,omitempty" json:"path,omitempty"`

	// Port is the HTTP port to probe, and (when set) the port opened in
	// the instance firewall alongside SSH. Only used when Kind == "http".
	Port int `yaml:"port,omitempty" json:"port,omitempty"`
}

// ServiceConfig defines service-level settings.
type ServiceConfig struct {
	// Replicas is the number of container instances
	Replicas int `yaml:"replicas,omitempty" json:"replicas,omitempty"`

	// Domains are custom domain names
	Domains []string `yaml:"domains,omitempty" json:"domains,omitempty"`

	// TLS configuration
	TLS *TLSConfig `yaml:"tls,omitempty" json:"tls,omitempty"`

	// Public determines if the service is publicly accessible
	Public bool `yaml:"public,omitempty" json:"public,omitempty"`
}

// TLSConfig defines TLS/HTTPS settings.
type TLSConfig struct {
	// Enabled enables TLS
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// CertificateARN is the ARN of an ACM certificate (AWS-specific)
	CertificateARN string `yaml:"certificate_arn,omitempty" json:"certificate_arn,omitempty"`

	// AutoCert enables automatic certificate provisioning
	AutoCert bool `yaml:"auto_cert,omitempty" json:"auto_cert,omitempty"`
}

// ResourceConfig defines compute resource requirements.
type ResourceConfig struct {
	// Size is a preset size (e.g., "micro", "small", "medium", "large")
	Size string `yaml:"size,omitempty" json:"size,omitempty"`

	// CPU in millicores (e.g., 256 = 0.25 vCPU)
	CPU int `yaml:"cpu,omitempty" json:"cpu,omitempty"`

	// Memory in MB
	Memory int `yaml:"memory,omitempty" json:"memory,omitempty"`
}

// SecretRef references a secret from a secret manager.
type SecretRef struct {
	// Name is the environment variable name
	Name string `yaml:"name" json:"name"`

	// Source is the secret source (e.g., "ssm:/path/to/secret", "secretsmanager:name")
	Source string `yaml:"source" json:"source"`
}

// Defaults applies default values to the configuration.
func (c *DeployConfig) Defaults() {
	if c.Region == "" {
		c.Region = "us-east-1"
	}
	if c.Service.Replicas == 0 {
		c.Service.Replicas = 1
	}
	if c.Resources.Size == "" {
		c.Resources.Size = "micro"
	}
	if len(c.Container.Ports) > 0 {
		for i := range c.Container.Ports {
			if c.Container.Ports[i].Protocol == "" {
				c.Container.Ports[i].Protocol = "TCP"
			}
		}
	}
	if c.Container.HealthCheck != nil {
		if c.Container.HealthCheck.Interval == 0 {
			c.Container.HealthCheck.Interval = 30 * time.Second
		}
		if c.Container.HealthCheck.Timeout == 0 {
			c.Container.HealthCheck.Timeout = 5 * time.Second
		}
		if c.Container.HealthCheck.HealthyThreshold == 0 {
			c.Container.HealthCheck.HealthyThreshold = 2
		}
		if c.Container.HealthCheck.UnhealthyThreshold == 0 {
			c.Container.HealthCheck.UnhealthyThreshold = 3
		}
	}
}
