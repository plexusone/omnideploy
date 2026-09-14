package pulumi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/plexusone/omnideploy/config"
)

func TestVerifyHealth_NoHealthCheckConfigured(t *testing.T) {
	cfg := &config.DeployConfig{}
	if err := verifyHealth(context.Background(), cfg, "https://example.test", nil); err != nil {
		t.Errorf("verifyHealth() = %v, want nil (no health check configured)", err)
	}
}

func TestVerifyHealth_NoServiceURL(t *testing.T) {
	cfg := &config.DeployConfig{Container: config.ContainerConfig{
		HealthCheck: &config.HealthCheck{Path: "/health"},
	}}
	if err := verifyHealth(context.Background(), cfg, "", nil); err != nil {
		t.Errorf("verifyHealth() = %v, want nil (no service URL to verify against)", err)
	}
}

func TestVerifyHealth_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/health" {
			t.Errorf("probed path = %s, want /api/health", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &config.DeployConfig{Container: config.ContainerConfig{
		HealthCheck: &config.HealthCheck{Path: "/api/health"},
	}}
	// Trailing slash on the service URL must not produce a double slash
	// before the health path.
	var messages []string
	err := verifyHealth(context.Background(), cfg, srv.URL+"/", func(s string) { messages = append(messages, s) })
	if err != nil {
		t.Fatalf("verifyHealth() = %v, want nil", err)
	}
	if len(messages) == 0 || !strings.Contains(messages[len(messages)-1], "health verified") {
		t.Errorf("messages = %v, want a final \"health verified\" message", messages)
	}
}

func TestVerifyHealthRetry_SucceedsAfterTransientFailures(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := verifyHealthRetry(context.Background(), srv.URL+"/health", 5, 10*time.Millisecond, time.Second, nil)
	if err != nil {
		t.Fatalf("verifyHealthRetry() = %v, want nil after eventual success", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want exactly 3 (stop retrying once healthy)", attempts)
	}
}

func TestVerifyHealthRetry_ExhaustsAttempts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	err := verifyHealthRetry(context.Background(), srv.URL+"/health", 3, 10*time.Millisecond, time.Second, nil)
	if err == nil {
		t.Fatal("verifyHealthRetry() = nil, want error after exhausting attempts")
	}
	if !strings.Contains(err.Error(), "3 attempts") {
		t.Errorf("error = %v, want it to mention the attempt count", err)
	}
}

func TestVerifyHealthRetry_ContextCanceledDuringBackoff(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := verifyHealthRetry(ctx, srv.URL+"/health", 10, 500*time.Millisecond, time.Second, nil)
	if err == nil {
		t.Fatal("verifyHealthRetry() = nil, want error when context is canceled mid-backoff")
	}
}

func TestAvailabilityZone(t *testing.T) {
	if got := availabilityZone("us-west-2"); got != "us-west-2a" {
		t.Errorf("availabilityZone(us-west-2) = %q, want us-west-2a", got)
	}
}

func TestBinaryContentHash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app")
	if err := os.WriteFile(path, []byte("binary contents v1"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	h1, err := binaryContentHash(path)
	if err != nil {
		t.Fatalf("binaryContentHash() = %v", err)
	}
	if h1 == "" {
		t.Error("binaryContentHash() returned empty hash")
	}

	// Same content hashes identically — this is what lets remote.Command's
	// Triggers skip a re-run on a no-op redeploy.
	h2, err := binaryContentHash(path)
	if err != nil {
		t.Fatalf("binaryContentHash() = %v", err)
	}
	if h1 != h2 {
		t.Errorf("hash changed across calls with unchanged content: %q vs %q", h1, h2)
	}

	// Different content hashes differently — this is what forces a re-run.
	if err := os.WriteFile(path, []byte("binary contents v2"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	h3, err := binaryContentHash(path)
	if err != nil {
		t.Fatalf("binaryContentHash() = %v", err)
	}
	if h3 == h1 {
		t.Error("hash unchanged after content changed")
	}
}

func TestBinaryContentHash_MissingFile(t *testing.T) {
	if _, err := binaryContentHash(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Error("binaryContentHash() = nil error, want error for a missing file")
	}
}

func testInstanceConfig() *config.InstanceConfig {
	return &config.InstanceConfig{
		Blueprint:   "ubuntu_22_04",
		Bundle:      "nano_3_0",
		BinaryPath:  "./bin/app",
		RemotePath:  "/opt/app",
		ServiceName: "app",
	}
}

func TestDefaultSystemdUnit(t *testing.T) {
	unit := defaultSystemdUnit(testInstanceConfig())

	for _, want := range []string{
		"ExecStart=/opt/app",
		"EnvironmentFile=-/opt/app.env",
		"ReadWritePaths=/opt",
		"NoNewPrivileges=true",
		"ProtectSystem=strict",
		"ProtectHome=true",
		"WantedBy=multi-user.target",
	} {
		if !strings.Contains(unit, want) {
			t.Errorf("defaultSystemdUnit() missing %q, got:\n%s", want, unit)
		}
	}
}

func TestInstallServiceScript(t *testing.T) {
	inst := testInstanceConfig()
	script := installServiceScript(inst, defaultSystemdUnit(inst), buildRemoteEnvFile(map[string]string{"LOG_LEVEL": "info"}, nil))

	for _, want := range []string{
		"chmod +x /opt/app",
		"/etc/systemd/system/app.service",
		"/opt/app.env",
		`LOG_LEVEL="info"`,
		"chmod 600 /opt/app.env",
		"systemctl daemon-reload",
		"systemctl enable app",
		"systemctl restart app",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("installServiceScript() missing %q, got:\n%s", want, script)
		}
	}
}

func TestRestartServiceScript_DoesNotReinstallUnit(t *testing.T) {
	script := restartServiceScript(testInstanceConfig(), buildRemoteEnvFile(map[string]string{"LOG_LEVEL": "info"}, nil))

	if strings.Contains(script, "/etc/systemd/system/") {
		t.Errorf("restartServiceScript() should not touch the unit file, got:\n%s", script)
	}
	for _, want := range []string{"chmod +x /opt/app", "/opt/app.env", `LOG_LEVEL="info"`, "systemctl restart app"} {
		if !strings.Contains(script, want) {
			t.Errorf("restartServiceScript() missing %q, got:\n%s", want, script)
		}
	}
}

func TestBuildRemoteEnvFile(t *testing.T) {
	content := buildRemoteEnvFile(
		map[string]string{"LOG_LEVEL": "info", "API_KEY": "plain-value"},
		map[string]string{"API_KEY": "secret-value", "TOKEN": `has "quotes" and \backslash`},
	)

	// Secrets win over plain on a key collision.
	if !strings.Contains(content, `API_KEY="secret-value"`) {
		t.Errorf("content = %q, want API_KEY to be the resolved secret value, not the plain one", content)
	}
	if !strings.Contains(content, `LOG_LEVEL="info"`) {
		t.Errorf("content = %q, want LOG_LEVEL from the plain environment", content)
	}
	if !strings.Contains(content, `TOKEN="has \"quotes\" and \\backslash"`) {
		t.Errorf("content = %q, want TOKEN's quotes/backslashes escaped", content)
	}

	// Deterministic ordering: two calls with the same input produce
	// byte-identical output, so Triggers doesn't flap between deploys.
	again := buildRemoteEnvFile(
		map[string]string{"LOG_LEVEL": "info", "API_KEY": "plain-value"},
		map[string]string{"API_KEY": "secret-value", "TOKEN": `has "quotes" and \backslash`},
	)
	if content != again {
		t.Errorf("buildRemoteEnvFile() not deterministic:\n%q\nvs\n%q", content, again)
	}
}

func TestBuildRemoteEnvFile_Empty(t *testing.T) {
	if got := buildRemoteEnvFile(nil, nil); got != "" {
		t.Errorf("buildRemoteEnvFile(nil, nil) = %q, want empty string", got)
	}
}

func TestHashString(t *testing.T) {
	if hashString("a") == hashString("b") {
		t.Error("hashString(a) == hashString(b), want different hashes for different content")
	}
	if hashString("a") != hashString("a") {
		t.Error("hashString(a) is not stable across calls")
	}
}
