package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joelhelbling/glovebox/internal/mod"
	"github.com/joelhelbling/glovebox/internal/ui"
	"github.com/spf13/cobra"
)

var modGlobal bool

var modCmd = &cobra.Command{
	Use:   "mod",
	Short: "Manage custom mods",
	Long: `Manage custom mods for your glovebox environment.

Mods are YAML files that define tools, packages, and configurations
to include in your Docker image. Custom mods can be created in:

  ~/.glovebox/mods/       Global mods (available everywhere)
  .glovebox/mods/         Project-local mods (this project only)

Local mods take precedence over embedded ones, so you can also
override built-in mods if needed.`,
}

var modCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new custom mod",
	Long: `Create a new custom mod with a starter template.

The mod name can include a category prefix (e.g., "tools/mytool").
Without --global, creates in .glovebox/mods/ (project-local).
With --global, creates in ~/.glovebox/mods/ (available everywhere).

Examples:
  glovebox mod create my-tool           # Creates custom/my-tool.yaml
  glovebox mod create tools/my-tool     # Creates tools/my-tool.yaml
  glovebox mod create my-tool --global  # Creates in ~/.glovebox/mods/`,
	Args: cobra.ExactArgs(1),
	RunE: runModCreate,
}

var modCatCmd = &cobra.Command{
	Use:   "cat <mod-id>",
	Short: "Output a mod's raw YAML content",
	Long: `Output the raw YAML content of a mod to stdout.

This is useful for inspecting mods or creating custom overrides:

  # View a mod
  glovebox mod cat ai/claude-code

  # Copy to local mods and customize
  glovebox mod cat ai/claude-code > .glovebox/mods/ai/claude-code.yaml

The command respects the mod load order (local > global > embedded),
so it shows the version that would actually be used.`,
	Args: cobra.ExactArgs(1),
	RunE: runModCat,
}

var modListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List available mods",
	Long: `List all available mods that can be added to your glovebox profile.

This shows built-in mods plus any custom mods found in:
  ~/.glovebox/mods/       Global custom mods
  .glovebox/mods/         Project-local custom mods

To create a custom mod, run:
  glovebox mod create <name>`,
	RunE: runModList,
}

func init() {
	modCreateCmd.Flags().BoolVarP(&modGlobal, "global", "g", false, "Create in global mods directory")
	modCmd.AddCommand(modCreateCmd)
	modCmd.AddCommand(modCatCmd)
	modCmd.AddCommand(modListCmd)
	rootCmd.AddCommand(modCmd)
}

func runModCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Determine the mod path
	var modDir string
	if modGlobal {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home directory: %w", err)
		}
		modDir = filepath.Join(home, ".glovebox", "mods")
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
		modDir = filepath.Join(cwd, ".glovebox", "mods")
	}

	// Parse name to extract category
	var category, modName string
	if strings.Contains(name, "/") {
		parts := strings.SplitN(name, "/", 2)
		category = parts[0]
		modName = parts[1]
	} else {
		category = "custom"
		modName = name
	}

	// Build full path
	modPath := filepath.Join(modDir, category, modName+".yaml")

	// Check if file already exists
	if _, err := os.Stat(modPath); err == nil {
		fmt.Printf("Mod already exists at %s\n", modPath)
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
	if err := os.MkdirAll(filepath.Dir(modPath), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Generate template content
	template := fmt.Sprintf(`name: %s
description: TODO - describe what this mod provides
category: %s

# Dependencies on other mods (optional)
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
`, modName, category)

	// Write the file
	if err := os.WriteFile(modPath, []byte(template), 0644); err != nil {
		return fmt.Errorf("writing mod: %w", err)
	}

	colorGreen.Printf("✓ Created mod at %s\n", modPath)

	modID := category + "/" + modName
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Edit %s to add your configuration\n", modPath)
	fmt.Printf("  2. glovebox add %s\n", modID)
	if modGlobal {
		fmt.Println("  3. glovebox build --base")
	} else {
		fmt.Println("  3. glovebox build")
	}

	return nil
}

func runModCat(cmd *cobra.Command, args []string) error {
	id := args[0]

	data, _, err := mod.LoadRaw(id)
	if err != nil {
		return err
	}

	// Write raw YAML to stdout (no trailing newline if content already has one)
	_, err = os.Stdout.Write(data)
	return err
}

func runModList(cmd *cobra.Command, args []string) error {
	modsByCategory, err := mod.ListAll()
	if err != nil {
		return fmt.Errorf("listing mods: %w", err)
	}

	if len(modsByCategory) == 0 {
		fmt.Println("No mods found.")
		return nil
	}

	// Sort categories in a logical order (not alphabetical)
	categoryOrder := []string{"core", "shells", "tools", "editors", "languages", "ai"}
	categoryRank := make(map[string]int)
	for i, cat := range categoryOrder {
		categoryRank[cat] = i
	}

	var categoryNames []string
	for cat := range modsByCategory {
		categoryNames = append(categoryNames, cat)
	}
	sort.Slice(categoryNames, func(i, j int) bool {
		rankI, knownI := categoryRank[categoryNames[i]]
		rankJ, knownJ := categoryRank[categoryNames[j]]
		// Known categories sorted by rank, unknown categories at the end alphabetically
		if knownI && knownJ {
			return rankI < rankJ
		}
		if knownI {
			return true
		}
		if knownJ {
			return false
		}
		return categoryNames[i] < categoryNames[j]
	})

	// Build UI categories
	var categories []ui.ModCategory
	for _, categoryName := range categoryNames {
		modIDs := modsByCategory[categoryName]
		sort.Strings(modIDs)

		category := ui.ModCategory{Name: categoryName}
		for _, id := range modIDs {
			// Extract just the mod name (after the category prefix, if any)
			modName := id
			if strings.Contains(id, "/") {
				parts := strings.SplitN(id, "/", 2)
				modName = parts[1]
			}

			m, err := mod.Load(id)
			if err != nil {
				category.Mods = append(category.Mods, ui.ModInfo{
					Name:  modName,
					Error: true,
				})
				continue
			}
			category.Mods = append(category.Mods, ui.ModInfo{
				Name:        modName,
				Description: m.Description,
			})
		}
		categories = append(categories, category)
	}

	// Render
	modList := ui.NewModList()
	modList.Print(categories)

	return nil
}
