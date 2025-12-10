package cmd

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [directory]",
	Short: "Run glovebox container with a mounted directory",
	Long: `Run the glovebox container with the specified directory mounted as workspace.

If no directory is specified, the current directory is used.

Each unique directory path gets its own mise volume, so tool installations
are cached between sessions for the same project.`,
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

	dirName := filepath.Base(absPath)
	containerPath := filepath.Join("/home/ubuntu", dirName)

	// Generate unique volume name
	hash := sha256.Sum256([]byte(absPath))
	shortHash := fmt.Sprintf("%x", hash)[:7]
	volumeName := fmt.Sprintf("glovebox-%s-%s", dirName, shortHash)

	// Ensure volume exists
	if err := ensureVolume(volumeName); err != nil {
		return err
	}

	fmt.Printf("Starting glovebox with workspace: %s -> %s\n", absPath, containerPath)
	fmt.Printf("Using mise volume: %s\n", volumeName)

	// Build docker run command
	dockerArgs := []string{
		"run", "-it", "--rm",
		"-v", fmt.Sprintf("%s:%s", absPath, containerPath),
		"-w", containerPath,
		"-v", fmt.Sprintf("%s:/home/ubuntu/.local/share/mise", volumeName),
		"--hostname", "glovebox",
	}

	// Add config directory mounts if they exist
	home, _ := os.UserHomeDir()
	if home != "" {
		anthropicDir := filepath.Join(home, ".anthropic")
		if _, err := os.Stat(anthropicDir); err == nil {
			dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/ubuntu/.anthropic", anthropicDir))
		}

		geminiDir := filepath.Join(home, ".config", "gemini")
		if _, err := os.Stat(geminiDir); err == nil {
			dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/ubuntu/.config/gemini", geminiDir))
		}
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
	dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("MISE_TRUSTED_CONFIG_PATHS=%s:%s/**", containerPath, containerPath))

	// Add image name
	dockerArgs = append(dockerArgs, "glovebox")

	// Run docker
	docker := exec.Command("docker", dockerArgs...)
	docker.Stdin = os.Stdin
	docker.Stdout = os.Stdout
	docker.Stderr = os.Stderr

	return docker.Run()
}

func ensureVolume(name string) error {
	// Check if volume exists
	check := exec.Command("docker", "volume", "inspect", name)
	if err := check.Run(); err == nil {
		return nil // Volume exists
	}

	// Create volume
	fmt.Printf("Creating mise volume: %s\n", name)
	create := exec.Command("docker", "volume", "create", name)
	create.Stdout = os.Stdout
	create.Stderr = os.Stderr
	return create.Run()
}
