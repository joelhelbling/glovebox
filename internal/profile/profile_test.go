package profile

import (
	"path/filepath"
	"testing"
)

func TestNewProfile(t *testing.T) {
	p := NewProfile()

	if p.Version != 1 {
		t.Errorf("expected Version 1, got %d", p.Version)
	}
	if p.Mods == nil {
		t.Error("Mods should not be nil")
	}
	if len(p.Mods) != 0 {
		t.Errorf("expected empty Mods slice, got %d items", len(p.Mods))
	}
	if p.Path != "" {
		t.Errorf("expected empty Path, got %q", p.Path)
	}
	if p.IsGlobal {
		t.Error("IsGlobal should be false for new profile")
	}
}

func TestProjectPath(t *testing.T) {
	tests := []struct {
		name string
		dir  string
		want string
	}{
		{
			name: "simple directory",
			dir:  "/home/user/project",
			want: "/home/user/project/.glovebox/profile.yaml",
		},
		{
			name: "root directory",
			dir:  "/",
			want: "/.glovebox/profile.yaml",
		},
		{
			name: "relative path",
			dir:  "myproject",
			want: "myproject/.glovebox/profile.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProjectPath(tt.dir)
			if got != tt.want {
				t.Errorf("ProjectPath(%q) = %q, want %q", tt.dir, got, tt.want)
			}
		})
	}
}

func TestProjectDir(t *testing.T) {
	got := ProjectDir("/home/user/project")
	want := "/home/user/project/.glovebox"
	if got != want {
		t.Errorf("ProjectDir() = %q, want %q", got, want)
	}
}

func TestGenerateImageName(t *testing.T) {
	t.Run("deterministic output", func(t *testing.T) {
		dir := "/some/project/path"
		first := GenerateImageName(dir)
		second := GenerateImageName(dir)

		if first != second {
			t.Errorf("GenerateImageName not deterministic: %q != %q", first, second)
		}
	})

	t.Run("format check", func(t *testing.T) {
		dir := "/home/user/myproject"
		got := GenerateImageName(dir)

		// Should start with "glovebox:"
		if len(got) < 10 || got[:9] != "glovebox:" {
			t.Errorf("expected format 'glovebox:<name>-<hash>', got %q", got)
		}

		// Should contain the directory name
		if !containsString(got, "myproject") {
			t.Errorf("expected image name to contain 'myproject', got %q", got)
		}
	})

	t.Run("different paths produce different names", func(t *testing.T) {
		name1 := GenerateImageName("/path/one")
		name2 := GenerateImageName("/path/two")

		if name1 == name2 {
			t.Error("different paths should produce different image names")
		}
	})

	t.Run("same dirname different paths", func(t *testing.T) {
		// Two projects named "app" in different locations
		name1 := GenerateImageName("/home/user1/app")
		name2 := GenerateImageName("/home/user2/app")

		// Both should contain "app" but have different hashes
		if name1 == name2 {
			t.Error("same dirname with different full paths should have different hashes")
		}
	})
}

func TestAddMod(t *testing.T) {
	t.Run("add new mod", func(t *testing.T) {
		p := NewProfile()
		added := p.AddMod("shells/bash")

		if !added {
			t.Error("AddMod should return true for new mod")
		}
		if len(p.Mods) != 1 {
			t.Errorf("expected 1 mod, got %d", len(p.Mods))
		}
		if p.Mods[0] != "shells/bash" {
			t.Errorf("expected 'shells/bash', got %q", p.Mods[0])
		}
	})

	t.Run("add duplicate mod", func(t *testing.T) {
		p := NewProfile()
		p.AddMod("shells/bash")
		added := p.AddMod("shells/bash")

		if added {
			t.Error("AddMod should return false for duplicate mod")
		}
		if len(p.Mods) != 1 {
			t.Errorf("expected 1 mod (no duplicate), got %d", len(p.Mods))
		}
	})

	t.Run("add multiple different mods", func(t *testing.T) {
		p := NewProfile()
		p.AddMod("shells/bash")
		p.AddMod("editors/vim")
		p.AddMod("tools/mise")

		if len(p.Mods) != 3 {
			t.Errorf("expected 3 mods, got %d", len(p.Mods))
		}
	})
}

func TestRemoveMod(t *testing.T) {
	t.Run("remove existing mod", func(t *testing.T) {
		p := NewProfile()
		p.AddMod("shells/bash")
		p.AddMod("editors/vim")

		removed := p.RemoveMod("shells/bash")

		if !removed {
			t.Error("RemoveMod should return true for existing mod")
		}
		if len(p.Mods) != 1 {
			t.Errorf("expected 1 mod remaining, got %d", len(p.Mods))
		}
		if p.Mods[0] != "editors/vim" {
			t.Errorf("expected 'editors/vim' to remain, got %q", p.Mods[0])
		}
	})

	t.Run("remove non-existent mod", func(t *testing.T) {
		p := NewProfile()
		p.AddMod("shells/bash")

		removed := p.RemoveMod("nonexistent")

		if removed {
			t.Error("RemoveMod should return false for non-existent mod")
		}
		if len(p.Mods) != 1 {
			t.Errorf("expected 1 mod (unchanged), got %d", len(p.Mods))
		}
	})

	t.Run("remove from empty profile", func(t *testing.T) {
		p := NewProfile()
		removed := p.RemoveMod("anything")

		if removed {
			t.Error("RemoveMod should return false for empty profile")
		}
	})
}

func TestHasMod(t *testing.T) {
	p := NewProfile()
	p.AddMod("shells/bash")
	p.AddMod("editors/vim")

	tests := []struct {
		name string
		id   string
		want bool
	}{
		{"existing mod", "shells/bash", true},
		{"another existing mod", "editors/vim", true},
		{"non-existent mod", "tools/mise", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.HasMod(tt.id)
			if got != tt.want {
				t.Errorf("HasMod(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	t.Run("save and load profile", func(t *testing.T) {
		tmpDir := t.TempDir()
		profilePath := filepath.Join(tmpDir, ".glovebox", "profile.yaml")

		// Create and save profile
		p := NewProfile()
		p.AddMod("shells/bash")
		p.AddMod("editors/vim")

		err := p.SaveTo(profilePath)
		if err != nil {
			t.Fatalf("SaveTo() error = %v", err)
		}

		// Load it back
		loaded, err := Load(profilePath)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if loaded.Version != p.Version {
			t.Errorf("Version mismatch: got %d, want %d", loaded.Version, p.Version)
		}
		if len(loaded.Mods) != len(p.Mods) {
			t.Errorf("Mods count mismatch: got %d, want %d", len(loaded.Mods), len(p.Mods))
		}
		for i, mod := range p.Mods {
			if loaded.Mods[i] != mod {
				t.Errorf("Mod[%d] mismatch: got %q, want %q", i, loaded.Mods[i], mod)
			}
		}
		if loaded.Path != profilePath {
			t.Errorf("Path mismatch: got %q, want %q", loaded.Path, profilePath)
		}
	})

	t.Run("load non-existent file returns nil", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonExistent := filepath.Join(tmpDir, "does-not-exist.yaml")

		loaded, err := Load(nonExistent)
		if err != nil {
			t.Errorf("Load() should not error for non-existent file, got %v", err)
		}
		if loaded != nil {
			t.Error("Load() should return nil for non-existent file")
		}
	})

	t.Run("save without path fails", func(t *testing.T) {
		p := NewProfile()
		err := p.Save()
		if err == nil {
			t.Error("Save() should error when Path is not set")
		}
	})
}

func TestImageName(t *testing.T) {
	t.Run("global profile returns base", func(t *testing.T) {
		p := NewProfile()
		p.IsGlobal = true

		got := p.ImageName()
		if got != "glovebox:base" {
			t.Errorf("ImageName() = %q, want 'glovebox:base'", got)
		}
	})

	t.Run("profile with explicit image name", func(t *testing.T) {
		p := NewProfile()
		p.Build.ImageName = "custom:image"

		got := p.ImageName()
		if got != "custom:image" {
			t.Errorf("ImageName() = %q, want 'custom:image'", got)
		}
	})

	t.Run("project profile generates name from path", func(t *testing.T) {
		p := NewProfile()
		p.Path = "/home/user/myproject/.glovebox/profile.yaml"

		got := p.ImageName()
		if !containsString(got, "myproject") {
			t.Errorf("ImageName() = %q, expected to contain 'myproject'", got)
		}
	})
}

func TestDockerfilePath(t *testing.T) {
	t.Run("global profile", func(t *testing.T) {
		// Create profile that looks like global
		p := NewProfile()
		p.IsGlobal = true

		got := p.DockerfilePath()
		// Should end with Dockerfile
		if filepath.Base(got) != "Dockerfile" {
			t.Errorf("DockerfilePath() = %q, expected to end with 'Dockerfile'", got)
		}
	})

	t.Run("project profile", func(t *testing.T) {
		p := NewProfile()
		p.Path = "/home/user/project/.glovebox/profile.yaml"

		got := p.DockerfilePath()
		want := "/home/user/project/.glovebox/Dockerfile"
		if got != want {
			t.Errorf("DockerfilePath() = %q, want %q", got, want)
		}
	})
}

func TestGlobalPath(t *testing.T) {
	path, err := GlobalPath()
	if err != nil {
		t.Fatalf("GlobalPath() error = %v", err)
	}

	// Should end with .glovebox/profile.yaml
	if filepath.Base(path) != "profile.yaml" {
		t.Errorf("GlobalPath() = %q, expected to end with 'profile.yaml'", path)
	}
	if filepath.Base(filepath.Dir(path)) != ".glovebox" {
		t.Errorf("GlobalPath() parent should be '.glovebox', got %q", filepath.Dir(path))
	}
}

func TestGlobalDir(t *testing.T) {
	dir, err := GlobalDir()
	if err != nil {
		t.Fatalf("GlobalDir() error = %v", err)
	}

	// Should end with .glovebox
	if filepath.Base(dir) != ".glovebox" {
		t.Errorf("GlobalDir() = %q, expected to end with '.glovebox'", dir)
	}
}

func TestUpdateBuildInfo(t *testing.T) {
	p := NewProfile()

	// Initially empty
	if !p.Build.LastBuiltAt.IsZero() {
		t.Error("LastBuiltAt should be zero initially")
	}

	p.UpdateBuildInfo("sha256:abc123")

	if p.Build.LastBuiltAt.IsZero() {
		t.Error("LastBuiltAt should be set after UpdateBuildInfo")
	}
	if p.Build.DockerfileDigest != "sha256:abc123" {
		t.Errorf("DockerfileDigest = %q, want 'sha256:abc123'", p.Build.DockerfileDigest)
	}
}

func TestLoadGlobal(t *testing.T) {
	// This test depends on whether global profile exists
	// We just verify it doesn't panic
	_, err := LoadGlobal()
	// err could be nil (if profile exists) or not (if doesn't exist)
	_ = err
}

func TestLoadProject(t *testing.T) {
	t.Run("non-existent project dir", func(t *testing.T) {
		p, err := LoadProject("/nonexistent/path")
		if err != nil {
			t.Errorf("LoadProject() should not error for non-existent, got %v", err)
		}
		if p != nil {
			t.Error("LoadProject() should return nil for non-existent project")
		}
	})

	t.Run("existing project profile", func(t *testing.T) {
		tmpDir := t.TempDir()
		profilePath := ProjectPath(tmpDir)

		// Create a profile
		p := NewProfile()
		p.AddMod("shells/bash")
		if err := p.SaveTo(profilePath); err != nil {
			t.Fatalf("SaveTo() error = %v", err)
		}

		// Load it
		loaded, err := LoadProject(tmpDir)
		if err != nil {
			t.Fatalf("LoadProject() error = %v", err)
		}
		if loaded == nil {
			t.Fatal("LoadProject() returned nil for existing profile")
		}
		if !loaded.HasMod("shells/bash") {
			t.Error("loaded profile should have 'shells/bash' mod")
		}
	})
}

func TestLoadEffective(t *testing.T) {
	t.Run("project profile takes precedence", func(t *testing.T) {
		tmpDir := t.TempDir()
		profilePath := ProjectPath(tmpDir)

		// Create a project profile
		p := NewProfile()
		p.AddMod("project-specific-mod")
		if err := p.SaveTo(profilePath); err != nil {
			t.Fatalf("SaveTo() error = %v", err)
		}

		// LoadEffective should find the project profile
		loaded, err := LoadEffective(tmpDir)
		if err != nil {
			t.Fatalf("LoadEffective() error = %v", err)
		}
		if loaded == nil {
			t.Fatal("LoadEffective() returned nil")
		}
		if !loaded.HasMod("project-specific-mod") {
			t.Error("LoadEffective should return project profile")
		}
	})
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
