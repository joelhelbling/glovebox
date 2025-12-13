package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/joelhelbling/glovebox/internal/mod"
	"github.com/joelhelbling/glovebox/internal/profile"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// osDescriptions provides human-friendly descriptions for OS options
var osDescriptions = map[string]string{
	"ubuntu": "Ubuntu 24.04 LTS - Best compatibility, most packages",
	"fedora": "Fedora 41 - Latest packages, good for development",
	"alpine": "Alpine Linux - Minimal, fast, small images",
}

var (
	initBase bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new glovebox profile",
	Long: `Initialize a new glovebox profile interactively.

Use --base to create the base image profile (~/.glovebox/profile.yaml).
This defines your standard development environment with your preferred
shell, editor, and tools. Build it once with 'glovebox build --base'.

Without --base, creates a project-specific profile (.glovebox/profile.yaml)
that extends the base image with additional tools for that project.`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&initBase, "base", "b", false, "Create base profile instead of project-local")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	// Determine profile path
	var profilePath string
	if initBase {
		var err error
		profilePath, err = profile.GlobalPath()
		if err != nil {
			return fmt.Errorf("getting global profile path: %w", err)
		}
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
		profilePath = profile.ProjectPath(cwd)
	}

	// Check if profile already exists
	if existingProfile, err := profile.Load(profilePath); err == nil && existingProfile != nil {
		reader := bufio.NewReader(os.Stdin)

		if existingProfile.WasManuallyEdited() {
			// Profile was manually edited - stronger warning
			colorYellow.Println("⚠ Warning: This profile has been manually edited!")
			fmt.Printf("Profile at %s contains changes made outside of glovebox init.\n", profilePath)
			fmt.Println("Overwriting will lose those customizations.")
			fmt.Print("\nOverwrite anyway? [y/N]: ")
		} else {
			fmt.Printf("Profile already exists at %s\n", profilePath)
			fmt.Print("Overwrite? [y/N]: ")
		}

		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Interactive mod selection
	selectedMods, err := interactiveModSelection()
	if err != nil {
		return fmt.Errorf("selecting mods: %w", err)
	}

	if len(selectedMods) == 0 {
		fmt.Println("No mods selected. Aborted.")
		return nil
	}

	// Create and save profile
	p := profile.NewProfile()
	p.Mods = selectedMods
	p.UpdateContentHash() // Store hash to detect future manual edits

	if err := p.SaveTo(profilePath); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	colorGreen.Printf("✓ Profile created at %s\n", profilePath)
	fmt.Println("\nNext steps:")
	if initBase {
		fmt.Println("  glovebox build --base   # Build the base image (glovebox:base)")
		fmt.Println("  glovebox run            # Run glovebox in any directory")
	} else {
		fmt.Println("  glovebox build          # Build the project image")
		fmt.Println("  glovebox run            # Run glovebox in this directory")
	}

	return nil
}

func interactiveModSelection() ([]string, error) {
	reader := bufio.NewReader(os.Stdin)

	// Step 1: Select OS
	selectedOS, err := selectOS(reader)
	if err != nil {
		return nil, err
	}

	// Start with the OS mod
	selected := []string{"os/" + selectedOS}

	// Step 2: Select other mods
	modsByCategory, err := mod.ListAll()
	if err != nil {
		return nil, fmt.Errorf("listing mods: %w", err)
	}

	// Sort categories for consistent ordering, with preferred order first
	categoryOrder := []string{"shells", "editors", "tools", "languages", "ai"}
	categoryRank := make(map[string]int)
	for i, cat := range categoryOrder {
		categoryRank[cat] = i
	}

	var categories []string
	for cat := range modsByCategory {
		// Skip os category (already selected) and core category
		if cat == "os" || cat == "core" {
			continue
		}
		categories = append(categories, cat)
	}
	sort.Slice(categories, func(i, j int) bool {
		rankI, knownI := categoryRank[categories[i]]
		rankJ, knownJ := categoryRank[categories[j]]
		if knownI && knownJ {
			return rankI < rankJ
		}
		if knownI {
			return true
		}
		if knownJ {
			return false
		}
		return categories[i] < categories[j]
	})

	fmt.Println("\nSelect additional mods for your glovebox environment.")
	fmt.Printf("OS: %s (dependencies will be resolved automatically)\n", selectedOS)

	for _, category := range categories {
		allMods := modsByCategory[category]

		// Filter mods compatible with selected OS
		compatibleMods := filterCompatibleMods(allMods, selectedOS)
		if len(compatibleMods) == 0 {
			continue
		}
		sort.Strings(compatibleMods)

		colorBold.Printf("\n%s:\n", cases.Title(language.English).String(category))

		// Display options
		for i, id := range compatibleMods {
			m, err := mod.Load(id)
			desc := ""
			if err == nil {
				desc = m.Description
			}
			// Show simplified name (strip OS suffix if present)
			displayName := simplifyModName(id, selectedOS)
			fmt.Printf("  %d) %-20s", i+1, displayName)
			colorDim.Printf(" %s\n", desc)
		}

		// Prompt for selection
		fmt.Printf("Select %s (comma-separated numbers, or 'none'): ", category)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "" || input == "none" || input == "n" {
			continue
		}

		// Parse selections
		parts := strings.Split(input, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			num, err := strconv.Atoi(part)
			if err != nil || num < 1 || num > len(compatibleMods) {
				fmt.Printf("  Invalid selection: %s (skipped)\n", part)
				continue
			}
			selected = append(selected, compatibleMods[num-1])
		}
	}

	return selected, nil
}

// selectOS prompts the user to select an operating system
func selectOS(reader *bufio.Reader) (string, error) {
	fmt.Println("\nSelect your base operating system:")

	// Display OS options with descriptions
	for i, osName := range mod.KnownOSNames {
		desc := osDescriptions[osName]
		fmt.Printf("  %d) %-10s", i+1, osName)
		colorDim.Printf(" %s\n", desc)
	}

	// Default to ubuntu
	fmt.Print("\nSelect OS [1]: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return mod.KnownOSNames[0], nil // Default to ubuntu
	}

	num, err := strconv.Atoi(input)
	if err != nil || num < 1 || num > len(mod.KnownOSNames) {
		return "", fmt.Errorf("invalid OS selection: %s", input)
	}

	return mod.KnownOSNames[num-1], nil
}

// filterCompatibleMods returns only mods that are compatible with the selected OS.
// A mod is compatible if:
// 1. It doesn't require any OS (OS-agnostic)
// 2. It requires the selected OS
// Mods that require a different OS are filtered out.
func filterCompatibleMods(modIDs []string, selectedOS string) []string {
	var compatible []string

	for _, id := range modIDs {
		m, err := mod.Load(id)
		if err != nil {
			continue
		}

		// Check if mod requires a different OS
		requiresDifferentOS := false
		for _, req := range m.Requires {
			if isOSName(req) && req != selectedOS {
				requiresDifferentOS = true
				break
			}
		}

		if !requiresDifferentOS {
			compatible = append(compatible, id)
		}
	}

	return compatible
}

// isOSName checks if a name is a known OS name
func isOSName(name string) bool {
	for _, os := range mod.KnownOSNames {
		if name == os {
			return true
		}
	}
	return false
}

// simplifyModName returns a display-friendly name for a mod.
// For OS-specific mods like "shells/zsh-ubuntu", it shows "zsh-ubuntu".
// For generic mods like "tools/homebrew", it shows "homebrew".
func simplifyModName(modID string, selectedOS string) string {
	// Extract just the mod name from category/name
	parts := strings.Split(modID, "/")
	if len(parts) > 1 {
		return parts[1]
	}
	return modID
}
