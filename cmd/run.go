package cmd

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fatih/color"
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

Each project gets its own home directory volume, so tool installations,
shell history, and configurations persist between sessions.`,
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

	// Generate volume name for home directory
	hash := sha256.Sum256([]byte(absPath))
	shortHash := fmt.Sprintf("%x", hash)[:7]
	dirName := filepath.Base(absPath)
	volumeName := fmt.Sprintf("glovebox-%s-%s-home", dirName, shortHash)

	// Ensure volume exists
	if err := ensureVolume(volumeName); err != nil {
		return err
	}

	fmt.Printf("Starting glovebox with workspace: %s\n", absPath)
	fmt.Printf("Using image: %s\n", imageName)
	fmt.Printf("Using home volume: %s\n", volumeName)

	// Build docker run command
	// Mount workspace at /<dirName> so the prompt shows the project name
	workspacePath := "/" + dirName
	dockerArgs := []string{
		"run", "-it", "--rm",
		"-v", fmt.Sprintf("%s:%s", absPath, workspacePath),
		"-w", workspacePath,
		"-v", fmt.Sprintf("%s:/home/ubuntu", volumeName),
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

// determineImage figures out which Docker image to use for the given directory
func determineImage(dir string) (string, error) {
	yellow := color.New(color.FgYellow)
	green := color.New(color.FgGreen)

	// Check for project profile
	projectProfile, err := profile.LoadProject(dir)
	if err != nil {
		return "", err
	}

	if projectProfile != nil {
		// Project profile exists - use project image
		imageName := projectProfile.ImageName()

		if !imageExists(imageName) {
			yellow.Printf("Project image %s not found. Building...\n\n", imageName)
			if err := buildProjectImage(projectProfile, green, yellow); err != nil {
				return "", fmt.Errorf("building project image: %w", err)
			}
			fmt.Println()
		}

		return imageName, nil
	}

	// No project profile - use base image
	if !imageExists("glovebox:base") {
		// Check if global profile exists
		globalProfile, err := profile.LoadGlobal()
		if err != nil {
			return "", err
		}

		if globalProfile == nil {
			return "", fmt.Errorf("no glovebox profile found.\nRun 'glovebox init --global' to create a global profile first")
		}

		yellow.Println("Base image glovebox:base not found. Building...\n")
		if err := buildBaseImage(green, yellow); err != nil {
			return "", fmt.Errorf("building base image: %w", err)
		}
		fmt.Println()
	}

	return "glovebox:base", nil
}

func ensureVolume(name string) error {
	// Check if volume exists
	check := exec.Command("docker", "volume", "inspect", name)
	if err := check.Run(); err == nil {
		return nil // Volume exists
	}

	// Create volume
	fmt.Printf("Creating home volume: %s\n", name)
	create := exec.Command("docker", "volume", "create", name)
	create.Stdout = os.Stdout
	create.Stderr = os.Stderr
	return create.Run()
}
