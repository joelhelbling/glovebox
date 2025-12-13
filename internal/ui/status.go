package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StatusSection represents a section in the status output
type StatusSection struct {
	Title string
	Items []StatusItem
}

// StatusItem represents a single item in a status section
type StatusItem struct {
	Label  string
	Value  string
	Status ItemStatus
	Indent int  // 0 = normal, 1 = sub-item, 2 = sub-sub-item
	IsList bool // true for list items (mods, etc.)
	Note   string
}

// ItemStatus indicates the state of a status item
type ItemStatus int

const (
	StatusNone ItemStatus = iota
	StatusOK
	StatusWarning
	StatusInfo
)

// Status renders the glovebox status output
type Status struct {
	term *Terminal
}

// NewStatus creates a new Status renderer
func NewStatus() *Status {
	return &Status{
		term: NewTerminal(),
	}
}

// Render produces the formatted status string for multiple sections
func (s *Status) Render(sections []StatusSection) string {
	var sb strings.Builder

	for i, section := range sections {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(s.renderSection(section))
	}

	return sb.String()
}

func (s *Status) renderSection(section StatusSection) string {
	var sb strings.Builder

	bar := s.term.VerticalBar()

	// Define styles
	var (
		barStyle   lipgloss.Style
		titleStyle lipgloss.Style
		labelStyle lipgloss.Style
		valueStyle lipgloss.Style
		okStyle    lipgloss.Style
		warnStyle  lipgloss.Style
		dimStyle   lipgloss.Style
		noteStyle  lipgloss.Style
	)

	if s.term.HasColors() {
		barStyle = s.term.NewStyle().Foreground(lipgloss.Color("240"))
		titleStyle = s.term.NewStyle().Bold(true)
		labelStyle = s.term.NewStyle().Foreground(lipgloss.Color("240"))
		valueStyle = s.term.NewStyle()
		okStyle = s.term.NewStyle().Foreground(lipgloss.Color("2"))   // green
		warnStyle = s.term.NewStyle().Foreground(lipgloss.Color("3")) // yellow
		dimStyle = s.term.NewStyle().Foreground(lipgloss.Color("240"))
		noteStyle = s.term.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	} else {
		barStyle = s.term.NewStyle()
		titleStyle = s.term.NewStyle()
		labelStyle = s.term.NewStyle()
		valueStyle = s.term.NewStyle()
		okStyle = s.term.NewStyle()
		warnStyle = s.term.NewStyle()
		dimStyle = s.term.NewStyle()
		noteStyle = s.term.NewStyle()
	}

	// Helper to render a line with the bar prefix
	line := func(content string) {
		sb.WriteString(fmt.Sprintf("  %s %s\n", barStyle.Render(bar), content))
	}

	// Title line
	sb.WriteString("\n")
	line(titleStyle.Render(section.Title))
	line("")

	// Render items
	for _, item := range section.Items {
		var content string

		// Calculate indent
		indent := strings.Repeat("  ", item.Indent)

		if item.IsList {
			// List item (like mods)
			content = indent + dimStyle.Render("- "+item.Value)
		} else if item.Label != "" {
			// Label-value pair
			paddedLabel := fmt.Sprintf("%-12s", item.Label)
			content = indent + labelStyle.Render(paddedLabel) + valueStyle.Render(item.Value)

			// Add status indicator
			switch item.Status {
			case StatusOK:
				content += " " + okStyle.Render("✓")
			case StatusWarning:
				content += " " + warnStyle.Render("⚠")
			}
		} else {
			// Just a value (like status messages)
			switch item.Status {
			case StatusOK:
				content = indent + okStyle.Render(item.Value)
			case StatusWarning:
				content = indent + warnStyle.Render(item.Value)
			case StatusInfo:
				content = indent + dimStyle.Render(item.Value)
			default:
				content = indent + item.Value
			}
		}

		line(content)

		// Add note if present
		if item.Note != "" {
			noteIndent := strings.Repeat("  ", item.Indent+1)
			line(noteIndent + noteStyle.Render(item.Note))
		}
	}

	return sb.String()
}

// Print renders and prints the status to stdout
func (s *Status) Print(sections []StatusSection) {
	fmt.Print(s.Render(sections))
}
