package config

import (
	"strings"
	"testing"
)

func TestParse_ExpandsEnvVars(t *testing.T) {
	t.Setenv("LOADER_TEST_IMAGE", "ghcr.io/example/app:v1")

	data := []byte(`
name: my-app
container:
  image: ${LOADER_TEST_IMAGE}
  ports:
    - container_port: 8080
environment:
  LOG_LEVEL: ${LOADER_TEST_LOG_LEVEL:-info}
`)

	cfg, err := Parse(data, "yaml")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.Container.Image != "ghcr.io/example/app:v1" {
		t.Errorf("Container.Image = %q, want the expanded value", cfg.Container.Image)
	}
	if cfg.Environment["LOG_LEVEL"] != "info" {
		t.Errorf("Environment[LOG_LEVEL] = %q, want the default \"info\"", cfg.Environment["LOG_LEVEL"])
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     DeployConfig
		wantErr string
	}{
		{
			name: "valid",
			cfg: DeployConfig{
				Name:      "app",
				Container: ContainerConfig{Image: "img", Ports: []PortMapping{{ContainerPort: 80}}},
			},
		},
		{name: "missing name", cfg: DeployConfig{Container: ContainerConfig{Image: "img", Ports: []PortMapping{{ContainerPort: 80}}}}, wantErr: "name is required"},
		{name: "missing image", cfg: DeployConfig{Name: "app", Container: ContainerConfig{Ports: []PortMapping{{ContainerPort: 80}}}}, wantErr: "container.image is required"},
		{name: "missing ports", cfg: DeployConfig{Name: "app", Container: ContainerConfig{Image: "img"}}, wantErr: "at least one container port"},
		{name: "bad port", cfg: DeployConfig{Name: "app", Container: ContainerConfig{Image: "img", Ports: []PortMapping{{ContainerPort: 0}}}}, wantErr: "container_port"},
		{name: "negative replicas", cfg: DeployConfig{Name: "app", Container: ContainerConfig{Image: "img", Ports: []PortMapping{{ContainerPort: 80}}}, Service: ServiceConfig{Replicas: -1}}, wantErr: "replicas cannot be negative"},
		{
			name: "valid instance",
			cfg: DeployConfig{
				Name:     "app",
				Instance: &InstanceConfig{Blueprint: "ubuntu_22_04", Bundle: "nano_3_0", BinaryPath: "./bin/app", RemotePath: "/opt/app", ServiceName: "app"},
			},
		},
		{
			name: "valid instance with http health check",
			cfg: DeployConfig{
				Name: "app",
				Instance: &InstanceConfig{
					Blueprint: "ubuntu_22_04", Bundle: "nano_3_0", BinaryPath: "./bin/app", RemotePath: "/opt/app", ServiceName: "app",
					HealthCheck: &VMHealthCheck{Kind: "http", Path: "/health", Port: 8080},
				},
			},
		},
		{
			name: "valid instance with systemd health check",
			cfg: DeployConfig{
				Name:     "app",
				Instance: &InstanceConfig{Blueprint: "ubuntu_22_04", Bundle: "nano_3_0", BinaryPath: "./bin/app", RemotePath: "/opt/app", ServiceName: "app", HealthCheck: &VMHealthCheck{Kind: "systemd"}},
			},
		},
		{name: "instance missing blueprint", cfg: DeployConfig{Name: "app", Instance: &InstanceConfig{Bundle: "nano_3_0", BinaryPath: "./bin/app", RemotePath: "/opt/app", ServiceName: "app"}}, wantErr: "instance.blueprint is required"},
		{name: "instance missing bundle", cfg: DeployConfig{Name: "app", Instance: &InstanceConfig{Blueprint: "ubuntu_22_04", BinaryPath: "./bin/app", RemotePath: "/opt/app", ServiceName: "app"}}, wantErr: "instance.bundle is required"},
		{name: "instance missing binary_path", cfg: DeployConfig{Name: "app", Instance: &InstanceConfig{Blueprint: "ubuntu_22_04", Bundle: "nano_3_0", RemotePath: "/opt/app", ServiceName: "app"}}, wantErr: "instance.binary_path is required"},
		{name: "instance missing remote_path", cfg: DeployConfig{Name: "app", Instance: &InstanceConfig{Blueprint: "ubuntu_22_04", Bundle: "nano_3_0", BinaryPath: "./bin/app", ServiceName: "app"}}, wantErr: "instance.remote_path is required"},
		{name: "instance missing service_name", cfg: DeployConfig{Name: "app", Instance: &InstanceConfig{Blueprint: "ubuntu_22_04", Bundle: "nano_3_0", BinaryPath: "./bin/app", RemotePath: "/opt/app"}}, wantErr: "instance.service_name is required"},
		{
			name: "instance http health check missing path",
			cfg: DeployConfig{
				Name:     "app",
				Instance: &InstanceConfig{Blueprint: "ubuntu_22_04", Bundle: "nano_3_0", BinaryPath: "./bin/app", RemotePath: "/opt/app", ServiceName: "app", HealthCheck: &VMHealthCheck{Kind: "http", Port: 8080}},
			},
			wantErr: "instance.health_check.path is required",
		},
		{
			name: "instance http health check bad port",
			cfg: DeployConfig{
				Name:     "app",
				Instance: &InstanceConfig{Blueprint: "ubuntu_22_04", Bundle: "nano_3_0", BinaryPath: "./bin/app", RemotePath: "/opt/app", ServiceName: "app", HealthCheck: &VMHealthCheck{Kind: "http", Path: "/health", Port: 0}},
			},
			wantErr: "instance.health_check.port",
		},
		{
			name: "instance unknown health check kind",
			cfg: DeployConfig{
				Name:     "app",
				Instance: &InstanceConfig{Blueprint: "ubuntu_22_04", Bundle: "nano_3_0", BinaryPath: "./bin/app", RemotePath: "/opt/app", ServiceName: "app", HealthCheck: &VMHealthCheck{Kind: "carrier-pigeon"}},
			},
			wantErr: "instance.health_check.kind must be",
		},
		{
			name: "both container and instance set",
			cfg: DeployConfig{
				Name:      "app",
				Container: ContainerConfig{Image: "img", Ports: []PortMapping{{ContainerPort: 80}}},
				Instance:  &InstanceConfig{Blueprint: "ubuntu_22_04", Bundle: "nano_3_0", BinaryPath: "./bin/app", RemotePath: "/opt/app", ServiceName: "app"},
			},
			wantErr: "exactly one of container/instance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}
