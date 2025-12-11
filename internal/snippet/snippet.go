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

// Snippet represents a composable piece of Dockerfile configuration
type Snippet struct {
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
	loaded := make(map[string]*Snippet)
	var order []string

	var loadWithDeps func(id string) error
	loadWithDeps = func(id string) error {
		if _, exists := loaded[id]; exists {
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
