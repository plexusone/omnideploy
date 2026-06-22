// Package main is the omnideploy CLI entry point.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/plexusone/omnideploy/backend"
	"github.com/plexusone/omnideploy/bootstrap"
	"github.com/plexusone/omnideploy/deploy"
	"github.com/plexusone/omnideploy/ecr"
	"github.com/plexusone/omnideploy/runtime"
	"github.com/plexusone/omnideploy/target"

	// Register targets
	_ "github.com/plexusone/omnideploy/target/lightsail"

	// Register backends
	_ "github.com/plexusone/omnideploy/backend/pulumi"

	// Register runtime adapters
	_ "github.com/plexusone/omnideploy/runtime/container"
	_ "github.com/plexusone/omnideploy/runtime/omniagent"
)

var (
	configFile  string
	targetName  string
	backendName string
	runtimeName string
	stackName   string
	autoApprove bool

	// Bootstrap flags
	bootstrapUser      string
	bootstrapCreateKey bool
	bootstrapRegion    string

	// ECR flags
	ecrRegion     string
	ecrForce      bool
	ecrDockerfile string
	ecrContext    string
	ecrTag        string
	ecrPlatform   string
	ecrImage      string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "omnideploy",
	Short: "Universal deployment tool for container applications",
	Long: `OmniDeploy - Deploy containers to any cloud provider using any IaC tool.

Supports multiple deployment targets (LightSail, ECS, Kubernetes) and
multiple IaC backends (Pulumi, CDK, Terraform).`,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Config file path")
	rootCmd.PersistentFlags().StringVarP(&targetName, "target", "t", "lightsail", "Deployment target (lightsail, ecs, kubernetes)")
	rootCmd.PersistentFlags().StringVarP(&backendName, "backend", "b", "pulumi", "IaC backend (pulumi, cdk, terraform)")
	rootCmd.PersistentFlags().StringVarP(&runtimeName, "runtime", "r", "", "Runtime adapter (omniagent, agentkit, container)")
	rootCmd.PersistentFlags().StringVarP(&stackName, "stack", "s", "", "Stack name (defaults to config name)")

	// Add commands
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(previewCmd)
	rootCmd.AddCommand(destroyCmd)
	rootCmd.AddCommand(targetsCmd)
	rootCmd.AddCommand(backendsCmd)
	rootCmd.AddCommand(runtimesCmd)
	rootCmd.AddCommand(bootstrapCmd)
	rootCmd.AddCommand(ecrCmd)
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Deploy or update a stack",
	Long:  "Deploy a new stack or update an existing one.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if configFile == "" {
			return fmt.Errorf("--config is required")
		}

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		deployer, err := createDeployer()
		if err != nil {
			return err
		}

		fmt.Printf("Deploying to %s using %s...\n", targetName, backendName)

		result, err := deployer.Apply(ctx, configFile, backend.ApplyOptions{
			StackName:   stackName,
			AutoApprove: autoApprove,
			OnOutput: func(msg string) {
				fmt.Print(msg)
			},
		})
		if err != nil {
			return fmt.Errorf("deployment failed: %w", err)
		}

		fmt.Println("\nDeployment complete!")
		fmt.Printf("Stack: %s\n", result.StackName)
		fmt.Printf("Resources: %d created, %d updated, %d deleted\n",
			result.ResourcesCreated, result.ResourcesUpdated, result.ResourcesDeleted)

		if len(result.Outputs) > 0 {
			fmt.Println("\nOutputs:")
			for k, v := range result.Outputs {
				fmt.Printf("  %s: %s\n", k, v)
			}
		}

		return nil
	},
}

var previewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Preview changes without deploying",
	Long:  "Show what would be created, updated, or deleted without making changes.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if configFile == "" {
			return fmt.Errorf("--config is required")
		}

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		deployer, err := createDeployer()
		if err != nil {
			return err
		}

		fmt.Printf("Previewing deployment to %s using %s...\n\n", targetName, backendName)

		result, err := deployer.Preview(ctx, configFile)
		if err != nil {
			return fmt.Errorf("preview failed: %w", err)
		}

		fmt.Println("Changes:")
		for _, change := range result.Changes {
			fmt.Printf("  %s %s: %s\n", change.Type, change.ResourceType, change.ResourceName)
		}
		fmt.Printf("\n%s\n", result.Summary)

		return nil
	},
}

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy a stack",
	Long:  "Remove all resources in a stack.",
	RunE: func(cmd *cobra.Command, args []string) error {
		name := stackName
		if name == "" && len(args) > 0 {
			name = args[0]
		}
		if name == "" {
			return fmt.Errorf("stack name required (--stack or as argument)")
		}

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		deployer, err := createDeployer()
		if err != nil {
			return err
		}

		if !autoApprove {
			fmt.Printf("This will destroy stack '%s'. Are you sure? [y/N] ", name)
			var confirm string
			if _, err := fmt.Scanln(&confirm); err != nil {
				slog.Warn("failed to read confirmation", "error", err)
			}
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		fmt.Printf("Destroying stack %s...\n", name)

		err = deployer.Destroy(ctx, name, backend.DestroyOptions{
			AutoApprove: true,
			OnOutput: func(msg string) {
				fmt.Print(msg)
			},
		})
		if err != nil {
			return fmt.Errorf("destroy failed: %w", err)
		}

		fmt.Println("Stack destroyed.")
		return nil
	},
}

var targetsCmd = &cobra.Command{
	Use:   "targets",
	Short: "List available deployment targets",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Available targets:")
		for name, t := range target.All() {
			fmt.Printf("  %-12s %s\n", name, t.Description())
		}
	},
}

var backendsCmd = &cobra.Command{
	Use:   "backends",
	Short: "List available IaC backends",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Available backends:")
		for name, b := range backend.All() {
			fmt.Printf("  %-12s %s\n", name, b.Description())
		}
	},
}

var runtimesCmd = &cobra.Command{
	Use:   "runtimes",
	Short: "List available runtime adapters",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Available runtime adapters:")
		for name, r := range runtime.All() {
			fmt.Printf("  %-12s %s\n", name, r.Description())
		}
	},
}

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Set up AWS IAM resources for OmniDeploy",
	Long: `Bootstrap creates the required AWS IAM resources for OmniDeploy:

  - IAM Policy (OmniDeployPolicy) with permissions for LightSail, ECR, and SSM
  - IAM Group (omnideploy-users) with the policy attached
  - Optionally, an IAM User with access keys

This command requires admin-level AWS credentials to create IAM resources.
After bootstrap, use the created credentials for deployments.

Examples:
  # Create policy and group only
  omnideploy bootstrap

  # Create policy, group, user, and access keys
  omnideploy bootstrap --user deployer --create-key

  # Show current bootstrap status
  omnideploy bootstrap status

  # Show the IAM policy document
  omnideploy bootstrap policy`,
}

var bootstrapRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the bootstrap process",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		cfg := bootstrap.DefaultConfig()
		if bootstrapRegion != "" {
			cfg.Region = bootstrapRegion
		}
		if bootstrapUser != "" {
			cfg.UserName = bootstrapUser
			cfg.CreateAccessKey = bootstrapCreateKey
		}

		b, err := bootstrap.New(ctx, cfg)
		if err != nil {
			return err
		}

		fmt.Println("Bootstrapping OmniDeploy IAM resources...")
		fmt.Printf("  Policy:  %s\n", cfg.PolicyName)
		fmt.Printf("  Group:   %s\n", cfg.GroupName)
		if cfg.UserName != "" {
			fmt.Printf("  User:    %s\n", cfg.UserName)
		}
		fmt.Println()

		result, err := b.Run(ctx)
		if err != nil {
			return err
		}

		fmt.Println("Bootstrap complete!")
		fmt.Printf("  Policy ARN: %s\n", result.PolicyARN)
		fmt.Printf("  Group:      %s\n", result.GroupName)

		if result.UserName != "" {
			fmt.Printf("  User:       %s\n", result.UserName)
		}

		if result.AccessKeyID != "" {
			fmt.Println()
			fmt.Println("Access Key Created:")
			fmt.Println("  Save these credentials - the secret will not be shown again!")
			fmt.Println()
			fmt.Printf("  AWS_ACCESS_KEY_ID=%s\n", result.AccessKeyID)
			fmt.Printf("  AWS_SECRET_ACCESS_KEY=%s\n", result.SecretAccessKey)
			fmt.Println()
			fmt.Println("Add to your shell profile or use:")
			fmt.Printf("  export AWS_ACCESS_KEY_ID=\"%s\"\n", result.AccessKeyID)
			fmt.Printf("  export AWS_SECRET_ACCESS_KEY=\"%s\"\n", result.SecretAccessKey)
			fmt.Printf("  export AWS_REGION=\"%s\"\n", cfg.Region)
		}

		return nil
	},
}

var bootstrapStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current bootstrap status",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		cfg := bootstrap.DefaultConfig()
		if bootstrapRegion != "" {
			cfg.Region = bootstrapRegion
		}

		b, err := bootstrap.New(ctx, cfg)
		if err != nil {
			return err
		}

		status, err := b.Status(ctx)
		if err != nil {
			return err
		}

		fmt.Println("OmniDeploy Bootstrap Status:")
		fmt.Println()

		if status.PolicyExists {
			fmt.Printf("  Policy:  ✓ %s\n", status.PolicyARN)
		} else {
			fmt.Printf("  Policy:  ✗ Not created\n")
		}

		if status.GroupExists {
			fmt.Printf("  Group:   ✓ %s\n", status.GroupName)
			if len(status.GroupMembers) > 0 {
				fmt.Printf("  Members: %v\n", status.GroupMembers)
			} else {
				fmt.Printf("  Members: (none)\n")
			}
		} else {
			fmt.Printf("  Group:   ✗ Not created\n")
		}

		if !status.PolicyExists || !status.GroupExists {
			fmt.Println()
			fmt.Println("Run 'omnideploy bootstrap run' to set up IAM resources.")
		}

		return nil
	},
}

var bootstrapPolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Show the IAM policy document",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("OmniDeploy IAM Policy:")
		fmt.Println()
		fmt.Println(bootstrap.PolicyJSON())
	},
}

var ecrCmd = &cobra.Command{
	Use:   "ecr",
	Short: "Manage ECR container repositories",
	Long: `Manage Amazon ECR (Elastic Container Registry) repositories.

ECR is used to store container images that are deployed to LightSail
or other AWS container services.

Examples:
  # Create a repository
  omnideploy ecr create my-app

  # List repositories
  omnideploy ecr list

  # Build and push image (reads image URI from config)
  omnideploy ecr push --config deploy.yaml

  # Build and push with explicit image URI
  omnideploy ecr push 123456789.dkr.ecr.us-west-2.amazonaws.com/my-app:latest

  # Get Docker login command
  omnideploy ecr login

  # Delete a repository
  omnideploy ecr delete my-app`,
}

var ecrCreateCmd = &cobra.Command{
	Use:   "create <repository-name>",
	Short: "Create an ECR repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		repoName := args[0]

		client, err := ecr.New(ctx, ecrRegion)
		if err != nil {
			return err
		}

		repo, created, err := client.EnsureRepository(ctx, repoName)
		if err != nil {
			return fmt.Errorf("failed to create repository: %w", err)
		}

		if created {
			fmt.Println("Repository created:")
		} else {
			fmt.Println("Repository already exists:")
		}
		fmt.Printf("  Name: %s\n", repo.Name)
		fmt.Printf("  URI:  %s\n", repo.URI)
		fmt.Println()
		fmt.Println("Update your deploy.yaml:")
		fmt.Printf("  container:\n")
		fmt.Printf("    image: %s:latest\n", repo.URI)

		return nil
	},
}

var ecrListCmd = &cobra.Command{
	Use:   "list",
	Short: "List ECR repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		client, err := ecr.New(ctx, ecrRegion)
		if err != nil {
			return err
		}

		repos, err := client.ListRepositories(ctx)
		if err != nil {
			return fmt.Errorf("failed to list repositories: %w", err)
		}

		if len(repos) == 0 {
			fmt.Println("No repositories found.")
			return nil
		}

		fmt.Println("ECR Repositories:")
		for _, r := range repos {
			fmt.Printf("  %s\n", r.Name)
			fmt.Printf("    URI: %s\n", r.URI)
		}

		return nil
	},
}

var ecrLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Get Docker login credentials for ECR",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		client, err := ecr.New(ctx, ecrRegion)
		if err != nil {
			return err
		}

		creds, err := client.GetLoginCredentials(ctx)
		if err != nil {
			return fmt.Errorf("failed to get login credentials: %w", err)
		}

		fmt.Println("Docker login command:")
		fmt.Println()
		fmt.Printf("  echo '%s' | docker login --username %s --password-stdin %s\n",
			creds.Password, creds.Username, creds.Registry)
		fmt.Println()
		fmt.Println("Or run directly:")
		fmt.Printf("  aws ecr get-login-password --region %s | docker login --username AWS --password-stdin %s\n",
			ecrRegion, creds.Registry)

		return nil
	},
}

var ecrDeleteCmd = &cobra.Command{
	Use:   "delete <repository-name>",
	Short: "Delete an ECR repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		repoName := args[0]

		if !ecrForce {
			fmt.Printf("Delete repository '%s'? This will delete all images. [y/N] ", repoName)
			var confirm string
			if _, err := fmt.Scanln(&confirm); err != nil {
				slog.Warn("failed to read confirmation", "error", err)
			}
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		client, err := ecr.New(ctx, ecrRegion)
		if err != nil {
			return err
		}

		if err := client.DeleteRepository(ctx, repoName, true); err != nil {
			return fmt.Errorf("failed to delete repository: %w", err)
		}

		fmt.Printf("Repository '%s' deleted.\n", repoName)
		return nil
	},
}

var ecrPushCmd = &cobra.Command{
	Use:   "push [image-uri]",
	Short: "Build and push a Docker image to ECR",
	Long: `Build a Docker image and push it to ECR.

The image URI can be specified as an argument or read from a config file.
This command handles ECR login automatically.

Examples:
  # Push using image URI from deploy.yaml
  omnideploy ecr push --config deploy.yaml

  # Push with explicit image URI
  omnideploy ecr push 123456789.dkr.ecr.us-west-2.amazonaws.com/my-app:latest

  # Push with custom Dockerfile and context
  omnideploy ecr push --config deploy.yaml --dockerfile Dockerfile.prod --context ./app

  # Push for specific platform (e.g., for ARM-based systems)
  omnideploy ecr push --config deploy.yaml --platform linux/amd64`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		// Determine image URI
		imageURI := ecrImage
		if len(args) > 0 {
			imageURI = args[0]
		}

		// If no image specified, try to read from config
		if imageURI == "" && configFile != "" {
			uri, err := getImageFromConfig(configFile)
			if err != nil {
				return fmt.Errorf("failed to read image from config: %w", err)
			}
			imageURI = uri
		}

		if imageURI == "" {
			return fmt.Errorf("image URI required: provide as argument, --image flag, or via --config")
		}

		// Override tag if specified
		if ecrTag != "" {
			// Replace tag in image URI
			if idx := strings.LastIndex(imageURI, ":"); idx != -1 {
				imageURI = imageURI[:idx] + ":" + ecrTag
			} else {
				imageURI = imageURI + ":" + ecrTag
			}
		}

		// Extract registry from image URI for login
		registry := extractRegistry(imageURI)
		if registry == "" {
			return fmt.Errorf("could not extract registry from image URI: %s", imageURI)
		}

		fmt.Printf("Image:      %s\n", imageURI)
		fmt.Printf("Dockerfile: %s\n", ecrDockerfile)
		fmt.Printf("Context:    %s\n", ecrContext)
		if ecrPlatform != "" {
			fmt.Printf("Platform:   %s\n", ecrPlatform)
		}
		fmt.Println()

		// Step 1: Login to ECR
		fmt.Println("Logging into ECR...")
		client, err := ecr.New(ctx, ecrRegion)
		if err != nil {
			return fmt.Errorf("failed to create ECR client: %w", err)
		}

		creds, err := client.GetLoginCredentials(ctx)
		if err != nil {
			return fmt.Errorf("failed to get ECR credentials: %w", err)
		}

		// nolint:gosec // G204: Registry URL comes from AWS ECR API, not user input
		loginCmd := exec.CommandContext(ctx, "docker", "login",
			"--username", creds.Username,
			"--password-stdin", creds.Registry)
		loginCmd.Stdin = strings.NewReader(creds.Password)
		loginCmd.Stdout = os.Stdout
		loginCmd.Stderr = os.Stderr

		if err := loginCmd.Run(); err != nil {
			return fmt.Errorf("docker login failed: %w", err)
		}
		fmt.Println("ECR login successful.")
		fmt.Println()

		// Step 2: Build the image
		fmt.Println("Building Docker image...")
		buildArgs := []string{"build", "-t", imageURI, "-f", ecrDockerfile}
		if ecrPlatform != "" {
			buildArgs = append(buildArgs, "--platform", ecrPlatform)
		}
		buildArgs = append(buildArgs, ecrContext)

		buildCmd := exec.CommandContext(ctx, "docker", buildArgs...)
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr

		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("docker build failed: %w", err)
		}
		fmt.Println("Build complete.")
		fmt.Println()

		// Step 3: Push the image
		fmt.Println("Pushing to ECR...")
		pushCmd := exec.CommandContext(ctx, "docker", "push", imageURI)
		pushCmd.Stdout = os.Stdout
		pushCmd.Stderr = os.Stderr

		if err := pushCmd.Run(); err != nil {
			return fmt.Errorf("docker push failed: %w", err)
		}

		fmt.Println()
		fmt.Printf("Successfully pushed %s\n", imageURI)
		return nil
	},
}

// getImageFromConfig reads the container.image field from a deploy config file.
func getImageFromConfig(configPath string) (string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}

	var config struct {
		Container struct {
			Image string `yaml:"image"`
		} `yaml:"container"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return "", err
	}

	if config.Container.Image == "" {
		return "", fmt.Errorf("container.image not found in config")
	}

	return config.Container.Image, nil
}

// extractRegistry extracts the registry hostname from an image URI.
// e.g., "123456789.dkr.ecr.us-west-2.amazonaws.com/my-app:latest" -> "123456789.dkr.ecr.us-west-2.amazonaws.com"
func extractRegistry(imageURI string) string {
	// Remove tag
	uri := imageURI
	if idx := strings.LastIndex(uri, ":"); idx != -1 {
		// Make sure it's a tag, not a port in the registry
		afterColon := uri[idx+1:]
		if !strings.Contains(afterColon, "/") {
			uri = uri[:idx]
		}
	}

	// Get registry (everything before first /)
	if idx := strings.Index(uri, "/"); idx != -1 {
		return uri[:idx]
	}

	return ""
}

func init() {
	upCmd.Flags().BoolVarP(&autoApprove, "yes", "y", false, "Auto-approve changes")
	destroyCmd.Flags().BoolVarP(&autoApprove, "yes", "y", false, "Auto-approve destruction")

	// Bootstrap subcommands
	bootstrapCmd.AddCommand(bootstrapRunCmd)
	bootstrapCmd.AddCommand(bootstrapStatusCmd)
	bootstrapCmd.AddCommand(bootstrapPolicyCmd)

	// Bootstrap flags
	bootstrapRunCmd.Flags().StringVar(&bootstrapUser, "user", "", "IAM user to create")
	bootstrapRunCmd.Flags().BoolVar(&bootstrapCreateKey, "create-key", false, "Create access key for user")
	bootstrapRunCmd.Flags().StringVar(&bootstrapRegion, "region", "us-west-2", "AWS region (must support LightSail containers)")
	bootstrapStatusCmd.Flags().StringVar(&bootstrapRegion, "region", "us-west-2", "AWS region")

	// ECR subcommands
	ecrCmd.AddCommand(ecrCreateCmd)
	ecrCmd.AddCommand(ecrListCmd)
	ecrCmd.AddCommand(ecrLoginCmd)
	ecrCmd.AddCommand(ecrDeleteCmd)
	ecrCmd.AddCommand(ecrPushCmd)

	// ECR flags
	ecrCmd.PersistentFlags().StringVar(&ecrRegion, "region", "us-west-2", "AWS region")
	ecrDeleteCmd.Flags().BoolVar(&ecrForce, "force", false, "Skip confirmation")

	// ECR push flags
	ecrPushCmd.Flags().StringVarP(&configFile, "config", "c", "", "Config file to read image URI from")
	ecrPushCmd.Flags().StringVar(&ecrImage, "image", "", "Image URI (overrides config)")
	ecrPushCmd.Flags().StringVar(&ecrDockerfile, "dockerfile", "Dockerfile", "Path to Dockerfile")
	ecrPushCmd.Flags().StringVar(&ecrContext, "context", ".", "Build context directory")
	ecrPushCmd.Flags().StringVar(&ecrTag, "tag", "", "Image tag (overrides tag in image URI)")
	ecrPushCmd.Flags().StringVar(&ecrPlatform, "platform", "", "Target platform (e.g., linux/amd64)")
}

func createDeployer() (*deploy.Deployer, error) {
	// Get target
	t, err := target.Get(targetName)
	if err != nil {
		return nil, fmt.Errorf("invalid target: %w", err)
	}

	// Get backend
	b, err := backend.Get(backendName)
	if err != nil {
		return nil, fmt.Errorf("invalid backend: %w", err)
	}

	// Get runtime adapter (optional)
	var r runtime.Adapter
	if runtimeName != "" {
		r, err = runtime.Get(runtimeName)
		if err != nil {
			return nil, fmt.Errorf("invalid runtime: %w", err)
		}
	}

	opts := []deploy.Option{
		deploy.WithTarget(t),
		deploy.WithBackend(b),
	}
	if r != nil {
		opts = append(opts, deploy.WithRuntime(r))
	}

	return deploy.New(opts...), nil
}
