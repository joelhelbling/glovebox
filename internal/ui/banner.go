package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// BannerInfo contains the information to display in the startup banner
type BannerInfo struct {
	Workspace       string
	Image           string
	Container       string
	ContainerStatus string // "new", "existing", "running"
	PassthroughEnv  []string
}

// Banner renders the glovebox startup banner
type Banner struct {
	term *Terminal
}

// NewBanner creates a new Banner renderer
func NewBanner() *Banner {
	return &Banner{
		term: NewTerminal(),
	}
}

// Render produces the formatted banner string
func (b *Banner) Render(info BannerInfo) string {
	var sb strings.Builder

	bar := b.term.VerticalBar()

	// Define styles based on terminal capabilities
	var (
		barStyle    lipgloss.Style
		titleStyle  lipgloss.Style
		labelStyle  lipgloss.Style
		valueStyle  lipgloss.Style
		statusStyle lipgloss.Style
	)

	if b.term.HasColors() {
		barStyle = b.term.NewStyle().Foreground(lipgloss.Color("240"))
		titleStyle = b.term.NewStyle().Bold(true)
		labelStyle = b.term.NewStyle().Foreground(lipgloss.Color("240"))
		valueStyle = b.term.NewStyle()
		statusStyle = b.term.NewStyle().Foreground(lipgloss.Color("240"))
	} else {
		barStyle = b.term.NewStyle()
		titleStyle = b.term.NewStyle()
		labelStyle = b.term.NewStyle()
		valueStyle = b.term.NewStyle()
		statusStyle = b.term.NewStyle()
	}

	// Helper to render a line with the bar prefix
	line := func(content string) {
		sb.WriteString(fmt.Sprintf("  %s %s\n", barStyle.Render(bar), content))
	}

	// Helper to render a label-value pair
	labelValue := func(label, value string) string {
		// Pad label to align values
		paddedLabel := fmt.Sprintf("%-11s", label)
		return labelStyle.Render(paddedLabel) + valueStyle.Render(value)
	}

	// Build the banner
	sb.WriteString("\n")
	line(titleStyle.Render("glovebox"))
	line("")
	line(labelValue("Workspace", info.Workspace))
	line(labelValue("Image", info.Image))

	// Container line with status
	containerLine := labelValue("Container", info.Container)
	if info.ContainerStatus != "" {
		containerLine += statusStyle.Render(fmt.Sprintf(" (%s)", info.ContainerStatus))
	}
	line(containerLine)

	// Passthrough env (if any)
	if len(info.PassthroughEnv) > 0 {
		line(labelValue("Env", strings.Join(info.PassthroughEnv, ", ")))
	}

	sb.WriteString("\n")

	return sb.String()
}

// Print renders and prints the banner to stdout
func (b *Banner) Print(info BannerInfo) {
	fmt.Print(b.Render(info))
}
