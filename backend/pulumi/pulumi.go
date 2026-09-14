// Package pulumi implements the Pulumi IaC backend.
package pulumi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/grokify/mogo/net/http/healthz"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lightsail"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optrefresh"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/plexusone/omnideploy/backend"
	"github.com/plexusone/omnideploy/config"
	"github.com/plexusone/omnideploy/secrets"
	"github.com/plexusone/omnideploy/target"
)

func init() {
	backend.Register(New())
}

// Backend implements the Pulumi IaC backend.
type Backend struct {
	// WorkDir is the working directory for Pulumi state
	WorkDir string
}

// New creates a new Pulumi backend.
func New() *Backend {
	workDir := os.Getenv("OMNIDEPLOY_WORK_DIR")
	if workDir == "" {
		home, _ := os.UserHomeDir()
		workDir = filepath.Join(home, ".omnideploy", "pulumi")
	}
	return &Backend{WorkDir: workDir}
}

// Name returns the backend name.
func (b *Backend) Name() string {
	return "pulumi"
}

// Description returns a human-readable description.
func (b *Backend) Description() string {
	return "Pulumi - Infrastructure as Code using Go"
}

// Apply provisions resources using Pulumi.
func (b *Backend) Apply(ctx context.Context, spec *target.ResourceSpec, opts backend.ApplyOptions) (*backend.Result, error) {
	stackName := opts.StackName
	if stackName == "" {
		stackName = spec.StackName
	}

	// Resolve secret refs up front so a missing secret fails the deploy
	// before any infrastructure changes (RMI-OMNIAGENT-006).
	resolved, err := resolveSecrets(ctx, spec)
	if err != nil {
		return nil, err
	}

	// Create the Pulumi program
	program := b.createProgram(spec, resolved)

	// Create or select the stack
	stack, err := b.getOrCreateStack(ctx, stackName, spec.Config.Name, program)
	if err != nil {
		return nil, fmt.Errorf("creating stack: %w", err)
	}

	// Set AWS region
	if err := stack.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: spec.Region}); err != nil {
		return nil, fmt.Errorf("setting region: %w", err)
	}

	// Run update
	var upOpts []optup.Option
	if opts.OnOutput != nil {
		upOpts = append(upOpts, optup.ProgressStreams(writerFunc(opts.OnOutput)))
	}

	result, err := stack.Up(ctx, upOpts...)
	if err != nil {
		return nil, fmt.Errorf("deploying: %w", err)
	}

	// Extract outputs
	outputs := make(map[string]string)
	for k, v := range result.Outputs {
		if s, ok := v.Value.(string); ok {
			outputs[k] = s
		}
	}

	// Post-deploy health verification (RMI-OMNIDEPLOY-001): the deployment
	// reaching ACTIVE proves the target's own internal health gate passed
	// (e.g. Lightsail already waited on its container health check before
	// returning), but not that the service is reachable from here — DNS
	// propagation, TLS cert readiness, or a network path issue can still
	// leave a "successfully deployed" service unreachable. Skip silently
	// when the config declares no health check or the target exposed no
	// URL output — there's nothing to verify against.
	if err := verifyHealth(ctx, spec.Config, outputs["url"], opts.OnOutput); err != nil {
		return nil, fmt.Errorf("deployment succeeded but health verification failed: %w", err)
	}

	// Extract resource changes
	var created, updated, deleted int
	if result.Summary.ResourceChanges != nil {
		changes := *result.Summary.ResourceChanges
		created = changes["create"]
		updated = changes["update"]
		deleted = changes["delete"]
	}

	return &backend.Result{
		StackName:        stackName,
		Outputs:          outputs,
		ResourcesCreated: created,
		ResourcesUpdated: updated,
		ResourcesDeleted: deleted,
	}, nil
}

// Preview shows what would be provisioned.
func (b *Backend) Preview(ctx context.Context, spec *target.ResourceSpec) (*backend.PreviewResult, error) {
	resolved, err := resolveSecrets(ctx, spec)
	if err != nil {
		return nil, err
	}
	program := b.createProgram(spec, resolved)

	stack, err := b.getOrCreateStack(ctx, spec.StackName, spec.Config.Name, program)
	if err != nil {
		return nil, fmt.Errorf("creating stack: %w", err)
	}

	if err := stack.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: spec.Region}); err != nil {
		return nil, fmt.Errorf("setting region: %w", err)
	}

	result, err := stack.Preview(ctx, optpreview.ProgressStreams(os.Stdout))
	if err != nil {
		return nil, fmt.Errorf("previewing: %w", err)
	}

	// Build summary from ChangeSummary
	var created, updated, deleted int
	if result.ChangeSummary != nil {
		created = result.ChangeSummary["create"]
		updated = result.ChangeSummary["update"]
		deleted = result.ChangeSummary["delete"]
	}

	return &backend.PreviewResult{
		StackName: spec.StackName,
		Changes:   []backend.Change{}, // Detailed steps not available in this API version
		Summary:   fmt.Sprintf("Changes: %d create, %d update, %d delete", created, updated, deleted),
	}, nil
}

// Destroy removes all resources.
func (b *Backend) Destroy(ctx context.Context, stackName string, opts backend.DestroyOptions) error {
	stack, err := auto.SelectStackLocalSource(ctx, stackName, b.WorkDir)
	if err != nil {
		return fmt.Errorf("selecting stack: %w", err)
	}

	var destroyOpts []optdestroy.Option
	if opts.OnOutput != nil {
		destroyOpts = append(destroyOpts, optdestroy.ProgressStreams(writerFunc(opts.OnOutput)))
	}

	_, err = stack.Destroy(ctx, destroyOpts...)
	if err != nil {
		return fmt.Errorf("destroying: %w", err)
	}

	return nil
}

// Refresh refreshes the state.
func (b *Backend) Refresh(ctx context.Context, stackName string) error {
	stack, err := auto.SelectStackLocalSource(ctx, stackName, b.WorkDir)
	if err != nil {
		return fmt.Errorf("selecting stack: %w", err)
	}

	_, err = stack.Refresh(ctx, optrefresh.ProgressStreams(os.Stdout))
	if err != nil {
		return fmt.Errorf("refreshing: %w", err)
	}

	return nil
}

// getOrCreateStack creates or selects a Pulumi stack.
func (b *Backend) getOrCreateStack(ctx context.Context, stackName, projectName string, program pulumi.RunFunc) (auto.Stack, error) {
	// Ensure work directory exists
	if err := os.MkdirAll(b.WorkDir, 0o755); err != nil {
		return auto.Stack{}, fmt.Errorf("creating work dir: %w", err)
	}

	// Try to create a new stack, or select existing
	stack, err := auto.UpsertStackInlineSource(ctx, stackName, projectName, program,
		auto.WorkDir(b.WorkDir),
	)
	if err != nil {
		return auto.Stack{}, err
	}

	return stack, nil
}

// createProgram creates the Pulumi program for the given spec.
func (b *Backend) createProgram(spec *target.ResourceSpec, resolvedSecrets map[string]string) pulumi.RunFunc {
	return func(ctx *pulumi.Context) error {
		switch spec.Target {
		case "lightsail":
			return b.deployLightsail(ctx, spec, resolvedSecrets)
		case "lightsail-instance":
			return b.deployLightsailInstance(ctx, spec, resolvedSecrets)
		default:
			return fmt.Errorf("unsupported target: %s", spec.Target)
		}
	}
}

// deployLightsail deploys to AWS LightSail.
func (b *Backend) deployLightsail(ctx *pulumi.Context, spec *target.ResourceSpec, resolvedSecrets map[string]string) error {
	cfg := spec.Config

	// Create container service
	service, err := lightsail.NewContainerService(ctx, cfg.Name, &lightsail.ContainerServiceArgs{
		Name:       pulumi.String(cfg.Name),
		Power:      pulumi.String(sizeToPower(cfg.Resources.Size)),
		Scale:      pulumi.Int(cfg.Service.Replicas),
		IsDisabled: pulumi.Bool(false),
		Tags:       pulumi.ToStringMap(cfg.Tags),
	})
	if err != nil {
		return fmt.Errorf("creating container service: %w", err)
	}

	// Build container definition
	containers := lightsail.ContainerServiceDeploymentVersionContainerArray{
		&lightsail.ContainerServiceDeploymentVersionContainerArgs{
			ContainerName: pulumi.String(cfg.Name),
			Image:         pulumi.String(cfg.Container.Image),
			Commands:      pulumi.ToStringArray(cfg.Container.Args),
			Environment:   buildEnvironment(cfg.Environment, resolvedSecrets),
			Ports:         buildPortMap(cfg.Container.Ports),
		},
	}

	// Build public endpoint
	var publicEndpoint *lightsail.ContainerServiceDeploymentVersionPublicEndpointArgs
	if cfg.Service.Public && len(cfg.Container.Ports) > 0 {
		endpointArgs := &lightsail.ContainerServiceDeploymentVersionPublicEndpointArgs{
			ContainerName: pulumi.String(cfg.Name),
			ContainerPort: pulumi.Int(cfg.Container.Ports[0].ContainerPort),
		}

		if cfg.Container.HealthCheck != nil && cfg.Container.HealthCheck.Path != "" {
			endpointArgs.HealthCheck = &lightsail.ContainerServiceDeploymentVersionPublicEndpointHealthCheckArgs{
				Path:               pulumi.String(cfg.Container.HealthCheck.Path),
				IntervalSeconds:    pulumi.Int(int(cfg.Container.HealthCheck.Interval.Seconds())),
				TimeoutSeconds:     pulumi.Int(int(cfg.Container.HealthCheck.Timeout.Seconds())),
				HealthyThreshold:   pulumi.Int(cfg.Container.HealthCheck.HealthyThreshold),
				UnhealthyThreshold: pulumi.Int(cfg.Container.HealthCheck.UnhealthyThreshold),
			}
		}

		publicEndpoint = endpointArgs
	}

	// Create deployment
	deployment, err := lightsail.NewContainerServiceDeploymentVersion(ctx, cfg.Name+"-deployment", &lightsail.ContainerServiceDeploymentVersionArgs{
		ServiceName:    service.Name,
		Containers:     containers,
		PublicEndpoint: publicEndpoint,
	})
	if err != nil {
		return fmt.Errorf("creating deployment: %w", err)
	}

	// Export outputs
	ctx.Export("url", service.Url)
	ctx.Export("state", deployment.State)
	ctx.Export("service_name", service.Name)

	return nil
}

// instanceSSHUser is the SSH user for Lightsail instances created from
// Ubuntu blueprints — the blueprint grokify-omniagent's reference
// implementation uses, and the common case for this target.
const instanceSSHUser = "ubuntu"

// deployLightsailInstance deploys to an AWS Lightsail VM instance running
// a pre-built binary as a systemd service, reproducing
// grokify-omniagent's reference deploy/lightsail/{setup,deploy}.sh
// declaratively (docs/specs/TRD.md). Unlike deployLightsail's Container
// Service target, the instance's disk is persistent across redeploys at
// no extra cost. remote.Command's Create/Update split (rather than
// lightsail.Instance's UserData, which is cloud-init and only runs once
// at first boot) is what makes redeploys idempotent: Create installs the
// systemd unit and starts the service; Update only restarts it; Triggers
// is keyed on the binary's content hash plus the unit content so either
// changing forces a re-run.
func (b *Backend) deployLightsailInstance(ctx *pulumi.Context, spec *target.ResourceSpec, resolvedSecrets map[string]string) error { //nolint:unparam // resolvedSecrets is wired for the .env injection landing in RMI-OMNIDEPLOY-006 (Phase 2); the createProgram call site requires the same signature as deployLightsail
	cfg := spec.Config
	inst := cfg.Instance

	keyPair, err := lightsail.NewKeyPair(ctx, cfg.Name+"-keypair", &lightsail.KeyPairArgs{
		Name: pulumi.String(cfg.Name),
	})
	if err != nil {
		return fmt.Errorf("creating key pair: %w", err)
	}

	instance, err := lightsail.NewInstance(ctx, cfg.Name, &lightsail.InstanceArgs{
		Name:             pulumi.String(cfg.Name),
		AvailabilityZone: pulumi.String(availabilityZone(spec.Region)),
		BlueprintId:      pulumi.String(inst.Blueprint),
		BundleId:         pulumi.String(inst.Bundle),
		KeyPairName:      keyPair.Name,
		Tags:             pulumi.ToStringMap(cfg.Tags),
	})
	if err != nil {
		return fmt.Errorf("creating instance: %w", err)
	}

	portInfos := lightsail.InstancePublicPortsPortInfoArray{
		&lightsail.InstancePublicPortsPortInfoArgs{
			Protocol: pulumi.String("tcp"),
			FromPort: pulumi.Int(22),
			ToPort:   pulumi.Int(22),
		},
	}
	if inst.HealthCheck != nil && inst.HealthCheck.Kind == "http" {
		portInfos = append(portInfos, &lightsail.InstancePublicPortsPortInfoArgs{
			Protocol: pulumi.String("tcp"),
			FromPort: pulumi.Int(inst.HealthCheck.Port),
			ToPort:   pulumi.Int(inst.HealthCheck.Port),
		})
	}
	firewall, err := lightsail.NewInstancePublicPorts(ctx, cfg.Name+"-firewall", &lightsail.InstancePublicPortsArgs{
		InstanceName: instance.Name,
		PortInfos:    portInfos,
	})
	if err != nil {
		return fmt.Errorf("configuring firewall: %w", err)
	}

	conn := remote.ConnectionArgs{
		Host:       instance.PublicIpAddress,
		User:       pulumi.String(instanceSSHUser),
		PrivateKey: keyPair.PrivateKey,
	}

	binaryHash, err := binaryContentHash(inst.BinaryPath)
	if err != nil {
		return fmt.Errorf("hashing binary at %s: %w", inst.BinaryPath, err)
	}

	// The firewall must exist before SSH can reach the instance.
	artifact, err := remote.NewCopyToRemote(ctx, cfg.Name+"-artifact", &remote.CopyToRemoteArgs{
		Connection: conn,
		RemotePath: pulumi.String(inst.RemotePath),
		Source:     pulumi.NewFileAsset(inst.BinaryPath),
		Triggers:   pulumi.Array{pulumi.String(binaryHash)},
	}, pulumi.DependsOn([]pulumi.Resource{firewall}))
	if err != nil {
		return fmt.Errorf("copying artifact: %w", err)
	}

	unit := inst.SystemdUnit
	if unit == "" {
		unit = defaultSystemdUnit(inst)
	}

	if _, err := remote.NewCommand(ctx, cfg.Name+"-service", &remote.CommandArgs{
		Connection: conn,
		Create:     pulumi.String(installServiceScript(inst, unit)),
		Update:     pulumi.String(restartServiceScript(inst)),
		Triggers:   pulumi.Array{pulumi.String(binaryHash), pulumi.String(unit)},
	}, pulumi.DependsOn([]pulumi.Resource{artifact})); err != nil {
		return fmt.Errorf("installing service: %w", err)
	}

	ctx.Export("public_ip", instance.PublicIpAddress)
	ctx.Export("service_name", pulumi.String(inst.ServiceName))

	return nil
}

// availabilityZone derives a default AZ from the deploy region — Lightsail
// requires one, and a single, predictable AZ per region is the right
// default for a single-instance, capped-cost deployment (no multi-AZ
// redundancy to configure).
func availabilityZone(region string) string {
	return region + "a"
}

// binaryContentHash hashes the local binary so remote.Command's Triggers
// can detect "nothing changed" and skip re-running Create/Update — the
// idempotent-redeploy behavior documented on deployLightsailInstance.
func binaryContentHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// defaultSystemdUnit renders a systemd unit for inst when the config
// doesn't supply its own (config.InstanceConfig.SystemdUnit), reproducing
// the reference setup.sh's hardening: no new privileges, a read-only
// system/home with one writable path (the binary's own directory, for
// e.g. a local SQLite file dropped next to it). EnvironmentFile uses the
// "-" prefix so a missing .env doesn't stop the service — Phase 2
// (RMI-OMNIDEPLOY-006) starts writing that file; this unit already
// expects it at that path.
func defaultSystemdUnit(inst *config.InstanceConfig) string {
	return fmt.Sprintf(`[Unit]
Description=%s
After=network.target

[Service]
Type=simple
ExecStart=%s
Restart=on-failure
RestartSec=5
EnvironmentFile=-%s.env
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=%s

[Install]
WantedBy=multi-user.target
`, inst.ServiceName, inst.RemotePath, inst.RemotePath, path.Dir(inst.RemotePath))
}

// installServiceScript is remote.Command's Create step: write the
// systemd unit, make the binary executable, and enable+start the
// service. Runs once — redeploys use restartServiceScript instead.
func installServiceScript(inst *config.InstanceConfig, unit string) string {
	return fmt.Sprintf(`set -eu
sudo mkdir -p %s
sudo chmod +x %s
sudo tee /etc/systemd/system/%s.service > /dev/null <<'OMNIDEPLOY_UNIT'
%s
OMNIDEPLOY_UNIT
sudo systemctl daemon-reload
sudo systemctl enable %s
sudo systemctl restart %s
`, path.Dir(inst.RemotePath), inst.RemotePath, inst.ServiceName, unit, inst.ServiceName, inst.ServiceName)
}

// restartServiceScript is remote.Command's Update step: the binary was
// already copied by remote.CopyToRemote by the time this runs, so this
// only needs to make it executable again and restart the service — no
// systemd unit reinstall, keeping redeploys fast and idempotent.
func restartServiceScript(inst *config.InstanceConfig) string {
	return fmt.Sprintf(`set -eu
sudo chmod +x %s
sudo systemctl daemon-reload
sudo systemctl restart %s
`, inst.RemotePath, inst.ServiceName)
}

// sizeToPower maps size to LightSail power.
func sizeToPower(size string) string {
	switch size {
	case "nano":
		return "nano"
	case "micro", "":
		return "micro"
	case "small":
		return "small"
	case "medium":
		return "medium"
	case "large":
		return "large"
	case "xlarge":
		return "xlarge"
	default:
		return "micro"
	}
}

// buildPortMap converts ports to Pulumi string map (port -> protocol).
func buildPortMap(ports []config.PortMapping) pulumi.StringMap {
	result := pulumi.StringMap{}
	for _, p := range ports {
		key := fmt.Sprintf("%d", p.ContainerPort)
		result[key] = pulumi.String(p.Protocol)
	}
	return result
}

// writerFunc adapts a callback to io.Writer.
type writerFunc func(string)

func (f writerFunc) Write(p []byte) (n int, err error) {
	f(string(p))
	return len(p), nil
}

// resolveSecrets resolves spec.Config.Secrets to values (RMI-OMNIAGENT-006).
// Returns nil when the config declares no secrets so no AWS client is
// constructed for secret-free deploys.
func resolveSecrets(ctx context.Context, spec *target.ResourceSpec) (map[string]string, error) {
	if len(spec.Config.Secrets) == 0 {
		return nil, nil
	}
	resolver, err := secrets.NewResolver(ctx, spec.Region)
	if err != nil {
		return nil, fmt.Errorf("building secret resolver: %w", err)
	}
	resolved, err := resolver.Resolve(ctx, spec.Config.Secrets)
	if err != nil {
		return nil, fmt.Errorf("resolving secrets: %w", err)
	}
	return resolved, nil
}

// buildEnvironment merges plain environment variables with resolved
// secrets into one container environment map. Secret values are wrapped
// with pulumi.ToSecret so they are encrypted in Pulumi state rather than
// stored plaintext; on a name collision the secret wins over the plain
// entry, so promoting a variable from environment to secrets needs no
// removal of the old key.
func buildEnvironment(plain map[string]string, resolvedSecrets map[string]string) pulumi.StringMap {
	env := make(pulumi.StringMap, len(plain)+len(resolvedSecrets))
	for k, v := range plain {
		env[k] = pulumi.String(v)
	}
	for k, v := range resolvedSecrets {
		env[k] = pulumi.ToSecret(pulumi.String(v)).(pulumi.StringOutput)
	}
	return env
}

// healthVerifyAttempts and healthVerifyDelay bound post-deploy health
// verification: a deployment that just reached ACTIVE may still need a
// few seconds for DNS/TLS to settle from this machine's vantage point.
const (
	healthVerifyAttempts = 5
	healthVerifyDelay    = 3 * time.Second
	healthVerifyTimeout  = 5 * time.Second
)

// verifyHealth probes cfg's configured health-check path against
// serviceURL, retrying briefly to absorb DNS/TLS settling immediately
// after a deployment reaches ACTIVE. No-ops (returns nil) when the config
// declares no health check or serviceURL is empty — there's nothing to
// verify against for that target/config combination.
func verifyHealth(ctx context.Context, cfg *config.DeployConfig, serviceURL string, onOutput func(string)) error {
	if cfg.Container.HealthCheck == nil || cfg.Container.HealthCheck.Path == "" || serviceURL == "" {
		return nil
	}
	url := strings.TrimRight(serviceURL, "/") + cfg.Container.HealthCheck.Path
	return verifyHealthRetry(ctx, url, healthVerifyAttempts, healthVerifyDelay, healthVerifyTimeout, onOutput)
}

// verifyHealthRetry is verifyHealth's retry loop, parameterized so tests
// can use short delays instead of the real (deployment-appropriate)
// defaults.
func verifyHealthRetry(ctx context.Context, url string, attempts int, delay, timeout time.Duration, onOutput func(string)) error {
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if onOutput != nil {
			onOutput(fmt.Sprintf("verifying health: %s (attempt %d/%d)", url, attempt, attempts))
		}
		if err := healthz.Probe(ctx, url, timeout); err != nil {
			lastErr = err
			if attempt < attempts {
				select {
				case <-time.After(delay):
				case <-ctx.Done():
					return fmt.Errorf("%s: %w", url, ctx.Err())
				}
			}
			continue
		}
		if onOutput != nil {
			onOutput(fmt.Sprintf("health verified: %s", url))
		}
		return nil
	}
	return fmt.Errorf("%s did not become healthy after %d attempts: %w", url, attempts, lastErr)
}
