package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/joelhelbling/glovebox/internal/profile"
	"github.com/joelhelbling/glovebox/internal/snippet"
	"github.com/spf13/cobra"
)

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
			return err
		}
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
		profilePath = profile.ProjectPath(cwd)
	}

	// Check if profile already exists
	if _, err := os.Stat(profilePath); err == nil {
		fmt.Printf("Profile already exists at %s\n", profilePath)
		fmt.Print("Overwrite? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Interactive snippet selection
	selectedSnippets, err := interactiveSnippetSelection()
	if err != nil {
		return err
	}

	if len(selectedSnippets) == 0 {
		fmt.Println("No snippets selected. Aborted.")
		return nil
	}

	// Create and save profile
	p := profile.NewProfile()
	p.Snippets = selectedSnippets

	if err := p.SaveTo(profilePath); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	green := color.New(color.FgGreen)
	green.Printf("✓ Profile created at %s\n", profilePath)
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

func interactiveSnippetSelection() ([]string, error) {
	snippetsByCategory, err := snippet.ListAll()
	if err != nil {
		return nil, fmt.Errorf("listing snippets: %w", err)
	}

	// Always include base
	selected := []string{"base"}

	reader := bufio.NewReader(os.Stdin)
	bold := color.New(color.Bold)
	dim := color.New(color.Faint)

	// Sort categories for consistent ordering
	var categories []string
	for cat := range snippetsByCategory {
		if cat == "core" {
			continue // Skip core (base is auto-included)
		}
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	fmt.Println("\nSelect snippets for your glovebox environment.")
	fmt.Println("Base dependencies are automatically included.\n")

	for _, category := range categories {
		snippets := snippetsByCategory[category]
		sort.Strings(snippets)

		bold.Printf("%s:\n", strings.Title(category))

		// Display options
		for i, id := range snippets {
			s, err := snippet.Load(id)
			desc := ""
			if err == nil {
				desc = s.Description
			}
			fmt.Printf("  %d) %-20s", i+1, id)
			dim.Printf(" %s\n", desc)
		}

		// Prompt for selection
		fmt.Printf("Select %s (comma-separated numbers, or 'none'): ", category)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "" || input == "none" || input == "n" {
			fmt.Println()
			continue
		}

		// Parse selections
		parts := strings.Split(input, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			num, err := strconv.Atoi(part)
			if err != nil || num < 1 || num > len(snippets) {
				fmt.Printf("  Invalid selection: %s (skipped)\n", part)
				continue
			}
			selected = append(selected, snippets[num-1])
		}
		fmt.Println()
	}

	return selected, nil
}
