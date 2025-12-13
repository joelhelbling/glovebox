package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderLogo returns the glovebox ASCII logo with tagline, styled to match status output
func (s *Status) RenderLogo() string {
	var sb strings.Builder

	tagline := "Now get in there and do some science!"
	bar := s.term.VerticalBar()

	var barStyle, taglineStyle lipgloss.Style
	if s.term.HasColors() {
		barStyle = s.term.NewStyle().Foreground(lipgloss.Color("240"))
		taglineStyle = s.term.NewStyle().Italic(true)
		tagline = taglineStyle.Render(tagline)
	} else {
		barStyle = s.term.NewStyle()
	}

	line := func(content string) {
		sb.WriteString(fmt.Sprintf("  %s %s\n", barStyle.Render(bar), content))
	}

	sb.WriteString("\n")
	line("╔═══════╗")
	line("║ ✋ 🤚 ║  " + tagline)
	line("╚═══════╝")

	return sb.String()
}
