package cmd

import (
	"fmt"
	"os"

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
		fmt.Printf("  Snippets: %d\n", len(globalProfile.Snippets))
		for _, s := range globalProfile.Snippets {
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
		fmt.Printf("  Snippets: %d\n", len(projectProfile.Snippets))
		for _, s := range projectProfile.Snippets {
			dim.Printf("    - %s\n", s)
		}

		// Check project Dockerfile - need base snippets for proper generation
		dockerfilePath := projectProfile.DockerfilePath()
		var baseSnippets []string
		if globalProfile != nil {
			baseSnippets = globalProfile.Snippets
		}
		showProjectDockerfileStatus(projectProfile, dockerfilePath, baseSnippets, green, yellow, dim)

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

	return nil
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
		dim.Printf("    Last built: %s\n", p.Build.LastBuiltAt.Format("2006-01-02 15:04:05 UTC"))
	} else {
		yellow.Println("    Status: Modified since generation ⚠")
	}

	// Check if profile would generate different content
	expectedContent, err := generateFunc(p.Snippets)
	if err != nil {
		return
	}
	expectedDigest := digest.Calculate(expectedContent)

	if expectedDigest != p.Build.DockerfileDigest {
		yellow.Println("    Note: Profile has changed since last build")
	}
}

func showProjectDockerfileStatus(p *profile.Profile, dockerfilePath string, baseSnippets []string, green, yellow, dim *color.Color) {
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
		dim.Printf("    Last built: %s\n", p.Build.LastBuiltAt.Format("2006-01-02 15:04:05 UTC"))
	} else {
		yellow.Println("    Status: Modified since generation ⚠")
	}

	// Check if profile would generate different content
	expectedContent, err := generator.GenerateProject(p.Snippets, baseSnippets)
	if err != nil {
		return
	}
	expectedDigest := digest.Calculate(expectedContent)

	if expectedDigest != p.Build.DockerfileDigest {
		yellow.Println("    Note: Profile has changed since last build")
	}
}
