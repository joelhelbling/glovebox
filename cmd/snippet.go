package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/joelhelbling/glovebox/internal/snippet"
	"github.com/spf13/cobra"
)

var snippetGlobal bool

var snippetCmd = &cobra.Command{
	Use:   "snippet",
	Short: "Manage custom snippets",
	Long: `Manage custom snippets for your glovebox environment.

Snippets are YAML files that define tools, packages, and configurations
to include in your Docker image. Custom snippets can be created in:

  ~/.glovebox/snippets/       Global snippets (available everywhere)
  .glovebox/snippets/         Project-local snippets (this project only)

Local snippets take precedence over embedded ones, so you can also
override built-in snippets if needed.`,
}

var snippetCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new custom snippet",
	Long: `Create a new custom snippet with a starter template.

The snippet name can include a category prefix (e.g., "tools/mytool").
Without --global, creates in .glovebox/snippets/ (project-local).
With --global, creates in ~/.glovebox/snippets/ (available everywhere).

Examples:
  glovebox snippet create my-tool           # Creates custom/my-tool.yaml
  glovebox snippet create tools/my-tool     # Creates tools/my-tool.yaml
  glovebox snippet create my-tool --global  # Creates in ~/.glovebox/snippets/`,
	Args: cobra.ExactArgs(1),
	RunE: runSnippetCreate,
}

var snippetCatCmd = &cobra.Command{
	Use:   "cat <snippet-id>",
	Short: "Output a snippet's raw YAML content",
	Long: `Output the raw YAML content of a snippet to stdout.

This is useful for inspecting snippets or creating custom overrides:

  # View a snippet
  glovebox snippet cat ai/claude-code

  # Copy to local snippets and customize
  glovebox snippet cat ai/claude-code > .glovebox/snippets/ai/claude-code.yaml

The command respects the snippet load order (local > global > embedded),
so it shows the version that would actually be used.`,
	Args: cobra.ExactArgs(1),
	RunE: runSnippetCat,
}

var snippetListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List available snippets",
	Long: `List all available snippets that can be added to your glovebox profile.

This shows built-in snippets plus any custom snippets found in:
  ~/.glovebox/snippets/       Global custom snippets
  .glovebox/snippets/         Project-local custom snippets

To create a custom snippet, run:
  glovebox snippet create <name>`,
	RunE: runSnippetList,
}

func init() {
	snippetCreateCmd.Flags().BoolVarP(&snippetGlobal, "global", "g", false, "Create in global snippets directory")
	snippetCmd.AddCommand(snippetCreateCmd)
	snippetCmd.AddCommand(snippetCatCmd)
	snippetCmd.AddCommand(snippetListCmd)
	rootCmd.AddCommand(snippetCmd)
}

func runSnippetCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Determine the snippet path
	var snippetDir string
	if snippetGlobal {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home directory: %w", err)
		}
		snippetDir = filepath.Join(home, ".glovebox", "snippets")
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
		snippetDir = filepath.Join(cwd, ".glovebox", "snippets")
	}

	// Parse name to extract category
	var category, snippetName string
	if strings.Contains(name, "/") {
		parts := strings.SplitN(name, "/", 2)
		category = parts[0]
		snippetName = parts[1]
	} else {
		category = "custom"
		snippetName = name
	}

	// Build full path
	snippetPath := filepath.Join(snippetDir, category, snippetName+".yaml")

	// Check if file already exists
	if _, err := os.Stat(snippetPath); err == nil {
		fmt.Printf("Snippet already exists at %s\n", snippetPath)
		fmt.Print("Overwrite? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(snippetPath), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Generate template content
	template := fmt.Sprintf(`name: %s
description: TODO - describe what this snippet provides
category: %s

# Dependencies on other snippets (optional)
# requires:
#   - base

# APT repositories to add (optional)
# apt_repos:
#   - ppa:some/repo

# APT packages to install (optional)
# apt_packages:
#   - some-package

# Commands to run as root (optional)
# run_as_root: |
#   curl -fsSL https://example.com/install.sh | bash

# Commands to run as ubuntu user (optional)
# run_as_user: |
#   echo "setup complete"

# Environment variables to set (optional)
# env:
#   MY_VAR: value

# Set as default shell (optional, use full path)
# user_shell: /usr/bin/bash
`, snippetName, category)

	// Write the file
	if err := os.WriteFile(snippetPath, []byte(template), 0644); err != nil {
		return fmt.Errorf("writing snippet: %w", err)
	}

	green := color.New(color.FgGreen)
	green.Printf("✓ Created snippet at %s\n", snippetPath)

	snippetID := category + "/" + snippetName
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Edit %s to add your configuration\n", snippetPath)
	fmt.Printf("  2. glovebox add %s\n", snippetID)
	if snippetGlobal {
		fmt.Println("  3. glovebox build --base")
	} else {
		fmt.Println("  3. glovebox build")
	}

	return nil
}

func runSnippetCat(cmd *cobra.Command, args []string) error {
	id := args[0]

	data, _, err := snippet.LoadRaw(id)
	if err != nil {
		return err
	}

	// Write raw YAML to stdout (no trailing newline if content already has one)
	_, err = os.Stdout.Write(data)
	return err
}

func runSnippetList(cmd *cobra.Command, args []string) error {
	snippetsByCategory, err := snippet.ListAll()
	if err != nil {
		return fmt.Errorf("listing snippets: %w", err)
	}

	if len(snippetsByCategory) == 0 {
		fmt.Println("No snippets found.")
		return nil
	}

	// Sort categories for consistent output
	var categories []string
	for cat := range snippetsByCategory {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	bold := color.New(color.Bold)
	dim := color.New(color.Faint)

	for _, category := range categories {
		snippets := snippetsByCategory[category]
		sort.Strings(snippets)

		bold.Printf("\n%s:\n", category)
		for _, id := range snippets {
			s, err := snippet.Load(id)
			if err != nil {
				fmt.Printf("  %s (error loading)\n", id)
				continue
			}
			fmt.Printf("  %-20s", id)
			dim.Printf(" %s\n", s.Description)
		}
	}
	fmt.Println()

	return nil
}
