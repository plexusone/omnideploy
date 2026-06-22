// Package main is the omnideploy CLI entry point.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/plexusone/omnideploy/backend"
	"github.com/plexusone/omnideploy/deploy"
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
			fmt.Scanln(&confirm)
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

func init() {
	upCmd.Flags().BoolVarP(&autoApprove, "yes", "y", false, "Auto-approve changes")
	destroyCmd.Flags().BoolVarP(&autoApprove, "yes", "y", false, "Auto-approve destruction")
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
