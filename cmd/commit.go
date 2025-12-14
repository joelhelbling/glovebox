package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/joelhelbling/glovebox/internal/docker"
	"github.com/joelhelbling/glovebox/internal/profile"
	"github.com/joelhelbling/glovebox/internal/ui"
	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit container changes to the image",
	Long: `Commit changes from the current project's container to its image.

This persists any modifications made during glovebox sessions (installed
packages, configuration changes, etc.) to the Docker image. The container
is then removed so the next 'glovebox run' starts fresh from the updated image.

Use this after installing tools or making configuration changes you want
to keep permanently.`,
	RunE: runCommit,
}

func init() {
	rootCmd.AddCommand(commitCmd)
}

func runCommit(cmd *cobra.Command, args []string) error {
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
		return fmt.Errorf("no container found for this project\nRun 'glovebox run' first to create a container")
	}

	// Determine image name
	imageName, err := getImageNameForCommit(absPath)
	if err != nil {
		return err
	}

	// Commit the container
	prompt := ui.NewPrompt()
	fmt.Printf("Committing container to %s...\n", imageName)

	commitCmd := exec.Command("docker", "commit", containerName, imageName)
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("committing container: %w", err)
	}

	// Remove the container
	rmCmd := exec.Command("docker", "container", "rm", containerName)
	if err := rmCmd.Run(); err != nil {
		fmt.Print(prompt.RenderWarning(fmt.Sprintf("could not remove container: %v", err)))
	}

	fmt.Print(prompt.RenderCommitSuccess(imageName))
	fmt.Println("Next 'glovebox run' will start fresh from the updated image.")

	return nil
}

// getImageNameForCommit determines which image to commit to
func getImageNameForCommit(dir string) (string, error) {
	// Check for project profile first
	projectProfile, err := profile.LoadProject(dir)
	if err != nil {
		return "", fmt.Errorf("checking project profile: %w", err)
	}

	if projectProfile != nil {
		return projectProfile.ImageName(), nil
	}

	// No project profile - use base image
	return "glovebox:base", nil
}
