package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone <repository>",
	Short: "Clone a git repository and start glovebox in it",
	Long: `Clone a git repository and start glovebox in the cloned directory.

Repository can be:
  - user/repo    (assumes GitHub, e.g., joelhelbling/glovebox)
  - Full URL     (GitHub, GitLab, Bitbucket, or any git URL)

Examples:
  glovebox clone rails/rails
  glovebox clone https://gitlab.com/user/repo.git`,
	Args: cobra.ExactArgs(1),
	RunE: runClone,
}

func init() {
	rootCmd.AddCommand(cloneCmd)
}

func runClone(cmd *cobra.Command, args []string) error {
	repoArg := args[0]

	// Convert user/repo format to GitHub URL if it doesn't look like a URL
	var repoURL string
	if !strings.Contains(repoArg, "://") && !strings.Contains(repoArg, "@") {
		repoURL = fmt.Sprintf("https://github.com/%s.git", repoArg)
	} else {
		repoURL = repoArg
	}

	// Extract directory name from URL
	cloneDir := strings.TrimSuffix(filepath.Base(repoURL), ".git")

	fmt.Printf("Cloning %s...\n", repoURL)

	// Clone the repository
	gitClone := exec.Command("git", "clone", repoURL)
	gitClone.Stdout = os.Stdout
	gitClone.Stderr = os.Stderr
	if err := gitClone.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	fmt.Printf("Entering %s and starting glovebox...\n", cloneDir)

	// Change to cloned directory and run glovebox
	absPath, err := filepath.Abs(cloneDir)
	if err != nil {
		return fmt.Errorf("resolving cloned directory path: %w", err)
	}

	// Run glovebox in the cloned directory
	return runRunWithPath(absPath)
}

// runRunWithPath is a helper to run glovebox with a specific path
func runRunWithPath(path string) error {
	// Reuse the run command logic
	return runRun(nil, []string{path})
}
