package config

import "testing"

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
		})
	}
}
