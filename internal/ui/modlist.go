package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ModInfo represents a mod for display
type ModInfo struct {
	Name        string   // just the mod name without category prefix
	Description string   // human-readable description
	Provides    []string // what this mod provides (for display)
	RequiresOS  string   // OS required by this mod (empty if OS-agnostic)
	Error       bool     // true if there was an error loading this mod
}

// ModCategory represents a category of mods
type ModCategory struct {
	Name string // category name (e.g., "ai", "editors")
	Mods []ModInfo
}

// ModList renders the mod list output
type ModList struct {
	term *Terminal
}

// NewModList creates a new ModList renderer
func NewModList() *ModList {
	return &ModList{
		term: NewTerminal(),
	}
}

// Render produces the formatted mod list string
func (m *ModList) Render(categories []ModCategory) string {
	var sb strings.Builder

	bar := m.term.VerticalBar()

	// Define styles
	var (
		barStyle      lipgloss.Style
		categoryStyle lipgloss.Style
		nameStyle     lipgloss.Style
		descStyle     lipgloss.Style
		errStyle      lipgloss.Style
	)

	if m.term.HasColors() {
		barStyle = m.term.NewStyle().Foreground(lipgloss.Color("240"))
		categoryStyle = m.term.NewStyle().Bold(true)
		nameStyle = m.term.NewStyle()
		descStyle = m.term.NewStyle().Foreground(lipgloss.Color("240"))
		errStyle = m.term.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	} else {
		barStyle = m.term.NewStyle()
		categoryStyle = m.term.NewStyle()
		nameStyle = m.term.NewStyle()
		descStyle = m.term.NewStyle()
		errStyle = m.term.NewStyle()
	}

	// Helper to render a line with the bar prefix
	line := func(content string) {
		sb.WriteString(fmt.Sprintf("  %s %s\n", barStyle.Render(bar), content))
	}

	sb.WriteString("\n")
	line(categoryStyle.Render("Available Mods"))
	line("")

	// Find the longest mod name across all categories for alignment
	maxNameLen := 0
	for _, category := range categories {
		for _, mod := range category.Mods {
			if len(mod.Name) > maxNameLen {
				maxNameLen = len(mod.Name)
			}
		}
	}

	for i, category := range categories {
		if i > 0 {
			line("")
		}

		// Category header with trailing slash
		line(categoryStyle.Render(category.Name + "/"))

		// Mods in this category (indented, with aligned descriptions)
		for _, mod := range category.Mods {
			paddedName := fmt.Sprintf("%-*s", maxNameLen, mod.Name)
			if mod.Error {
				line("  " + nameStyle.Render(paddedName) + "  " + errStyle.Render("(error loading)"))
			} else {
				// Build the line content
				content := "  " + nameStyle.Render(paddedName)
				if mod.Description != "" {
					content += "  " + descStyle.Render(mod.Description)
				}
				// Add OS requirement indicator if present
				if mod.RequiresOS != "" {
					content += " " + descStyle.Render("["+mod.RequiresOS+"]")
				}
				line(content)
			}
		}
	}

	sb.WriteString("\n")
	return sb.String()
}

// Print renders and prints the mod list to stdout
func (m *ModList) Print(categories []ModCategory) {
	fmt.Print(m.Render(categories))
}
