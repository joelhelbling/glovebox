package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joelhelbling/glovebox/internal/docker"
	"github.com/spf13/cobra"
)

var diffRaw bool

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show changes in the container's filesystem",
	Long: `Show filesystem changes in the current project's container.

By default, changes are grouped by category and noise (history files,
cache, etc.) is filtered out. Use --raw to see all changes as reported
by Docker.

Change types:
  A = Added
  C = Changed
  D = Deleted`,
	RunE: runDiff,
}

func init() {
	diffCmd.Flags().BoolVar(&diffRaw, "raw", false, "Show raw docker diff output (no filtering)")
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
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
		fmt.Println("No container found for this project.")
		return nil
	}

	// Get the diff
	diffCmd := exec.Command("docker", "diff", containerName)
	output, err := diffCmd.Output()
	if err != nil {
		return fmt.Errorf("getting container diff: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Println("No changes detected in container.")
		return nil
	}

	if diffRaw {
		// Raw mode: just print docker diff output
		fmt.Println(string(output))
		return nil
	}

	// Use the shared filterNoise function
	allChanges := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			allChanges = append(allChanges, line)
		}
	}

	meaningful := filterNoise(allChanges)
	noiseCount := len(allChanges) - len(meaningful)

	// Count workspace changes separately
	workspaceCount := 0
	for _, line := range allChanges {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 && strings.HasPrefix(parts[1], "/workspace") {
			workspaceCount++
		}
	}

	// Categorize meaningful changes
	var (
		brew   []string
		config []string
		system []string
		other  []string
	)

	for _, line := range meaningful {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		path := parts[1]

		switch {
		case strings.Contains(path, "/.linuxbrew/"):
			brew = append(brew, line)
		case strings.Contains(path, "/home/dev/.") || strings.Contains(path, "/root/."):
			config = append(config, line)
		case strings.HasPrefix(path, "/var/") || strings.HasPrefix(path, "/etc/") || strings.HasPrefix(path, "/usr/"):
			system = append(system, line)
		default:
			other = append(other, line)
		}
	}

	// Print categorized output
	printCategory := func(name string, items []string, showAll bool) {
		if len(items) == 0 {
			return
		}
		sort.Strings(items)
		colorBold.Printf("\n%s (%d):\n", name, len(items))
		if showAll {
			for _, item := range items {
				fmt.Printf("  %s\n", item)
			}
		} else {
			limit := 10
			if len(items) < limit {
				limit = len(items)
			}
			for _, item := range items[:limit] {
				fmt.Printf("  %s\n", item)
			}
			if len(items) > 10 {
				colorDim.Printf("  ... and %d more\n", len(items)-10)
			}
		}
	}

	fmt.Printf("Container: %s\n", containerName)
	fmt.Printf("Total changes: %d\n", len(allChanges))

	// Show meaningful changes first
	printCategory("Homebrew", brew, false)
	printCategory("Config files", config, true)
	printCategory("System", system, false)
	printCategory("Other", other, true)

	// Show filtered categories last
	if noiseCount > 0 {
		colorDim.Printf("\nFiltered as noise (%d): ", noiseCount)
		colorDim.Printf("use --raw to see all\n")
	}
	if workspaceCount > 0 {
		colorDim.Printf("Workspace mount (%d): ", workspaceCount)
		colorDim.Printf("changes are on host filesystem\n")
	}

	return nil
}
