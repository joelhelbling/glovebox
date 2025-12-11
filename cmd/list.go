package cmd

import (
	"fmt"
	"sort"

	"github.com/fatih/color"
	"github.com/joelhelbling/glovebox/internal/snippet"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available snippets",
	Long: `List all available snippets that can be added to your glovebox profile.

This shows built-in snippets plus any custom snippets found in:
  ~/.glovebox/snippets/       Global custom snippets
  .glovebox/snippets/         Project-local custom snippets

To create a custom snippet, run:
  glovebox snippet create <name>`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
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
