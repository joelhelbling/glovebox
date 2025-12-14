package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/joelhelbling/glovebox/internal/docker"
	"github.com/joelhelbling/glovebox/internal/ui"
	"github.com/spf13/cobra"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Discard container changes and start fresh",
	Long: `Remove the current project's container, discarding any uncommitted changes.

The next 'glovebox run' will create a fresh container from the image.
Use this to discard experimental changes or reset to a clean state.

This only removes the container, not the image. Your base configuration
and any previously committed changes are preserved in the image.`,
	RunE: runReset,
}

func init() {
	rootCmd.AddCommand(resetCmd)
}

func runReset(cmd *cobra.Command, args []string) error {
	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	absPath, err := filepath.Abs(cwd)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Get container name for this project
	containerName := docker.ContainerName(absPath)

	// Check if container exists
	if !docker.ContainerExists(containerName) {
		fmt.Println("No container found for this project. Nothing to reset.")
		return nil
	}

	// Remove the container
	prompt := ui.NewPrompt()
	rmCmd := exec.Command("docker", "container", "rm", containerName)
	if err := rmCmd.Run(); err != nil {
		return fmt.Errorf("removing container: %w", err)
	}

	fmt.Print(prompt.RenderEraseSuccess())

	return nil
}
