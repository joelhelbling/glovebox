package snippet

import (
	"embed"
	"fmt"
	"io/fs"
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

// Load reads a snippet from the embedded filesystem by its ID (e.g., "shells/fish")
func Load(id string) (*Snippet, error) {
	path := filepath.Join("snippets", id+".yaml")
	data, err := snippetFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("snippet not found: %s", id)
	}

	var s Snippet
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing snippet %s: %w", id, err)
	}

	return &s, nil
}

// ListAll returns all available snippet IDs organized by category
func ListAll() (map[string][]string, error) {
	result := make(map[string][]string)

	err := fs.WalkDir(snippetFS, "snippets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		// Convert path like "snippets/shells/fish.yaml" to "shells/fish"
		rel := strings.TrimPrefix(path, "snippets/")
		rel = strings.TrimSuffix(rel, ".yaml")

		// Extract category from path
		parts := strings.Split(rel, "/")
		if len(parts) == 1 {
			// Top-level snippet (e.g., "base")
			result["core"] = append(result["core"], rel)
		} else {
			category := parts[0]
			result[category] = append(result[category], rel)
		}

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
