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
	Long:  `Show the current status of your glovebox profile and whether the Dockerfile is up to date.`,
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

	p, err := profile.LoadEffective(cwd)
	if err != nil {
		return err
	}

	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	dim := color.New(color.Faint)

	if p == nil {
		yellow.Println("No profile found.")
		fmt.Println("Run 'glovebox init' to create one.")
		return nil
	}

	bold.Println("Profile:")
	fmt.Printf("  Path: %s\n", p.Path)
	fmt.Printf("  Snippets: %d\n", len(p.Snippets))
	for _, s := range p.Snippets {
		fmt.Printf("    - %s\n", s)
	}

	// Check Dockerfile status
	dockerfilePath := "Dockerfile"
	fmt.Println()
	bold.Println("Dockerfile:")

	// Check if Dockerfile exists
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		yellow.Println("  Status: Not generated")
		fmt.Println("  Run 'glovebox build' to generate.")
		return nil
	}

	// Check if we have build info
	if p.Build.DockerfileDigest == "" {
		yellow.Println("  Status: Exists but not tracked")
		fmt.Println("  Run 'glovebox build' to regenerate and track.")
		return nil
	}

	// Compare digests
	currentDigest, err := digest.CalculateFile(dockerfilePath)
	if err != nil {
		return fmt.Errorf("calculating Dockerfile digest: %w", err)
	}

	if currentDigest == p.Build.DockerfileDigest {
		green.Println("  Status: Up to date ✓")
		dim.Printf("  Last built: %s\n", p.Build.LastBuiltAt.Format("2006-01-02 15:04:05 UTC"))
		dim.Printf("  Digest: %s\n", digest.Short(currentDigest))
	} else {
		yellow.Println("  Status: Modified since generation ⚠")
		dim.Printf("  Expected: %s\n", digest.Short(p.Build.DockerfileDigest))
		dim.Printf("  Current:  %s\n", digest.Short(currentDigest))
		fmt.Println("\n  The Dockerfile has been modified directly.")
		fmt.Println("  Run 'glovebox build' to see the diff and options.")
	}

	// Check if profile would generate different content
	expectedContent, err := generator.Generate(p.Snippets)
	if err != nil {
		return fmt.Errorf("generating expected Dockerfile: %w", err)
	}
	expectedDigest := digest.Calculate(expectedContent)

	if expectedDigest != p.Build.DockerfileDigest {
		fmt.Println()
		yellow.Println("  Note: Profile has changed since last build.")
		fmt.Println("  Run 'glovebox build' to regenerate.")
	}

	return nil
}
