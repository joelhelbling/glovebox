package snippet

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed all:snippets
var snippetFS embed.FS

// InstallPhase constants
const (
	InstallPhaseBuild       = "build"        // Default: install during docker build
	InstallPhasePostInstall = "post_install" // Install on first container run
)

// Snippet represents a composable piece of Dockerfile configuration
type Snippet struct {
	Name         string            `yaml:"name"`
	Description  string            `yaml:"description"`
	Category     string            `yaml:"category"`
	InstallPhase string            `yaml:"install_phase,omitempty"` // "build" (default) or "post_install"
	Requires     []string          `yaml:"requires,omitempty"`
	AptRepos     []string          `yaml:"apt_repos,omitempty"`
	AptPackages  []string          `yaml:"apt_packages,omitempty"`
	RunAsRoot    string            `yaml:"run_as_root,omitempty"`
	RunAsUser    string            `yaml:"run_as_user,omitempty"`
	Env          map[string]string `yaml:"env,omitempty"`
	UserShell    string            `yaml:"user_shell,omitempty"`
}

// GetInstallPhase returns the install phase, defaulting to "build" if not specified
func (s *Snippet) GetInstallPhase() string {
	if s.InstallPhase == "" {
		return InstallPhaseBuild
	}
	return s.InstallPhase
}

// IsBuildTime returns true if this snippet should be installed during docker build
func (s *Snippet) IsBuildTime() bool {
	return s.GetInstallPhase() == InstallPhaseBuild
}

// IsPostInstall returns true if this snippet should be installed on first container run
func (s *Snippet) IsPostInstall() bool {
	return s.GetInstallPhase() == InstallPhasePostInstall
}

// snippetSearchPaths returns the directories to search for snippets, in priority order:
// 1. Project-local: .glovebox/snippets/
// 2. User global: ~/.glovebox/snippets/
// Embedded snippets are checked last (in Load function)
func snippetSearchPaths() []string {
	var paths []string

	// Project-local snippets
	cwd, err := os.Getwd()
	if err == nil {
		paths = append(paths, filepath.Join(cwd, ".glovebox", "snippets"))
	}

	// User global snippets
	home, err := os.UserHomeDir()
	if err == nil {
		paths = append(paths, filepath.Join(home, ".glovebox", "snippets"))
	}

	return paths
}

// loadFromFile attempts to load a snippet from a filesystem path
func loadFromFile(path string) (*Snippet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var s Snippet
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing snippet: %w", err)
	}

	return &s, nil
}

// Load reads a snippet by its ID (e.g., "shells/fish"), checking:
// 1. Project-local: .glovebox/snippets/<id>.yaml
// 2. User global: ~/.glovebox/snippets/<id>.yaml
// 3. Embedded snippets (bundled in binary)
func Load(id string) (*Snippet, error) {
	filename := id + ".yaml"

	// Check local filesystem paths first
	for _, searchPath := range snippetSearchPaths() {
		fullPath := filepath.Join(searchPath, filename)
		if s, err := loadFromFile(fullPath); err == nil {
			return s, nil
		}
	}

	// Fall back to embedded snippets
	embeddedPath := filepath.Join("snippets", filename)
	data, err := snippetFS.ReadFile(embeddedPath)
	if err != nil {
		return nil, fmt.Errorf("snippet not found: %s", id)
	}

	var s Snippet
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing snippet %s: %w", id, err)
	}

	return &s, nil
}

// LoadRaw reads a snippet's raw YAML content by its ID.
// Returns the raw bytes and the source path (or "embedded" for built-in snippets).
func LoadRaw(id string) ([]byte, string, error) {
	filename := id + ".yaml"

	// Check local filesystem paths first
	for _, searchPath := range snippetSearchPaths() {
		fullPath := filepath.Join(searchPath, filename)
		data, err := os.ReadFile(fullPath)
		if err == nil {
			return data, fullPath, nil
		}
	}

	// Fall back to embedded snippets
	embeddedPath := filepath.Join("snippets", filename)
	data, err := snippetFS.ReadFile(embeddedPath)
	if err != nil {
		return nil, "", fmt.Errorf("snippet not found: %s", id)
	}

	return data, "embedded", nil
}

// addSnippetToResult adds a snippet ID to the result map, extracting category from path
func addSnippetToResult(result map[string][]string, seen map[string]bool, id string) {
	if seen[id] {
		return
	}
	seen[id] = true

	parts := strings.Split(id, "/")
	if len(parts) == 1 {
		// Top-level snippet (e.g., "base")
		result["core"] = append(result["core"], id)
	} else {
		category := parts[0]
		result[category] = append(result[category], id)
	}
}

// listLocalSnippets walks a local directory and adds found snippets to result
func listLocalSnippets(dir string, result map[string][]string, seen map[string]bool) {
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		// Convert path to snippet ID
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		id := strings.TrimSuffix(rel, ".yaml")
		addSnippetToResult(result, seen, id)
		return nil
	})
}

// ListAll returns all available snippet IDs organized by category.
// It includes snippets from:
// 1. Project-local: .glovebox/snippets/
// 2. User global: ~/.glovebox/snippets/
// 3. Embedded snippets (bundled in binary)
// Local snippets take precedence and can override embedded ones.
func ListAll() (map[string][]string, error) {
	result := make(map[string][]string)
	seen := make(map[string]bool)

	// Check local filesystem paths first (they take precedence)
	for _, searchPath := range snippetSearchPaths() {
		listLocalSnippets(searchPath, result, seen)
	}

	// Add embedded snippets (if not already seen)
	err := fs.WalkDir(snippetFS, "snippets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		// Convert path like "snippets/shells/fish.yaml" to "shells/fish"
		rel := strings.TrimPrefix(path, "snippets/")
		id := strings.TrimSuffix(rel, ".yaml")
		addSnippetToResult(result, seen, id)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("listing snippets: %w", err)
	}

	return result, nil
}

// LoadMultiple loads multiple snippets by their IDs and resolves dependencies
func LoadMultiple(ids []string) ([]*Snippet, error) {
	return LoadMultipleExcluding(ids, nil)
}

// LoadMultipleExcluding loads multiple snippets by their IDs and resolves dependencies,
// but excludes any snippets (and their dependencies) that are already satisfied by
// the provided base snippet IDs. This is used for project builds where the base image
// already contains certain snippets.
func LoadMultipleExcluding(ids []string, baseSnippetIDs []string) ([]*Snippet, error) {
	// Build a set of snippet IDs that are already in the base (including their dependencies)
	baseSatisfied := make(map[string]bool)
	if len(baseSnippetIDs) > 0 {
		// Resolve all base snippet IDs including their dependencies
		allBaseIDs, err := resolveAllDependencies(baseSnippetIDs)
		if err != nil {
			return nil, fmt.Errorf("resolving base snippets: %w", err)
		}
		for _, id := range allBaseIDs {
			baseSatisfied[id] = true
		}
	}

	return loadMultipleInternal(ids, baseSatisfied)
}

// loadMultipleInternal is the core implementation that loads snippets with dependency
// resolution, optionally skipping snippets that are already satisfied.
func loadMultipleInternal(ids []string, satisfied map[string]bool) ([]*Snippet, error) {
	loaded := make(map[string]*Snippet)
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

		s, err := Load(id)
		if err != nil {
			return err
		}

		// Load dependencies first
		for _, dep := range s.Requires {
			if err := loadWithDeps(dep); err != nil {
				return fmt.Errorf("dependency %s of %s: %w", dep, id, err)
			}
		}

		loaded[id] = s
		order = append(order, id)
		return nil
	}

	for _, id := range ids {
		if err := loadWithDeps(id); err != nil {
			return nil, err
		}
	}

	// Return snippets in dependency order
	result := make([]*Snippet, len(order))
	for i, id := range order {
		result[i] = loaded[id]
	}

	return result, nil
}

// resolveAllDependencies returns a list of all snippet IDs (including the given IDs
// and all their transitive dependencies) in dependency order.
func resolveAllDependencies(ids []string) ([]string, error) {
	resolved := make(map[string]bool)
	var order []string

	var resolve func(id string) error
	resolve = func(id string) error {
		if resolved[id] {
			return nil
		}

		s, err := Load(id)
		if err != nil {
			return err
		}

		// Resolve dependencies first
		for _, dep := range s.Requires {
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
