package mod

import (
	"strings"
	"testing"
)

func TestValidateModID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid simple id", "base", false},
		{"valid category path", "shells/bash", false},
		{"valid nested path", "ai/claude-code", false},
		{"path traversal simple", "..", true},
		{"path traversal prefix", "../etc/passwd", true},
		{"path traversal middle", "shells/../../../etc/passwd", true},
		{"path traversal suffix", "shells/..", true},
		{"absolute path unix", "/etc/passwd", true},
		{"absolute path with category", "/shells/bash", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateModID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateModID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
			if err != nil && tt.wantErr {
				// Verify error message is informative
				if !strings.Contains(err.Error(), tt.id) {
					t.Errorf("error should contain the invalid id, got: %v", err)
				}
			}
		})
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
		check   func(*testing.T, *Mod)
	}{
		{
			name:    "load OS mod",
			id:      "os/ubuntu",
			wantErr: false,
			check: func(t *testing.T, m *Mod) {
				if m.Name != "ubuntu" {
					t.Errorf("expected name 'ubuntu', got %q", m.Name)
				}
				if m.Category != "os" {
					t.Errorf("expected category 'os', got %q", m.Category)
				}
				if m.DockerfileFrom == "" {
					t.Error("expected dockerfile_from in OS mod")
				}
			},
		},
		{
			name:    "load shell mod with category path",
			id:      "shells/bash",
			wantErr: false,
			check: func(t *testing.T, m *Mod) {
				if m.Name != "bash" {
					t.Errorf("expected name 'bash', got %q", m.Name)
				}
				if m.Category != "shell" {
					t.Errorf("expected category 'shell', got %q", m.Category)
				}
			},
		},
		{
			name:    "load tool mod",
			id:      "tools/homebrew",
			wantErr: false,
			check: func(t *testing.T, m *Mod) {
				if m.Name != "homebrew" {
					t.Errorf("expected name 'homebrew', got %q", m.Name)
				}
				// homebrew requires base
				if len(m.Requires) == 0 || m.Requires[0] != "base" {
					t.Errorf("expected homebrew to require 'base', got %v", m.Requires)
				}
			},
		},
		{
			name:    "load non-existent mod",
			id:      "nonexistent/fake",
			wantErr: true,
			check:   nil,
		},
		{
			name:    "reject path traversal with ..",
			id:      "../../../etc/passwd",
			wantErr: true,
			check:   nil,
		},
		{
			name:    "reject path traversal in middle",
			id:      "shells/../../../etc/passwd",
			wantErr: true,
			check:   nil,
		},
		{
			name:    "reject absolute path",
			id:      "/etc/passwd",
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := Load(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
				return
			}
			if tt.check != nil && m != nil {
				tt.check(t, m)
			}
		})
	}
}

func TestLoadRaw(t *testing.T) {
	t.Run("load embedded mod", func(t *testing.T) {
		data, source, err := LoadRaw("os/ubuntu")
		if err != nil {
			t.Fatalf("LoadRaw('os/ubuntu') error = %v", err)
		}
		if source != "embedded" {
			t.Errorf("expected source 'embedded', got %q", source)
		}
		if len(data) == 0 {
			t.Error("expected non-empty data")
		}
		// Verify it contains expected content
		if !contains(string(data), "name: ubuntu") {
			t.Error("expected YAML to contain 'name: ubuntu'")
		}
	})

	t.Run("load non-existent mod", func(t *testing.T) {
		_, _, err := LoadRaw("nonexistent/fake")
		if err == nil {
			t.Error("expected error for non-existent mod")
		}
	})

	t.Run("reject path traversal", func(t *testing.T) {
		_, _, err := LoadRaw("../../../etc/passwd")
		if err == nil {
			t.Error("expected error for path traversal")
		}
	})

	t.Run("reject absolute path", func(t *testing.T) {
		_, _, err := LoadRaw("/etc/passwd")
		if err == nil {
			t.Error("expected error for absolute path")
		}
	})
}

func TestListAll(t *testing.T) {
	result, err := ListAll()
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}

	// Should have os category with ubuntu mod
	if _, ok := result["os"]; !ok {
		t.Error("expected 'os' category in result")
	}
	if !containsString(result["os"], "os/ubuntu") {
		t.Error("expected 'os/ubuntu' in os category")
	}

	// Should have shells category
	if _, ok := result["shells"]; !ok {
		t.Error("expected 'shells' category in result")
	}
	if !containsString(result["shells"], "shells/bash") {
		t.Error("expected 'shells/bash' in shells category")
	}

	// Should have tools category
	if _, ok := result["tools"]; !ok {
		t.Error("expected 'tools' category in result")
	}

	// Should have editors category
	if _, ok := result["editors"]; !ok {
		t.Error("expected 'editors' category in result")
	}
}

func TestLoadMultiple(t *testing.T) {
	t.Run("load single mod with dependencies", func(t *testing.T) {
		// mise requires base (provided by os/ubuntu)
		mods, err := LoadMultiple([]string{"os/ubuntu", "tools/mise"})
		if err != nil {
			t.Fatalf("LoadMultiple() error = %v", err)
		}

		// Should have at least 2 mods: ubuntu, mise
		if len(mods) < 2 {
			t.Errorf("expected at least 2 mods (ubuntu, mise), got %d", len(mods))
		}

		// Verify dependency order: ubuntu should come before mise
		names := make([]string, len(mods))
		for i, m := range mods {
			names[i] = m.Name
		}

		ubuntuIdx := indexOf(names, "ubuntu")
		miseIdx := indexOf(names, "mise")

		if ubuntuIdx == -1 || miseIdx == -1 {
			t.Errorf("expected ubuntu, mise in results, got %v", names)
			return
		}

		if ubuntuIdx > miseIdx {
			t.Error("ubuntu should come before mise")
		}
	})

	t.Run("load multiple mods with shared dependencies", func(t *testing.T) {
		// Both neovim-ubuntu and mise require base (ubuntu provides base)
		mods, err := LoadMultiple([]string{"os/ubuntu", "editors/neovim-ubuntu", "tools/mise"})
		if err != nil {
			t.Fatalf("LoadMultiple() error = %v", err)
		}

		// Count how many times each mod appears
		counts := make(map[string]int)
		for _, m := range mods {
			counts[m.Name]++
		}

		// Ubuntu should only appear once (deduped)
		if counts["ubuntu"] != 1 {
			t.Errorf("ubuntu should appear exactly once, got %d", counts["ubuntu"])
		}
	})

	t.Run("load non-existent mod", func(t *testing.T) {
		_, err := LoadMultiple([]string{"nonexistent/fake"})
		if err == nil {
			t.Error("expected error for non-existent mod")
		}
	})
}

func TestLoadMultipleExcluding(t *testing.T) {
	t.Run("exclude base mods", func(t *testing.T) {
		// Load mise but exclude ubuntu and homebrew (as if they're in base image)
		baseModIDs := []string{"os/ubuntu", "tools/homebrew"}
		mods, err := LoadMultipleExcluding([]string{"tools/mise"}, baseModIDs)
		if err != nil {
			t.Fatalf("LoadMultipleExcluding() error = %v", err)
		}

		// Should only have mise, not ubuntu or homebrew
		for _, m := range mods {
			if m.Name == "ubuntu" {
				t.Error("ubuntu should be excluded")
			}
			if m.Name == "homebrew" {
				t.Error("homebrew should be excluded")
			}
		}

		// Should have exactly mise
		if len(mods) != 1 {
			t.Errorf("expected 1 mod (mise only), got %d", len(mods))
		}
		if len(mods) > 0 && mods[0].Name != "mise" {
			t.Errorf("expected mise, got %s", mods[0].Name)
		}
	})

	t.Run("empty base mods with OS mod", func(t *testing.T) {
		// Should behave like LoadMultiple when no base mods
		mods, err := LoadMultipleExcluding([]string{"os/ubuntu", "tools/mise"}, nil)
		if err != nil {
			t.Fatalf("LoadMultipleExcluding() error = %v", err)
		}

		// Should include all dependencies
		names := make([]string, len(mods))
		for i, m := range mods {
			names[i] = m.Name
		}

		if !containsString(names, "ubuntu") {
			t.Error("expected ubuntu to be included with empty exclusions")
		}
	})
}

func TestAddModToResult(t *testing.T) {
	t.Run("add categorized os mod", func(t *testing.T) {
		result := make(map[string][]string)
		seen := make(map[string]bool)

		addModToResult(result, seen, "os/ubuntu")

		if !containsString(result["os"], "os/ubuntu") {
			t.Error("expected 'os/ubuntu' in 'os' category")
		}
	})

	t.Run("add categorized shell mod", func(t *testing.T) {
		result := make(map[string][]string)
		seen := make(map[string]bool)

		addModToResult(result, seen, "shells/bash")

		if !containsString(result["shells"], "shells/bash") {
			t.Error("expected 'shells/bash' in 'shells' category")
		}
	})

	t.Run("skip already seen mod", func(t *testing.T) {
		result := make(map[string][]string)
		seen := make(map[string]bool)

		addModToResult(result, seen, "os/ubuntu")
		addModToResult(result, seen, "os/ubuntu") // add again

		if len(result["os"]) != 1 {
			t.Errorf("expected 1 entry in os, got %d", len(result["os"]))
		}
	})
}

func TestModStruct(t *testing.T) {
	m, err := Load("os/ubuntu")
	if err != nil {
		t.Fatalf("Load('os/ubuntu') error = %v", err)
	}

	// Test that all fields are accessible
	if m.Name == "" {
		t.Error("Name should not be empty")
	}
	if m.Description == "" {
		t.Error("Description should not be empty")
	}
	if m.Category == "" {
		t.Error("Category should not be empty")
	}

	// OS mod should have dockerfile_from
	if m.DockerfileFrom == "" {
		t.Error("DockerfileFrom should not be empty for OS mod")
	}

	// OS mod should have run_as_root
	if m.RunAsRoot == "" {
		t.Error("RunAsRoot should not be empty for OS mod")
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func indexOf(slice []string, s string) int {
	for i, item := range slice {
		if item == s {
			return i
		}
	}
	return -1
}

// Tests for new provides and validation logic

func TestEffectiveProvides(t *testing.T) {
	t.Run("mod with no explicit provides", func(t *testing.T) {
		m := &Mod{Name: "vim"}
		provides := m.EffectiveProvides()
		if len(provides) != 1 {
			t.Errorf("expected 1 provide, got %d", len(provides))
		}
		if provides[0] != "vim" {
			t.Errorf("expected 'vim', got %q", provides[0])
		}
	})

	t.Run("mod with explicit provides", func(t *testing.T) {
		m := &Mod{Name: "zsh-ubuntu", Provides: []string{"zsh"}}
		provides := m.EffectiveProvides()
		if len(provides) != 2 {
			t.Errorf("expected 2 provides, got %d", len(provides))
		}
		// Should contain both the name and explicit provides
		if !containsString(provides, "zsh-ubuntu") {
			t.Error("expected 'zsh-ubuntu' in provides")
		}
		if !containsString(provides, "zsh") {
			t.Error("expected 'zsh' in provides")
		}
	})

	t.Run("mod with multiple explicit provides", func(t *testing.T) {
		m := &Mod{Name: "ubuntu", Provides: []string{"base", "linux"}}
		provides := m.EffectiveProvides()
		if len(provides) != 3 {
			t.Errorf("expected 3 provides, got %d", len(provides))
		}
		if !containsString(provides, "ubuntu") {
			t.Error("expected 'ubuntu' in provides")
		}
		if !containsString(provides, "base") {
			t.Error("expected 'base' in provides")
		}
		if !containsString(provides, "linux") {
			t.Error("expected 'linux' in provides")
		}
	})
}

func TestBuildProvidesMap(t *testing.T) {
	mods := []*Mod{
		{Name: "ubuntu", Category: "os", Provides: []string{"base"}},
		{Name: "zsh-ubuntu", Category: "shell", Provides: []string{"zsh"}},
		{Name: "vim-ubuntu", Category: "editor"},
	}

	providesMap := BuildProvidesMap(mods)

	// Each mod provides its own name
	if _, ok := providesMap["ubuntu"]; !ok {
		t.Error("expected 'ubuntu' in provides map")
	}
	if _, ok := providesMap["zsh-ubuntu"]; !ok {
		t.Error("expected 'zsh-ubuntu' in provides map")
	}
	if _, ok := providesMap["vim-ubuntu"]; !ok {
		t.Error("expected 'vim-ubuntu' in provides map")
	}

	// Explicit provides
	if _, ok := providesMap["base"]; !ok {
		t.Error("expected 'base' in provides map")
	}
	if _, ok := providesMap["zsh"]; !ok {
		t.Error("expected 'zsh' in provides map")
	}

	// Check that the right mods provide each name
	if providesMap["base"][0].Name != "ubuntu" {
		t.Error("expected 'ubuntu' to provide 'base'")
	}
	if providesMap["zsh"][0].Name != "zsh-ubuntu" {
		t.Error("expected 'zsh-ubuntu' to provide 'zsh'")
	}
}

func TestValidateOSCategory(t *testing.T) {
	t.Run("no OS mod", func(t *testing.T) {
		mods := []*Mod{
			{Name: "vim", Category: "editor"},
			{Name: "zsh", Category: "shell"},
		}
		osMod, err := ValidateOSCategory(mods)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if osMod != nil {
			t.Error("expected nil OS mod")
		}
	})

	t.Run("single OS mod", func(t *testing.T) {
		mods := []*Mod{
			{Name: "ubuntu", Category: "os"},
			{Name: "vim", Category: "editor"},
		}
		osMod, err := ValidateOSCategory(mods)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if osMod == nil {
			t.Error("expected OS mod")
		}
		if osMod.Name != "ubuntu" {
			t.Errorf("expected 'ubuntu', got %q", osMod.Name)
		}
	})

	t.Run("multiple OS mods", func(t *testing.T) {
		mods := []*Mod{
			{Name: "ubuntu", Category: "os"},
			{Name: "fedora", Category: "os"},
		}
		_, err := ValidateOSCategory(mods)
		if err == nil {
			t.Error("expected error for multiple OS mods")
		}
		if !strings.Contains(err.Error(), "multiple OS mods") {
			t.Errorf("expected error about multiple OS mods, got: %v", err)
		}
	})
}

func TestValidateRequires(t *testing.T) {
	t.Run("all requirements satisfied", func(t *testing.T) {
		mods := []*Mod{
			{Name: "ubuntu", Category: "os"},
			{Name: "zsh-ubuntu", Provides: []string{"zsh"}, Requires: []string{"ubuntu"}},
			{Name: "oh-my-zsh", Requires: []string{"zsh"}},
		}
		providesMap := BuildProvidesMap(mods)
		err := ValidateRequires(mods, providesMap)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("requirement not satisfied", func(t *testing.T) {
		mods := []*Mod{
			{Name: "ubuntu", Category: "os"},
			{Name: "oh-my-zsh", Requires: []string{"zsh"}}, // zsh not provided
		}
		providesMap := BuildProvidesMap(mods)
		err := ValidateRequires(mods, providesMap)
		if err == nil {
			t.Error("expected error for unsatisfied requirement")
		}
		if !strings.Contains(err.Error(), "oh-my-zsh") || !strings.Contains(err.Error(), "zsh") {
			t.Errorf("expected error to mention mod and requirement, got: %v", err)
		}
	})

	t.Run("requirement satisfied by provides", func(t *testing.T) {
		mods := []*Mod{
			{Name: "zsh-ubuntu", Provides: []string{"zsh"}},
			{Name: "oh-my-zsh", Requires: []string{"zsh"}},
		}
		providesMap := BuildProvidesMap(mods)
		err := ValidateRequires(mods, providesMap)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestValidateCrossOSDependencies(t *testing.T) {
	t.Run("no cross-OS issues", func(t *testing.T) {
		osMod := &Mod{Name: "ubuntu", Category: "os"}
		mods := []*Mod{
			osMod,
			{Name: "vim-ubuntu", Requires: []string{"ubuntu"}},
		}
		err := ValidateCrossOSDependencies(mods, osMod)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("cross-OS dependency error", func(t *testing.T) {
		osMod := &Mod{Name: "ubuntu", Category: "os"}
		mods := []*Mod{
			osMod,
			{Name: "vim-fedora", Requires: []string{"fedora"}},
		}
		err := ValidateCrossOSDependencies(mods, osMod)
		if err == nil {
			t.Error("expected error for cross-OS dependency")
		}
		if !strings.Contains(err.Error(), "vim-fedora") || !strings.Contains(err.Error(), "fedora") {
			t.Errorf("expected error to mention mod and OS, got: %v", err)
		}
	})

	t.Run("no OS mod selected", func(t *testing.T) {
		mods := []*Mod{
			{Name: "vim", Requires: []string{"fedora"}},
		}
		// With no OS mod, we don't validate cross-OS (user might be doing something custom)
		err := ValidateCrossOSDependencies(mods, nil)
		if err != nil {
			t.Errorf("unexpected error when no OS selected: %v", err)
		}
	})
}

func TestValidateMods(t *testing.T) {
	t.Run("valid configuration", func(t *testing.T) {
		mods := []*Mod{
			{Name: "ubuntu", Category: "os", Provides: []string{"base"}},
			{Name: "zsh-ubuntu", Provides: []string{"zsh"}, Requires: []string{"ubuntu"}},
			{Name: "oh-my-zsh", Requires: []string{"zsh"}},
		}
		osMod, err := ValidateMods(mods)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if osMod == nil || osMod.Name != "ubuntu" {
			t.Error("expected ubuntu OS mod")
		}
	})

	t.Run("multiple OS mods", func(t *testing.T) {
		mods := []*Mod{
			{Name: "ubuntu", Category: "os"},
			{Name: "fedora", Category: "os"},
		}
		_, err := ValidateMods(mods)
		if err == nil {
			t.Error("expected error for multiple OS mods")
		}
	})

	t.Run("unsatisfied requirement", func(t *testing.T) {
		mods := []*Mod{
			{Name: "ubuntu", Category: "os"},
			{Name: "oh-my-zsh", Requires: []string{"zsh"}},
		}
		_, err := ValidateMods(mods)
		if err == nil {
			t.Error("expected error for unsatisfied requirement")
		}
	})

	t.Run("cross-OS dependency", func(t *testing.T) {
		mods := []*Mod{
			{Name: "ubuntu", Category: "os"},
			{Name: "vim-fedora", Requires: []string{"fedora"}},
		}
		_, err := ValidateMods(mods)
		if err == nil {
			t.Error("expected error for cross-OS dependency")
		}
	})
}

func TestDockerfileFrom(t *testing.T) {
	t.Run("OS mod with dockerfile_from", func(t *testing.T) {
		m := &Mod{
			Name:           "ubuntu",
			Category:       "os",
			DockerfileFrom: "ubuntu:24.04",
		}
		if m.DockerfileFrom != "ubuntu:24.04" {
			t.Errorf("expected 'ubuntu:24.04', got %q", m.DockerfileFrom)
		}
	})

	t.Run("non-OS mod without dockerfile_from", func(t *testing.T) {
		m := &Mod{
			Name:     "vim",
			Category: "editor",
		}
		if m.DockerfileFrom != "" {
			t.Errorf("expected empty dockerfile_from, got %q", m.DockerfileFrom)
		}
	})
}
