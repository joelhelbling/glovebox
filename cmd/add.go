package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/joelhelbling/glovebox/internal/profile"
	"github.com/joelhelbling/glovebox/internal/snippet"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <snippet>",
	Short: "Add a snippet to your profile",
	Long: `Add a snippet to your glovebox profile.

Run 'glovebox list' to see available snippets.

Examples:
  glovebox add shells/fish
  glovebox add ai/claude-code
  glovebox add custom/my-tool`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	snippetID := args[0]

	// Verify snippet exists
	if _, err := snippet.Load(snippetID); err != nil {
		return fmt.Errorf("snippet '%s' not found. Run 'glovebox list' to see available snippets", snippetID)
	}

	// Load effective profile
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	p, err := profile.LoadEffective(cwd)
	if err != nil {
		return err
	}

	if p == nil {
		return fmt.Errorf("no profile found. Run 'glovebox init' first")
	}

	// Add snippet
	if !p.AddSnippet(snippetID) {
		fmt.Printf("Snippet '%s' is already in your profile.\n", snippetID)
		return nil
	}

	// Save profile
	if err := p.Save(); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	green := color.New(color.FgGreen)
	green.Printf("✓ Added '%s' to profile\n", snippetID)
	fmt.Println("\nRun 'glovebox build' to regenerate your Dockerfile.")

	return nil
}
