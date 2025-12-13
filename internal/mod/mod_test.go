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
			name:    "load base mod",
			id:      "base",
			wantErr: false,
			check: func(t *testing.T, m *Mod) {
				if m.Name != "base" {
					t.Errorf("expected name 'base', got %q", m.Name)
				}
				if m.Category != "core" {
					t.Errorf("expected category 'core', got %q", m.Category)
				}
				if len(m.AptPackages) == 0 {
					t.Error("expected apt packages in base mod")
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
		data, source, err := LoadRaw("base")
		if err != nil {
			t.Fatalf("LoadRaw('base') error = %v", err)
		}
		if source != "embedded" {
			t.Errorf("expected source 'embedded', got %q", source)
		}
		if len(data) == 0 {
			t.Error("expected non-empty data")
		}
		// Verify it contains expected content
		if !contains(string(data), "name: base") {
			t.Error("expected YAML to contain 'name: base'")
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

	// Should have core category with base mod
	if _, ok := result["core"]; !ok {
		t.Error("expected 'core' category in result")
	}
	if !containsString(result["core"], "base") {
		t.Error("expected 'base' in core category")
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
		// mise requires tools/homebrew which requires base
		mods, err := LoadMultiple([]string{"tools/mise"})
		if err != nil {
			t.Fatalf("LoadMultiple() error = %v", err)
		}

		// Should have at least 3 mods: base, homebrew, mise
		if len(mods) < 3 {
			t.Errorf("expected at least 3 mods (base, homebrew, mise), got %d", len(mods))
		}

		// Verify dependency order: base should come before homebrew, homebrew before mise
		names := make([]string, len(mods))
		for i, m := range mods {
			names[i] = m.Name
		}

		baseIdx := indexOf(names, "base")
		homebrewIdx := indexOf(names, "homebrew")
		miseIdx := indexOf(names, "mise")

		if baseIdx == -1 || homebrewIdx == -1 || miseIdx == -1 {
			t.Errorf("expected base, homebrew, mise in results, got %v", names)
			return
		}

		if baseIdx > homebrewIdx {
			t.Error("base should come before homebrew")
		}
		if homebrewIdx > miseIdx {
			t.Error("homebrew should come before mise")
		}
	})

	t.Run("load multiple mods with shared dependencies", func(t *testing.T) {
		// Both neovim and mise require homebrew
		mods, err := LoadMultiple([]string{"editors/neovim", "tools/mise"})
		if err != nil {
			t.Fatalf("LoadMultiple() error = %v", err)
		}

		// Count how many times each mod appears
		counts := make(map[string]int)
		for _, m := range mods {
			counts[m.Name]++
		}

		// Homebrew should only appear once (deduped)
		if counts["homebrew"] != 1 {
			t.Errorf("homebrew should appear exactly once, got %d", counts["homebrew"])
		}
		// Base should only appear once
		if counts["base"] != 1 {
			t.Errorf("base should appear exactly once, got %d", counts["base"])
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
		// Load mise but exclude base and homebrew (as if they're in base image)
		baseModIDs := []string{"base", "tools/homebrew"}
		mods, err := LoadMultipleExcluding([]string{"tools/mise"}, baseModIDs)
		if err != nil {
			t.Fatalf("LoadMultipleExcluding() error = %v", err)
		}

		// Should only have mise, not base or homebrew
		for _, m := range mods {
			if m.Name == "base" {
				t.Error("base should be excluded")
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

	t.Run("empty base mods", func(t *testing.T) {
		// Should behave like LoadMultiple when no base mods
		mods, err := LoadMultipleExcluding([]string{"tools/mise"}, nil)
		if err != nil {
			t.Fatalf("LoadMultipleExcluding() error = %v", err)
		}

		// Should include all dependencies
		names := make([]string, len(mods))
		for i, m := range mods {
			names[i] = m.Name
		}

		if !containsString(names, "base") {
			t.Error("expected base to be included with empty exclusions")
		}
	})
}

func TestAddModToResult(t *testing.T) {
	t.Run("add top-level mod to core", func(t *testing.T) {
		result := make(map[string][]string)
		seen := make(map[string]bool)

		addModToResult(result, seen, "base")

		if !containsString(result["core"], "base") {
			t.Error("expected 'base' in 'core' category")
		}
	})

	t.Run("add categorized mod", func(t *testing.T) {
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

		addModToResult(result, seen, "base")
		addModToResult(result, seen, "base") // add again

		if len(result["core"]) != 1 {
			t.Errorf("expected 1 entry in core, got %d", len(result["core"]))
		}
	})
}

func TestModStruct(t *testing.T) {
	m, err := Load("base")
	if err != nil {
		t.Fatalf("Load('base') error = %v", err)
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

	// AptPackages should be populated for base
	if len(m.AptPackages) == 0 {
		t.Error("AptPackages should not be empty for base mod")
	}

	// Base mod should have run_as_root
	if m.RunAsRoot == "" {
		t.Error("RunAsRoot should not be empty for base mod")
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
