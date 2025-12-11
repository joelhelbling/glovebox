package cmd

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/joelhelbling/glovebox/internal/digest"
	"github.com/joelhelbling/glovebox/internal/generator"
	"github.com/joelhelbling/glovebox/internal/profile"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show profile and Dockerfile status",
	Long:  `Show the current status of your glovebox profiles, images, and Dockerfiles.`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	dim := color.New(color.Faint)

	// Check global profile
	globalProfile, err := profile.LoadGlobal()
	if err != nil {
		return err
	}

	bold.Println("Base Image (glovebox:base):")
	if globalProfile == nil {
		yellow.Println("  Profile: Not configured")
		fmt.Println("  Run 'glovebox init --global' to create.")
	} else {
		globalPath, _ := profile.GlobalPath()
		fmt.Printf("  Profile: %s\n", globalPath)
		fmt.Printf("  Mods: %d\n", len(globalProfile.Mods))
		for _, s := range globalProfile.Mods {
			dim.Printf("    - %s\n", s)
		}

		// Check base Dockerfile
		dockerfilePath := globalProfile.DockerfilePath()
		showDockerfileStatus(globalProfile, dockerfilePath, generator.GenerateBase, green, yellow, dim)

		// Check if image exists
		if imageExists("glovebox:base") {
			green.Println("  Image: Built ✓")
		} else {
			yellow.Println("  Image: Not built")
			fmt.Println("  Run 'glovebox build --base' to build.")
		}
	}

	fmt.Println()

	// Check project profile
	projectProfile, err := profile.LoadProject(cwd)
	if err != nil {
		return err
	}

	bold.Println("Project Image:")
	if projectProfile == nil {
		dim.Println("  Profile: None (will use glovebox:base)")
		fmt.Println("  Run 'glovebox init' to create a project-specific profile.")
	} else {
		fmt.Printf("  Profile: %s\n", projectProfile.Path)
		fmt.Printf("  Mods: %d\n", len(projectProfile.Mods))
		for _, s := range projectProfile.Mods {
			dim.Printf("    - %s\n", s)
		}

		// Check project Dockerfile - need base mods for proper generation
		dockerfilePath := projectProfile.DockerfilePath()
		var baseMods []string
		if globalProfile != nil {
			baseMods = globalProfile.Mods
		}
		showProjectDockerfileStatus(projectProfile, dockerfilePath, baseMods, green, yellow, dim)

		// Check if image exists
		imageName := projectProfile.ImageName()
		fmt.Printf("  Image name: %s\n", imageName)
		if imageExists(imageName) {
			green.Println("  Image: Built ✓")
		} else {
			yellow.Println("  Image: Not built")
			fmt.Println("  Run 'glovebox build' to build.")
		}
	}

	// Show volumes section
	fmt.Println()
	bold.Println("Volumes:")
	showVolumeStatus(cwd, green, yellow, dim)

	return nil
}

func showVolumeStatus(cwd string, green, yellow, dim *color.Color) {
	// Calculate volume name (same logic as run.go)
	absPath, err := filepath.Abs(cwd)
	if err != nil {
		yellow.Printf("  Error: %v\n", err)
		return
	}

	hash := sha256.Sum256([]byte(absPath))
	shortHash := fmt.Sprintf("%x", hash)[:7]
	dirName := filepath.Base(absPath)
	volumeName := fmt.Sprintf("glovebox-%s-%s-home", dirName, shortHash)

	// Workspace mount
	fmt.Printf("  Workspace: %s → /%s\n", absPath, dirName)

	// Home volume
	fmt.Printf("  Home volume: %s → /home/ubuntu\n", volumeName)
	if volumeExists(volumeName) {
		green.Println("    Status: Exists ✓")
	} else {
		dim.Println("    Status: Will be created on first run")
	}
}

func volumeExists(name string) bool {
	cmd := exec.Command("docker", "volume", "inspect", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	// Check that output contains the volume name (not just an error)
	return strings.Contains(string(output), name)
}

func showDockerfileStatus(p *profile.Profile, dockerfilePath string, generateFunc func([]string) (string, error), green, yellow, dim *color.Color) {
	fmt.Printf("  Dockerfile: %s\n", dockerfilePath)

	// Check if Dockerfile exists
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		yellow.Println("    Status: Not generated")
		return
	}

	// Check if we have build info
	if p.Build.DockerfileDigest == "" {
		yellow.Println("    Status: Exists but not tracked")
		return
	}

	// Compare digests
	currentDigest, err := digest.CalculateFile(dockerfilePath)
	if err != nil {
		yellow.Printf("    Status: Error reading (%v)\n", err)
		return
	}

	if currentDigest == p.Build.DockerfileDigest {
		green.Println("    Status: Up to date ✓")
		dim.Printf("    Last built: %s\n", p.Build.LastBuiltAt.Local().Format("2006-01-02 15:04:05 MST"))
	} else {
		yellow.Println("    Status: Modified since generation ⚠")
	}

	// Check if profile would generate different content
	expectedContent, err := generateFunc(p.Mods)
	if err != nil {
		return
	}
	expectedDigest := digest.Calculate(expectedContent)

	if expectedDigest != p.Build.DockerfileDigest {
		yellow.Println("    Note: Profile has changed since last build")
	}
}

func showProjectDockerfileStatus(p *profile.Profile, dockerfilePath string, baseMods []string, green, yellow, dim *color.Color) {
	fmt.Printf("  Dockerfile: %s\n", dockerfilePath)

	// Check if Dockerfile exists
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		yellow.Println("    Status: Not generated")
		return
	}

	// Check if we have build info
	if p.Build.DockerfileDigest == "" {
		yellow.Println("    Status: Exists but not tracked")
		return
	}

	// Compare digests
	currentDigest, err := digest.CalculateFile(dockerfilePath)
	if err != nil {
		yellow.Printf("    Status: Error reading (%v)\n", err)
		return
	}

	if currentDigest == p.Build.DockerfileDigest {
		green.Println("    Status: Up to date ✓")
		dim.Printf("    Last built: %s\n", p.Build.LastBuiltAt.Local().Format("2006-01-02 15:04:05 MST"))
	} else {
		yellow.Println("    Status: Modified since generation ⚠")
	}

	// Check if profile would generate different content
	expectedContent, err := generator.GenerateProject(p.Mods, baseMods)
	if err != nil {
		return
	}
	expectedDigest := digest.Calculate(expectedContent)

	if expectedDigest != p.Build.DockerfileDigest {
		yellow.Println("    Note: Profile has changed since last build")
	}
}
