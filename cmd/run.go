package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/joelhelbling/glovebox/internal/docker"
	"github.com/joelhelbling/glovebox/internal/profile"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [directory]",
	Short: "Run glovebox container with a mounted directory",
	Long: `Run the glovebox container with the specified directory mounted as workspace.

If no directory is specified, the current directory is used.

The command will:
1. Check for a project profile (.glovebox/profile.yaml) and use that image
2. Fall back to glovebox:base if no project profile exists
3. Build images automatically if they don't exist

Each project gets its own persistent container. Any changes you make to the
container (installing tools, configuring editors, etc.) are preserved in the
container's writable layer. After exiting, you'll be prompted to commit
changes to the image if any were detected.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRun,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
	// Determine target directory
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Verify directory exists
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("directory not found: %s", absPath)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", absPath)
	}

	// Determine which image to use
	imageName, err := determineImage(absPath)
	if err != nil {
		return err
	}

	// Generate container name for this project
	containerName := docker.ContainerName(absPath)
	dirName := filepath.Base(absPath)

	// Check if container already exists
	containerExists := docker.ContainerExists(containerName)
	containerRunning := docker.ContainerRunning(containerName)

	fmt.Printf("Starting glovebox with workspace: %s\n", collapsePath(absPath))
	fmt.Printf("Using image: %s\n", imageName)

	// Mount workspace at /<dirName> so the prompt shows the project name
	workspacePath := "/" + dirName

	if containerRunning {
		// Container is already running - attach to it
		colorYellow.Printf("Container %s is already running. Attaching...\n", containerName)
		return attachToContainer(containerName)
	}

	if containerExists {
		// Container exists but stopped - start it
		colorDim.Printf("Container: %s (existing)\n", containerName)
		if err := startContainer(containerName, absPath, workspacePath); err != nil {
			return err
		}
	} else {
		// Create new container
		colorDim.Printf("Container: %s (new)\n", containerName)
		if err := createAndStartContainer(containerName, imageName, absPath, workspacePath); err != nil {
			return err
		}
	}

	// After container exits, check for changes and offer to commit
	return handlePostExit(containerName, imageName)
}

// attachToContainer attaches to a running container
func attachToContainer(name string) error {
	docker := exec.Command("docker", "attach", name)
	docker.Stdin = os.Stdin
	docker.Stdout = os.Stdout
	docker.Stderr = os.Stderr
	return docker.Run()
}

// startContainer starts an existing stopped container
func startContainer(name, hostPath, workspacePath string) error {
	// Start the container in attached mode
	docker := exec.Command("docker", "start", "-ai", name)
	docker.Stdin = os.Stdin
	docker.Stdout = os.Stdout
	docker.Stderr = os.Stderr
	return docker.Run()
}

// createAndStartContainer creates a new container and starts it
func createAndStartContainer(name, imageName, hostPath, workspacePath string) error {
	dockerArgs := []string{
		"run", "-it",
		"--name", name,
		"-v", fmt.Sprintf("%s:%s", hostPath, workspacePath),
		"-w", workspacePath,
		"--hostname", "glovebox",
	}

	// Add environment variables if set
	envVars := []string{
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
		"GOOGLE_API_KEY",
		"GEMINI_API_KEY",
	}
	for _, env := range envVars {
		if val := os.Getenv(env); val != "" {
			dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("%s=%s", env, val))
		}
	}

	// Add mise trusted config path
	dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("MISE_TRUSTED_CONFIG_PATHS=%s:%s/**", workspacePath, workspacePath))

	// Add image name
	dockerArgs = append(dockerArgs, imageName)

	// Run docker
	docker := exec.Command("docker", dockerArgs...)
	docker.Stdin = os.Stdin
	docker.Stdout = os.Stdout
	docker.Stderr = os.Stderr

	return docker.Run()
}

// handlePostExit checks for container changes and offers to commit them
func handlePostExit(containerName, imageName string) error {
	// Get the diff
	changes, err := getContainerDiff(containerName)
	if err != nil {
		// Don't fail on diff errors, just skip the commit prompt
		return nil
	}

	if len(changes) == 0 {
		return nil
	}

	// Filter and summarize changes
	summary := summarizeChanges(changes)
	if len(summary) == 0 {
		return nil
	}

	fmt.Println()
	colorYellow.Println("Changes detected in container:")
	for _, s := range summary {
		fmt.Printf("  %s\n", s)
	}

	fmt.Println()
	fmt.Println("What would you like to do?")
	fmt.Println("  [y]es   - commit changes to image (fresh container next run)")
	fmt.Println("  [n]o    - keep uncommitted changes (resume this container next run)")
	fmt.Println("  [e]rase - discard changes (fresh container next run)")
	fmt.Print("Choice: ")

	switch getPostExitChoice() {
	case "yes":
		if err := commitContainer(containerName, imageName); err != nil {
			colorYellow.Printf("Warning: could not commit changes: %v\n", err)
			return nil
		}
		colorGreen.Printf("Changes committed to %s\n", imageName)

		// Remove the container so next run starts fresh from the committed image
		if err := deleteContainer(containerName); err != nil {
			colorYellow.Printf("Warning: could not remove container: %v\n", err)
		}
	case "erase":
		if err := deleteContainer(containerName); err != nil {
			colorYellow.Printf("Warning: could not remove container: %v\n", err)
			return nil
		}
		colorDim.Println("Container removed. Next run will start fresh.")
	default:
		// "no" - leave container as-is
		colorDim.Println("Changes kept in container.")
	}

	return nil
}

// getContainerDiff returns the filesystem changes in a container
func getContainerDiff(name string) ([]string, error) {
	cmd := exec.Command("docker", "diff", name)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var changes []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line != "" {
			changes = append(changes, line)
		}
	}
	return changes, nil
}

// summarizeChanges filters and summarizes container changes for display
func summarizeChanges(changes []string) []string {
	// Group changes by top-level directory
	dirCounts := make(map[string]int)
	var importantChanges []string

	for _, change := range changes {
		// Parse change type and path (e.g., "A /home/ubuntu/.bashrc")
		parts := strings.SplitN(change, " ", 2)
		if len(parts) != 2 {
			continue
		}
		changeType := parts[0]
		path := parts[1]

		// Skip workspace mount changes (those are on the host)
		if strings.HasPrefix(path, "/workspace") {
			continue
		}

		// Count by top-level meaningful directory
		pathParts := strings.Split(strings.TrimPrefix(path, "/"), "/")
		if len(pathParts) >= 2 {
			topDir := "/" + pathParts[0] + "/" + pathParts[1]
			dirCounts[topDir]++
		}

		// Highlight specific important changes
		if strings.Contains(path, "/.linuxbrew/Cellar/") {
			// Extract package name from Cellar path
			cellarParts := strings.Split(path, "/Cellar/")
			if len(cellarParts) > 1 {
				pkgParts := strings.Split(cellarParts[1], "/")
				if len(pkgParts) > 0 {
					importantChanges = append(importantChanges, fmt.Sprintf("%s brew package: %s", changeType, pkgParts[0]))
				}
			}
		}
	}

	// Dedupe important changes
	seen := make(map[string]bool)
	var result []string
	for _, c := range importantChanges {
		if !seen[c] {
			seen[c] = true
			result = append(result, c)
		}
	}

	// Add summary counts for directories with many changes
	for dir, count := range dirCounts {
		if count > 5 {
			result = append(result, fmt.Sprintf("%d changes in %s", count, dir))
		}
	}

	// If we have too many specific changes, just show counts
	if len(result) > 10 {
		result = result[:10]
		result = append(result, fmt.Sprintf("... and %d more changes", len(changes)-10))
	}

	// If no meaningful summary, show total count
	if len(result) == 0 && len(changes) > 0 {
		result = append(result, fmt.Sprintf("%d filesystem changes", len(changes)))
	}

	return result
}

// getPostExitChoice prompts user for what to do with container changes
// Returns "yes", "no", or "erase"
func getPostExitChoice() string {
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "no"
	}
	response = strings.TrimSpace(strings.ToLower(response))
	switch response {
	case "y", "yes":
		return "yes"
	case "e", "erase":
		return "erase"
	default:
		return "no"
	}
}

// commitContainer commits container changes to its image
func commitContainer(containerName, imageName string) error {
	cmd := exec.Command("docker", "commit", containerName, imageName)
	return cmd.Run()
}

// deleteContainer removes a container without printing
func deleteContainer(containerName string) error {
	cmd := exec.Command("docker", "container", "rm", containerName)
	return cmd.Run()
}

// determineImage figures out which Docker image to use for the given directory
func determineImage(dir string) (string, error) {
	// Check for project profile
	projectProfile, err := profile.LoadProject(dir)
	if err != nil {
		return "", fmt.Errorf("checking project profile: %w", err)
	}

	if projectProfile != nil {
		// Project profile exists - use project image
		imageName := projectProfile.ImageName()

		if !docker.ImageExists(imageName) {
			colorYellow.Printf("Project image %s not found. Building...\n\n", imageName)
			if err := buildProjectImage(projectProfile); err != nil {
				return "", fmt.Errorf("building project image: %w", err)
			}
			fmt.Println()
		}

		return imageName, nil
	}

	// No project profile - use base image
	if !docker.ImageExists("glovebox:base") {
		// Check if global profile exists
		globalProfile, err := profile.LoadGlobal()
		if err != nil {
			return "", fmt.Errorf("checking global profile: %w", err)
		}

		if globalProfile == nil {
			return "", fmt.Errorf("no glovebox profile found.\nRun 'glovebox init --global' to create a global profile first")
		}

		colorYellow.Println("Base image glovebox:base not found. Building...")
		if err := buildBaseImage(); err != nil {
			return "", fmt.Errorf("building base image: %w", err)
		}
		fmt.Println()
	}

	return "glovebox:base", nil
}
