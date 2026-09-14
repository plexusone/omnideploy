package pulumi

import (
	"context"
	"net/http"
	"net/http/httptest"
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
