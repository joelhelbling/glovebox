package ui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Terminal holds information about terminal capabilities
type Terminal struct {
	SupportsUnicode bool
	ColorProfile    termenv.Profile
	output          *lipgloss.Renderer
}

// NewTerminal detects terminal capabilities and returns a Terminal instance
func NewTerminal() *Terminal {
	output := lipgloss.DefaultRenderer()

	return &Terminal{
		SupportsUnicode: detectUnicodeSupport(),
		ColorProfile:    output.ColorProfile(),
		output:          output,
	}
}

// detectUnicodeSupport checks if the terminal likely supports Unicode
func detectUnicodeSupport() bool {
	// Check TERM for known limited terminals
	term := strings.ToLower(os.Getenv("TERM"))
	limitedTerms := []string{"dumb", "linux", "cons25", "cygwin"}
	for _, t := range limitedTerms {
		if term == t {
			return false
		}
	}

	// Check if locale indicates UTF-8 support
	for _, env := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		val := strings.ToUpper(os.Getenv(env))
		if strings.Contains(val, "UTF-8") || strings.Contains(val, "UTF8") {
			return true
		}
	}

	// Default to true for modern terminals (most support Unicode now)
	// Only fall back to ASCII for explicitly limited terminals
	return term != ""
}

// Renderer returns the lipgloss renderer for this terminal
func (t *Terminal) Renderer() *lipgloss.Renderer {
	return t.output
}

// VerticalBar returns the appropriate vertical bar character
func (t *Terminal) VerticalBar() string {
	if t.SupportsUnicode {
		return "┃"
	}
	return "|"
}

// HasColors returns true if the terminal supports colors
func (t *Terminal) HasColors() bool {
	return t.ColorProfile != termenv.Ascii
}

// NewStyle creates a new style bound to this terminal's renderer
func (t *Terminal) NewStyle() lipgloss.Style {
	return t.output.NewStyle()
}
