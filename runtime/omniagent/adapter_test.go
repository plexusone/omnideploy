package omniagent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAdapter_Load_ExpandsEnvVars(t *testing.T) {
	t.Setenv("ADAPTER_TEST_TOKEN", "super-secret-token")

	content := `
gateway:
  address: "0.0.0.0:8080"

agent:
  provider: anthropic
  model: claude-sonnet-5

deploy:
  name: test-app
  image: ghcr.io/example/app:latest
  replicas: 1
  environment:
    OMNIAGENT_GATEWAY_ADDRESS: "0.0.0.0:8080"
    DISCORD_BOT_TOKEN: ${ADAPTER_TEST_TOKEN}
    LOG_LEVEL: ${ADAPTER_TEST_LOG_LEVEL:-info}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "omniagent.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	a := &Adapter{}
	cfg, err := a.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got := cfg.Environment["DISCORD_BOT_TOKEN"]; got != "super-secret-token" {
		t.Errorf("Environment[DISCORD_BOT_TOKEN] = %q, want the expanded secret value (not the literal ${...} placeholder)", got)
	}
	if got := cfg.Environment["LOG_LEVEL"]; got != "info" {
		t.Errorf("Environment[LOG_LEVEL] = %q, want the default \"info\"", got)
	}
	if got := cfg.Environment["OMNIAGENT_AGENT_PROVIDER"]; got != "anthropic" {
		t.Errorf("Environment[OMNIAGENT_AGENT_PROVIDER] = %q, want anthropic", got)
	}
	if len(cfg.Container.Ports) != 1 || cfg.Container.Ports[0].ContainerPort != 8080 {
		t.Errorf("Container.Ports = %+v, want a single port 8080 derived from gateway.address", cfg.Container.Ports)
	}
}

func TestAdapter_Detect(t *testing.T) {
	dir := t.TempDir()

	byName := filepath.Join(dir, "omniagent.yaml")
	if err := os.WriteFile(byName, []byte("name: x\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	a := &Adapter{}
	if !a.Detect(byName) {
		t.Error("Detect() = false for a filename containing \"omniagent\", want true")
	}

	byContent := filepath.Join(dir, "deploy.yaml")
	if err := os.WriteFile(byContent, []byte("gateway:\n  address: \"0.0.0.0:8080\"\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !a.Detect(byContent) {
		t.Error("Detect() = false for a file with a top-level gateway: key, want true")
	}

	notOmniagent := filepath.Join(dir, "other.yaml")
	if err := os.WriteFile(notOmniagent, []byte("name: x\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if a.Detect(notOmniagent) {
		t.Error("Detect() = true for an unrelated config, want false")
	}
}
