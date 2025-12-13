package ui

import (
	"strings"
	"testing"
)

func TestBannerRender(t *testing.T) {
	banner := NewBanner()

	info := BannerInfo{
		Workspace:       "~/code/myproject",
		Image:           "glovebox:myproject-abc123",
		Container:       "glovebox-myproject-abc123",
		ContainerStatus: "new",
		PassthroughEnv:  []string{"FOO", "BAR"},
	}

	output := banner.Render(info)

	// Check that key information is present
	if !strings.Contains(output, "glovebox") {
		t.Error("expected banner to contain 'glovebox'")
	}
	if !strings.Contains(output, "~/code/myproject") {
		t.Error("expected banner to contain workspace path")
	}
	if !strings.Contains(output, "glovebox:myproject-abc123") {
		t.Error("expected banner to contain image name")
	}
	if !strings.Contains(output, "glovebox-myproject-abc123") {
		t.Error("expected banner to contain container name")
	}
	if !strings.Contains(output, "new") {
		t.Error("expected banner to contain container status")
	}
	if !strings.Contains(output, "FOO") || !strings.Contains(output, "BAR") {
		t.Error("expected banner to contain passthrough env vars")
	}
}

func TestBannerRenderWithoutEnv(t *testing.T) {
	banner := NewBanner()

	info := BannerInfo{
		Workspace:       "~/code/myproject",
		Image:           "glovebox:myproject-abc123",
		Container:       "glovebox-myproject-abc123",
		ContainerStatus: "existing",
		PassthroughEnv:  nil,
	}

	output := banner.Render(info)

	// Should not contain "Env" line when no passthrough vars
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Env") {
			t.Error("expected banner to not contain 'Env' line when no passthrough vars")
		}
	}
}

func TestTerminalVerticalBar(t *testing.T) {
	term := NewTerminal()

	bar := term.VerticalBar()

	// Should be either Unicode or ASCII
	if bar != "┃" && bar != "|" {
		t.Errorf("expected vertical bar to be '┃' or '|', got %q", bar)
	}
}
