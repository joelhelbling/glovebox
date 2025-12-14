package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PostExitInfo contains information for the post-exit prompt
type PostExitInfo struct {
	Changes []string
}

// PostExitResult represents what the user chose
type PostExitResult struct {
	Committed bool
	Erased    bool
	Kept      bool
	ImageName string
	Error     error
}

// Prompt renders interactive prompts
type Prompt struct {
	term *Terminal
}

// NewPrompt creates a new Prompt renderer
func NewPrompt() *Prompt {
	return &Prompt{
		term: NewTerminal(),
	}
}

// RenderPostExitPrompt produces the formatted post-exit prompt
func (p *Prompt) RenderPostExitPrompt(changes []string) string {
	var sb strings.Builder

	bar := p.term.VerticalBar()

	// Define styles
	var (
		barStyle    lipgloss.Style
		titleStyle  lipgloss.Style
		changeStyle lipgloss.Style
		optionStyle lipgloss.Style
		keyStyle    lipgloss.Style
		descStyle   lipgloss.Style
	)

	if p.term.HasColors() {
		barStyle = p.term.NewStyle().Foreground(lipgloss.Color("240"))
		titleStyle = p.term.NewStyle().Bold(true).Foreground(lipgloss.Color("3")) // yellow
		changeStyle = p.term.NewStyle().Foreground(lipgloss.Color("240"))
		optionStyle = p.term.NewStyle()
		keyStyle = p.term.NewStyle().Bold(true)
		descStyle = p.term.NewStyle().Foreground(lipgloss.Color("240"))
	} else {
		barStyle = p.term.NewStyle()
		titleStyle = p.term.NewStyle()
		changeStyle = p.term.NewStyle()
		optionStyle = p.term.NewStyle()
		keyStyle = p.term.NewStyle()
		descStyle = p.term.NewStyle()
	}

	// Helper to render a line with the bar prefix
	line := func(content string) {
		sb.WriteString(fmt.Sprintf("  %s %s\n", barStyle.Render(bar), content))
	}

	// Title
	sb.WriteString("\n")
	line(titleStyle.Render("Changes detected in container"))
	line("")

	// List changes
	for _, change := range changes {
		line(changeStyle.Render("  " + change))
	}

	line("")
	line(optionStyle.Render("What would you like to do?"))
	line("")

	// Options with highlighted keys
	renderOption := func(key, rest, desc string) string {
		return fmt.Sprintf("  %s%s  %s",
			keyStyle.Render("["+key+"]"),
			optionStyle.Render(rest),
			descStyle.Render(desc))
	}

	line(renderOption("y", "es", "commit changes to image (fresh container next run)"))
	line(renderOption("n", "o", "keep uncommitted changes (resume this container next run)"))
	line(renderOption("e", "rase", "discard changes (fresh container next run)"))
	line("")

	return sb.String()
}

// PrintPostExitPrompt renders and prints the post-exit prompt
func (p *Prompt) PrintPostExitPrompt(changes []string) {
	fmt.Print(p.RenderPostExitPrompt(changes))
}

// RenderCommitSuccess renders the success message after committing
func (p *Prompt) RenderCommitSuccess(imageName string) string {
	bar := p.term.VerticalBar()

	var barStyle, msgStyle lipgloss.Style
	if p.term.HasColors() {
		barStyle = p.term.NewStyle().Foreground(lipgloss.Color("240"))
		msgStyle = p.term.NewStyle().Foreground(lipgloss.Color("2")) // green
	} else {
		barStyle = p.term.NewStyle()
		msgStyle = p.term.NewStyle()
	}

	return fmt.Sprintf("  %s %s\n",
		barStyle.Render(bar),
		msgStyle.Render(fmt.Sprintf("✓ Changes committed to %s", imageName)))
}

// RenderEraseSuccess renders the success message after erasing
func (p *Prompt) RenderEraseSuccess() string {
	bar := p.term.VerticalBar()

	var barStyle, msgStyle lipgloss.Style
	if p.term.HasColors() {
		barStyle = p.term.NewStyle().Foreground(lipgloss.Color("240"))
		msgStyle = p.term.NewStyle().Foreground(lipgloss.Color("240"))
	} else {
		barStyle = p.term.NewStyle()
		msgStyle = p.term.NewStyle()
	}

	return fmt.Sprintf("  %s %s\n",
		barStyle.Render(bar),
		msgStyle.Render("Container removed. Next run will start fresh."))
}

// RenderKeepSuccess renders the message when keeping changes
func (p *Prompt) RenderKeepSuccess() string {
	bar := p.term.VerticalBar()

	var barStyle, msgStyle lipgloss.Style
	if p.term.HasColors() {
		barStyle = p.term.NewStyle().Foreground(lipgloss.Color("240"))
		msgStyle = p.term.NewStyle().Foreground(lipgloss.Color("240"))
	} else {
		barStyle = p.term.NewStyle()
		msgStyle = p.term.NewStyle()
	}

	return fmt.Sprintf("  %s %s\n",
		barStyle.Render(bar),
		msgStyle.Render("Changes kept in container."))
}

// RenderWarning renders a warning message
func (p *Prompt) RenderWarning(msg string) string {
	bar := p.term.VerticalBar()

	var barStyle, msgStyle lipgloss.Style
	if p.term.HasColors() {
		barStyle = p.term.NewStyle().Foreground(lipgloss.Color("240"))
		msgStyle = p.term.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	} else {
		barStyle = p.term.NewStyle()
		msgStyle = p.term.NewStyle()
	}

	return fmt.Sprintf("  %s %s\n",
		barStyle.Render(bar),
		msgStyle.Render("⚠ "+msg))
}

// RenderChoicePrompt renders just the "Choice: " prompt
func (p *Prompt) RenderChoicePrompt() string {
	bar := p.term.VerticalBar()

	var barStyle lipgloss.Style
	if p.term.HasColors() {
		barStyle = p.term.NewStyle().Foreground(lipgloss.Color("240"))
	} else {
		barStyle = p.term.NewStyle()
	}

	return fmt.Sprintf("  %s Choice: ", barStyle.Render(bar))
}

// RenderExitSummary renders the exit summary with changes (no prompt)
func (p *Prompt) RenderExitSummary(changes []string) string {
	var sb strings.Builder

	bar := p.term.VerticalBar()

	var (
		barStyle   lipgloss.Style
		titleStyle lipgloss.Style
		dimStyle   lipgloss.Style
		cmdStyle   lipgloss.Style
	)

	if p.term.HasColors() {
		barStyle = p.term.NewStyle().Foreground(lipgloss.Color("240"))
		titleStyle = p.term.NewStyle().Bold(true)
		dimStyle = p.term.NewStyle().Foreground(lipgloss.Color("240"))
		cmdStyle = p.term.NewStyle().Foreground(lipgloss.Color("6")) // cyan
	} else {
		barStyle = p.term.NewStyle()
		titleStyle = p.term.NewStyle()
		dimStyle = p.term.NewStyle()
		cmdStyle = p.term.NewStyle()
	}

	line := func(content string) {
		sb.WriteString(fmt.Sprintf("  %s %s\n", barStyle.Render(bar), content))
	}

	sb.WriteString("\n")

	if len(changes) > 0 {
		line(titleStyle.Render("Session ended") + dimStyle.Render(" · container has uncommitted changes:"))
		line("")
		for _, change := range changes {
			line(dimStyle.Render("  " + change))
		}
		line("")
		line(dimStyle.Render("To persist: ") + cmdStyle.Render("glovebox commit"))
		line(dimStyle.Render("To discard: ") + cmdStyle.Render("glovebox reset"))
	} else {
		line(titleStyle.Render("Session ended"))
	}

	sb.WriteString("\n")

	return sb.String()
}

// PrintExitSummary renders and prints the exit summary
func (p *Prompt) PrintExitSummary(changes []string) {
	fmt.Print(p.RenderExitSummary(changes))
}
