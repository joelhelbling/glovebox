package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	GlobalProfileDir  = ".glovebox"
	ProfileFileName   = "profile.yaml"
	ProjectProfileDir = ".glovebox"
)

// BuildInfo tracks when and how the Dockerfile was generated
type BuildInfo struct {
	LastBuiltAt      time.Time `yaml:"last_built_at,omitempty"`
	DockerfileDigest string    `yaml:"dockerfile_digest,omitempty"`
}

// Profile represents a glovebox configuration
type Profile struct {
	Version  int       `yaml:"version"`
	Snippets []string  `yaml:"snippets"`
	Build    BuildInfo `yaml:"build,omitempty"`

	// Path is not serialized - it's the location this profile was loaded from
	Path string `yaml:"-"`
}

// NewProfile creates a new empty profile
func NewProfile() *Profile {
	return &Profile{
		Version:  1,
		Snippets: []string{},
	}
}

// GlobalPath returns the path to the global profile
func GlobalPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, GlobalProfileDir, ProfileFileName), nil
}

// ProjectPath returns the path to the project profile in the given directory
func ProjectPath(dir string) string {
	return filepath.Join(dir, ProjectProfileDir, ProfileFileName)
}

// Load reads a profile from the given path
func Load(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No profile at this path
		}
		return nil, fmt.Errorf("reading profile: %w", err)
	}

	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing profile: %w", err)
	}

	p.Path = path
	return &p, nil
}

// LoadEffective loads the effective profile for the current context.
// It checks for a project profile first, then falls back to global.
// Returns nil if no profile exists.
func LoadEffective(projectDir string) (*Profile, error) {
	// Check for project profile first
	projectPath := ProjectPath(projectDir)
	if p, err := Load(projectPath); err != nil {
		return nil, err
	} else if p != nil {
		return p, nil
	}

	// Fall back to global profile
	globalPath, err := GlobalPath()
	if err != nil {
		return nil, err
	}

	return Load(globalPath)
}

// Save writes the profile to its path
func (p *Profile) Save() error {
	if p.Path == "" {
		return fmt.Errorf("profile has no path set")
	}

	// Ensure directory exists
	dir := filepath.Dir(p.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating profile directory: %w", err)
	}

	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("serializing profile: %w", err)
	}

	if err := os.WriteFile(p.Path, data, 0644); err != nil {
		return fmt.Errorf("writing profile: %w", err)
	}

	return nil
}

// SaveTo writes the profile to a specific path and updates p.Path
func (p *Profile) SaveTo(path string) error {
	p.Path = path
	return p.Save()
}

// AddSnippet adds a snippet to the profile if not already present
func (p *Profile) AddSnippet(id string) bool {
	for _, s := range p.Snippets {
		if s == id {
			return false // Already present
		}
	}
	p.Snippets = append(p.Snippets, id)
	return true
}

// RemoveSnippet removes a snippet from the profile
func (p *Profile) RemoveSnippet(id string) bool {
	for i, s := range p.Snippets {
		if s == id {
			p.Snippets = append(p.Snippets[:i], p.Snippets[i+1:]...)
			return true
		}
	}
	return false
}

// HasSnippet checks if a snippet is in the profile
func (p *Profile) HasSnippet(id string) bool {
	for _, s := range p.Snippets {
		if s == id {
			return true
		}
	}
	return false
}

// UpdateBuildInfo updates the build metadata
func (p *Profile) UpdateBuildInfo(digest string) {
	p.Build.LastBuiltAt = time.Now().UTC()
	p.Build.DockerfileDigest = digest
}
