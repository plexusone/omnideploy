package lightsailinstance

import (
	"testing"

	"github.com/plexusone/omnideploy/config"
	"github.com/plexusone/omnideploy/target"
)

func validConfig() *config.DeployConfig {
	return &config.DeployConfig{
		Name: "test-app",
		Instance: &config.InstanceConfig{
			Blueprint:   "ubuntu_22_04",
			Bundle:      "nano_3_0",
			BinaryPath:  "./bin/app",
			RemotePath:  "/opt/app",
			ServiceName: "app",
		},
	}
}

func TestName(t *testing.T) {
	if got := (&Target{}).Name(); got != "lightsail-instance" {
		t.Errorf("Name() = %q, want lightsail-instance", got)
	}
}

func TestValidate_Valid(t *testing.T) {
	if err := (&Target{}).Validate(validConfig()); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestValidate_NoInstance(t *testing.T) {
	cfg := &config.DeployConfig{
		Name:      "test-app",
		Container: config.ContainerConfig{Image: "img", Ports: []config.PortMapping{{ContainerPort: 80}}},
	}
	if err := (&Target{}).Validate(cfg); err == nil {
		t.Error("Validate() = nil, want error for a container-only config on the instance target")
	}
}

func TestValidate_DelegatesToConfigValidate(t *testing.T) {
	cfg := validConfig()
	cfg.Name = "" // config.Validate's own "name is required" check
	if err := (&Target{}).Validate(cfg); err == nil {
		t.Error("Validate() = nil, want error propagated from config.DeployConfig.Validate")
	}
}

func TestResourceSpec_Valid(t *testing.T) {
	cfg := validConfig()
	spec, err := (&Target{}).ResourceSpec(cfg)
	if err != nil {
		t.Fatalf("ResourceSpec() error = %v", err)
	}
	if spec.Target != "lightsail-instance" || spec.StackName != "test-app" {
		t.Errorf("spec = %+v, want Target=lightsail-instance StackName=test-app", spec)
	}
	if len(spec.Resources) == 0 {
		t.Error("spec.Resources is empty, want the instance/keypair/firewall/artifact/service resources")
	}
	var haveInstance, haveFirewall bool
	for _, r := range spec.Resources {
		switch r.Type {
		case "lightsail_instance":
			haveInstance = true
		case "lightsail_instance_public_ports":
			haveFirewall = true
		}
	}
	if !haveInstance || !haveFirewall {
		t.Errorf("resources = %+v, missing lightsail_instance or lightsail_instance_public_ports", spec.Resources)
	}
}

func TestResourceSpec_InvalidPropagates(t *testing.T) {
	cfg := &config.DeployConfig{Name: "test-app"} // neither Container nor Instance set
	if _, err := (&Target{}).ResourceSpec(cfg); err == nil {
		t.Error("ResourceSpec() = nil error, want validation failure to propagate")
	}
}

func TestResourceSpec_SSHOnlyFirewallByDefault(t *testing.T) {
	cfg := validConfig() // no HealthCheck set
	spec, err := (&Target{}).ResourceSpec(cfg)
	if err != nil {
		t.Fatalf("ResourceSpec() error = %v", err)
	}
	fw := findResource(t, spec.Resources, "lightsail_instance_public_ports")
	if fw.Properties["ssh_only"] != true {
		t.Errorf("firewall ssh_only = %v, want true when no HTTP health check is configured", fw.Properties["ssh_only"])
	}
	if fw.Properties["app_port"] != 0 {
		t.Errorf("firewall app_port = %v, want 0 when no HTTP health check is configured", fw.Properties["app_port"])
	}
}

func TestResourceSpec_OpensAppPortForHTTPHealthCheck(t *testing.T) {
	cfg := validConfig()
	cfg.Instance.HealthCheck = &config.VMHealthCheck{Kind: "http", Path: "/health", Port: 8080}
	spec, err := (&Target{}).ResourceSpec(cfg)
	if err != nil {
		t.Fatalf("ResourceSpec() error = %v", err)
	}
	fw := findResource(t, spec.Resources, "lightsail_instance_public_ports")
	if fw.Properties["ssh_only"] != false {
		t.Errorf("firewall ssh_only = %v, want false when an HTTP health check is configured", fw.Properties["ssh_only"])
	}
	if fw.Properties["app_port"] != 8080 {
		t.Errorf("firewall app_port = %v, want 8080", fw.Properties["app_port"])
	}
}

func TestResourceSpec_SystemdHealthCheckStaysSSHOnly(t *testing.T) {
	cfg := validConfig()
	cfg.Instance.HealthCheck = &config.VMHealthCheck{Kind: "systemd"}
	spec, err := (&Target{}).ResourceSpec(cfg)
	if err != nil {
		t.Fatalf("ResourceSpec() error = %v", err)
	}
	fw := findResource(t, spec.Resources, "lightsail_instance_public_ports")
	if fw.Properties["ssh_only"] != true {
		t.Errorf("firewall ssh_only = %v, want true for a systemd (non-HTTP) health check", fw.Properties["ssh_only"])
	}
}

func findResource(t *testing.T, resources []target.Resource, resourceType string) target.Resource {
	t.Helper()
	for _, r := range resources {
		if r.Type == resourceType {
			return r
		}
	}
	t.Fatalf("no resource of type %q found in %+v", resourceType, resources)
	return target.Resource{}
}
