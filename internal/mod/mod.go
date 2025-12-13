package mod

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed all:mods
var modFS embed.FS

// Mod represents a composable piece of Dockerfile configuration
type Mod struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Category    string            `yaml:"category"`
	Requires    []string          `yaml:"requires,omitempty"`
	AptRepos    []string          `yaml:"apt_repos,omitempty"`
	AptPackages []string          `yaml:"apt_packages,omitempty"`
	RunAsRoot   string            `yaml:"run_as_root,omitempty"`
	RunAsUser   string            `yaml:"run_as_user,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	UserShell   string            `yaml:"user_shell,omitempty"`
}

// modSearchPaths returns the directories to search for mods, in priority order:
// 1. Project-local: .glovebox/mods/
// 2. User global: ~/.glovebox/mods/
// Embedded mods are checked last (in Load function)
func modSearchPaths() []string {
	var paths []string

	// Project-local mods
	cwd, err := os.Getwd()
	if err == nil {
		paths = append(paths, filepath.Join(cwd, ".glovebox", "mods"))
	}

	// User global mods
	home, err := os.UserHomeDir()
	if err == nil {
		paths = append(paths, filepath.Join(home, ".glovebox", "mods"))
	}

	return paths
}

// loadFromFile attempts to load a mod from a filesystem path
func loadFromFile(path string) (*Mod, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m Mod
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing mod: %w", err)
	}

	return &m, nil
}

// validateModID checks that a mod ID doesn't contain path traversal sequences
func validateModID(id string) error {
	if strings.Contains(id, "..") {
		return fmt.Errorf("invalid mod id: %s (path traversal not allowed)", id)
	}
	if filepath.IsAbs(id) {
		return fmt.Errorf("invalid mod id: %s (absolute paths not allowed)", id)
	}
	return nil
}

// Load reads a mod by its ID (e.g., "shells/fish"), checking:
// 1. Project-local: .glovebox/mods/<id>.yaml
// 2. User global: ~/.glovebox/mods/<id>.yaml
// 3. Embedded mods (bundled in binary)
func Load(id string) (*Mod, error) {
	if err := validateModID(id); err != nil {
		return nil, err
	}

	filename := id + ".yaml"

	// Check local filesystem paths first
	for _, searchPath := range modSearchPaths() {
		fullPath := filepath.Join(searchPath, filename)
		if m, err := loadFromFile(fullPath); err == nil {
			return m, nil
		}
	}

	// Fall back to embedded mods
	embeddedPath := filepath.Join("mods", filename)
	data, err := modFS.ReadFile(embeddedPath)
	if err != nil {
		return nil, fmt.Errorf("mod not found: %s", id)
	}

	var m Mod
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing mod %s: %w", id, err)
	}

	return &m, nil
}

// LoadRaw reads a mod's raw YAML content by its ID.
// Returns the raw bytes and the source path (or "embedded" for built-in mods).
func LoadRaw(id string) ([]byte, string, error) {
	if err := validateModID(id); err != nil {
		return nil, "", err
	}

	filename := id + ".yaml"

	// Check local filesystem paths first
	for _, searchPath := range modSearchPaths() {
		fullPath := filepath.Join(searchPath, filename)
		data, err := os.ReadFile(fullPath)
		if err == nil {
			return data, fullPath, nil
		}
	}

	// Fall back to embedded mods
	embeddedPath := filepath.Join("mods", filename)
	data, err := modFS.ReadFile(embeddedPath)
	if err != nil {
		return nil, "", fmt.Errorf("mod not found: %s", id)
	}

	return data, "embedded", nil
}

// addModToResult adds a mod ID to the result map, extracting category from path
func addModToResult(result map[string][]string, seen map[string]bool, id string) {
	if seen[id] {
		return
	}
	seen[id] = true

	parts := strings.Split(id, "/")
	if len(parts) == 1 {
		// Top-level mod (e.g., "base")
		result["core"] = append(result["core"], id)
	} else {
		category := parts[0]
		result[category] = append(result[category], id)
	}
}

// listLocalMods walks a local directory and adds found mods to result
func listLocalMods(dir string, result map[string][]string, seen map[string]bool) {
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		// Convert path to mod ID
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		id := strings.TrimSuffix(rel, ".yaml")
		addModToResult(result, seen, id)
		return nil
	})
}

// ListAll returns all available mod IDs organized by category.
// It includes mods from:
// 1. Project-local: .glovebox/mods/
// 2. User global: ~/.glovebox/mods/
// 3. Embedded mods (bundled in binary)
// Local mods take precedence and can override embedded ones.
func ListAll() (map[string][]string, error) {
	result := make(map[string][]string)
	seen := make(map[string]bool)

	// Check local filesystem paths first (they take precedence)
	for _, searchPath := range modSearchPaths() {
		listLocalMods(searchPath, result, seen)
	}

	// Add embedded mods (if not already seen)
	err := fs.WalkDir(modFS, "mods", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		// Convert path like "mods/shells/fish.yaml" to "shells/fish"
		rel := strings.TrimPrefix(path, "mods/")
		id := strings.TrimSuffix(rel, ".yaml")
		addModToResult(result, seen, id)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("listing mods: %w", err)
	}

	return result, nil
}

// LoadMultiple loads multiple mods by their IDs and resolves dependencies
func LoadMultiple(ids []string) ([]*Mod, error) {
	return LoadMultipleExcluding(ids, nil)
}

// LoadMultipleExcluding loads multiple mods by their IDs and resolves dependencies,
// but excludes any mods (and their dependencies) that are already satisfied by
// the provided base mod IDs. This is used for project builds where the base image
// already contains certain mods.
func LoadMultipleExcluding(ids []string, baseModIDs []string) ([]*Mod, error) {
	// Build a set of mod IDs that are already in the base (including their dependencies)
	baseSatisfied := make(map[string]bool)
	if len(baseModIDs) > 0 {
		// Resolve all base mod IDs including their dependencies
		allBaseIDs, err := resolveAllDependencies(baseModIDs)
		if err != nil {
			return nil, fmt.Errorf("resolving base mods: %w", err)
		}
		for _, id := range allBaseIDs {
			baseSatisfied[id] = true
		}
	}

	return loadMultipleInternal(ids, baseSatisfied)
}

// loadMultipleInternal is the core implementation that loads mods with dependency
// resolution, optionally skipping mods that are already satisfied.
func loadMultipleInternal(ids []string, satisfied map[string]bool) ([]*Mod, error) {
	loaded := make(map[string]*Mod)
	var order []string

	var loadWithDeps func(id string) error
	loadWithDeps = func(id string) error {
		// Skip if already loaded in this run
		if _, exists := loaded[id]; exists {
			return nil
		}

		// Skip if already satisfied by base
		if satisfied != nil && satisfied[id] {
			return nil
		}

		m, err := Load(id)
		if err != nil {
			return err
		}

		// Load dependencies first
		for _, dep := range m.Requires {
			if err := loadWithDeps(dep); err != nil {
				return fmt.Errorf("dependency %s of %s: %w", dep, id, err)
			}
		}

		loaded[id] = m
		order = append(order, id)
		return nil
	}

	for _, id := range ids {
		if err := loadWithDeps(id); err != nil {
			return nil, err
		}
	}

	// Return mods in dependency order
	result := make([]*Mod, len(order))
	for i, id := range order {
		result[i] = loaded[id]
	}

	return result, nil
}

// resolveAllDependencies returns a list of all mod IDs (including the given IDs
// and all their transitive dependencies) in dependency order.
func resolveAllDependencies(ids []string) ([]string, error) {
	resolved := make(map[string]bool)
	var order []string

	var resolve func(id string) error
	resolve = func(id string) error {
		if resolved[id] {
			return nil
		}

		m, err := Load(id)
		if err != nil {
			return err
		}

		// Resolve dependencies first
		for _, dep := range m.Requires {
			if err := resolve(dep); err != nil {
				return fmt.Errorf("dependency %s of %s: %w", dep, id, err)
			}
		}

		resolved[id] = true
		order = append(order, id)
		return nil
	}

	for _, id := range ids {
		if err := resolve(id); err != nil {
			return nil, err
		}
	}

	return order, nil
}
