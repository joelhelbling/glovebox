package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/joelhelbling/glovebox/internal/docker"
	"github.com/joelhelbling/glovebox/internal/mod"
	"github.com/joelhelbling/glovebox/internal/profile"
	"github.com/joelhelbling/glovebox/internal/ui"
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

	// Mount workspace at /<dirName> so the prompt shows the project name
	workspacePath := "/" + dirName

	// Determine container status for banner
	var containerStatus string
	if containerRunning {
		containerStatus = "running"
	} else if containerExists {
		containerStatus = "existing"
	} else {
		containerStatus = "new"
	}

	// Determine OS from profile
	osName := getOSFromProfile(absPath)

	// Get passthrough env vars for banner (only relevant for new containers)
	var passthroughVars []string
	if !containerExists {
		passthroughEnv, err := profile.EffectivePassthroughEnv(absPath)
		if err != nil {
			colorYellow.Printf("Warning: could not load passthrough env: %v\n", err)
		} else {
			result := docker.BuildRunArgs(docker.RunArgsConfig{
				ContainerName:  containerName,
				ImageName:      imageName,
				HostPath:       absPath,
				WorkspacePath:  workspacePath,
				PassthroughEnv: passthroughEnv,
				EnvLookup:      os.Getenv,
			})
			passthroughVars = result.PassedVars
		}
	}

	// Display the banner
	banner := ui.NewBanner()
	banner.Print(ui.BannerInfo{
		Workspace:       collapsePath(absPath),
		OS:              osName,
		Image:           imageName,
		Container:       containerName,
		ContainerStatus: containerStatus,
		PassthroughEnv:  passthroughVars,
	})

	if containerRunning {
		// Container is already running - attach to it
		colorYellow.Printf("Attaching to running container...\n")
		return attachToContainer(containerName)
	}

	if containerExists {
		// Container exists but stopped - start it
		if err := startContainer(containerName, absPath, workspacePath); err != nil {
			return err
		}
	} else {
		// Create new container (passthrough already computed above)
		if err := createAndStartContainerWithEnv(containerName, imageName, absPath, workspacePath, passthroughVars); err != nil {
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
	return ignoreExitError(docker.Run())
}

// ignoreExitError filters out normal container exit codes while preserving
// Docker-specific errors that indicate real problems.
//
// Exit codes:
//   - 125: Docker daemon error (failed to create/start container)
//   - 126: Command cannot be invoked (permission denied)
//   - 127: Command not found in container
//   - 137: Container killed by SIGKILL (often OOM killer)
//   - Other: Normal exit (including non-zero from last shell command)
func ignoreExitError(err error) error {
	if err == nil {
		return nil
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return err // Not an exit error, return as-is
	}

	code := exitErr.ExitCode()
	switch {
	case code >= 125 && code <= 127:
		// Docker daemon errors - these are real failures
		return fmt.Errorf("docker error (exit %d): %w", code, err)
	case code == 137:
		// SIGKILL - often OOM, worth mentioning
		return fmt.Errorf("container was killed (exit 137, possibly out of memory)")
	default:
		// Normal container exit, ignore
		return nil
	}
}

// startContainer starts an existing stopped container
func startContainer(name, hostPath, workspacePath string) error {
	// Start the container in attached mode
	docker := exec.Command("docker", "start", "-ai", name)
	docker.Stdin = os.Stdin
	docker.Stdout = os.Stdout
	docker.Stderr = os.Stderr
	return ignoreExitError(docker.Run())
}

// createAndStartContainerWithEnv creates a new container with pre-computed env vars
func createAndStartContainerWithEnv(name, imageName, hostPath, workspacePath string, _ []string) error {
	// Get passthrough env config from profiles
	passthroughEnv, err := profile.EffectivePassthroughEnv(hostPath)
	if err != nil {
		// Non-fatal: continue without passthrough vars
		passthroughEnv = nil
	}

	// Build docker run arguments
	result := docker.BuildRunArgs(docker.RunArgsConfig{
		ContainerName:  name,
		ImageName:      imageName,
		HostPath:       hostPath,
		WorkspacePath:  workspacePath,
		PassthroughEnv: passthroughEnv,
		EnvLookup:      os.Getenv,
	})

	// Run docker
	cmd := exec.Command("docker", result.Args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return ignoreExitError(cmd.Run())
}

// handlePostExit shows a summary of container changes (no prompt)
func handlePostExit(containerName, imageName string) error {
	// Get the diff
	changes, err := getContainerDiff(containerName)
	if err != nil {
		// Don't fail on diff errors, just show simple exit
		prompt := ui.NewPrompt()
		prompt.PrintExitSummary(nil)
		return nil
	}

	// Filter and summarize changes
	summary := summarizeChanges(changes)

	// Display the exit summary (with or without changes)
	prompt := ui.NewPrompt()
	prompt.PrintExitSummary(summary)

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

// isNoiseChange returns true for changes that are expected every session
// and don't represent meaningful modifications worth mentioning
func isNoiseChange(path string) bool {
	// Exact paths that are always noise (parent dirs marked changed due to children)
	noisePaths := []string{
		"/home",
		"/home/dev",
		"/root",
		"/var",
		"/var/log",
		"/var/cache",
	}

	for _, p := range noisePaths {
		if path == p {
			return true
		}
	}

	noisePatterns := []string{
		// Shell history files
		".bash_history",
		".zsh_history",
		".local/share/fish/fish_history",
		".history",
		// Cache directories
		"/.cache/",
		"/.local/share/recently-used",
		// Temp files
		"/tmp/",
		"/var/tmp/",
		// Lock files
		".lock",
		".pid",
		// Editor swap/backup files
		".swp",
		".swo",
		"~",
		// Logs
		"/var/log/",
		// Package manager caches
		"/var/cache/",
		"/var/lib/apt/",
		"/var/lib/dpkg/",
	}

	for _, pattern := range noisePatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

// summarizeChanges filters and summarizes container changes for display.
// Returns nil if only noise changes were detected.
func summarizeChanges(changes []string) []string {
	var brewPackages []string
	var configFiles []string
	var otherChanges []string
	meaningfulCount := 0

	for _, change := range changes {
		// Parse change type and path (e.g., "A /home/dev/.bashrc")
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

		// Skip noise
		if isNoiseChange(path) {
			continue
		}

		meaningfulCount++

		// Categorize the change
		switch {
		case strings.Contains(path, "/.linuxbrew/Cellar/"):
			// Homebrew package
			cellarParts := strings.Split(path, "/Cellar/")
			if len(cellarParts) > 1 {
				pkgParts := strings.Split(cellarParts[1], "/")
				if len(pkgParts) > 0 {
					brewPackages = append(brewPackages, pkgParts[0])
				}
			}
		case strings.Contains(path, "/home/dev/.") || strings.Contains(path, "/root/."):
			// Dotfile/config file
			pathParts := strings.Split(path, "/")
			if len(pathParts) > 0 {
				filename := pathParts[len(pathParts)-1]
				if changeType == "A" {
					configFiles = append(configFiles, "added "+filename)
				} else if changeType == "C" {
					configFiles = append(configFiles, "modified "+filename)
				}
			}
		default:
			// Other meaningful change
			if changeType == "A" {
				otherChanges = append(otherChanges, "added "+path)
			} else if changeType == "C" {
				otherChanges = append(otherChanges, "modified "+path)
			} else if changeType == "D" {
				otherChanges = append(otherChanges, "deleted "+path)
			}
		}
	}

	// If no meaningful changes, return nil
	if meaningfulCount == 0 {
		return nil
	}

	var result []string

	// Dedupe and add brew packages
	seen := make(map[string]bool)
	for _, pkg := range brewPackages {
		if !seen[pkg] {
			seen[pkg] = true
			result = append(result, "brew install "+pkg)
		}
	}

	// Dedupe and add config files (limit to 5)
	seen = make(map[string]bool)
	configCount := 0
	for _, cf := range configFiles {
		if !seen[cf] && configCount < 5 {
			seen[cf] = true
			result = append(result, cf)
			configCount++
		}
	}
	if len(configFiles) > 5 {
		result = append(result, fmt.Sprintf("...and %d more config changes", len(configFiles)-5))
	}

	// Add other changes (limit to 3)
	if len(otherChanges) > 0 {
		limit := 3
		if len(otherChanges) < limit {
			limit = len(otherChanges)
		}
		result = append(result, otherChanges[:limit]...)
		if len(otherChanges) > 3 {
			result = append(result, fmt.Sprintf("...and %d more changes", len(otherChanges)-3))
		}
	}

	return result
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

// getOSFromProfile determines the OS name from the effective profile
func getOSFromProfile(dir string) string {
	// Try project profile first, then global
	p, err := profile.LoadProject(dir)
	if err != nil || p == nil {
		p, err = profile.LoadGlobal()
		if err != nil || p == nil {
			return ""
		}
	}

	// Look for OS mod in profile
	for _, modID := range p.Mods {
		m, err := mod.Load(modID)
		if err != nil {
			continue
		}
		if m.Category == "os" {
			return m.Name
		}
	}
	return ""
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
