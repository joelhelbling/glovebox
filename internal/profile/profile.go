package profile

import (
	"crypto/sha256"
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
	ImageName        string    `yaml:"image_name,omitempty"`
	BaseDigest       string    `yaml:"base_digest,omitempty"`  // For project profiles, tracks when base changed
	ContentHash      string    `yaml:"content_hash,omitempty"` // Hash of mods list to detect manual edits
}

// Profile represents a glovebox configuration
type Profile struct {
	Version        int       `yaml:"version"`
	Mods           []string  `yaml:"mods"`
	PassthroughEnv []string  `yaml:"passthrough_env,omitempty"`
	Build          BuildInfo `yaml:"build,omitempty"`

	// Path is not serialized - it's the location this profile was loaded from
	Path string `yaml:"-"`
	// IsGlobal indicates if this is the global (base) profile
	IsGlobal bool `yaml:"-"`
}

// NewProfile creates a new empty profile
func NewProfile() *Profile {
	return &Profile{
		Version: 1,
		Mods:    []string{},
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

	// Determine if this is a global profile
	globalPath, _ := GlobalPath()
	p.IsGlobal = (path == globalPath)

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

// AddMod adds a mod to the profile if not already present
func (p *Profile) AddMod(id string) bool {
	for _, m := range p.Mods {
		if m == id {
			return false // Already present
		}
	}
	p.Mods = append(p.Mods, id)
	return true
}

// RemoveMod removes a mod from the profile
func (p *Profile) RemoveMod(id string) bool {
	for i, m := range p.Mods {
		if m == id {
			p.Mods = append(p.Mods[:i], p.Mods[i+1:]...)
			return true
		}
	}
	return false
}

// HasMod checks if a mod is in the profile
func (p *Profile) HasMod(id string) bool {
	for _, m := range p.Mods {
		if m == id {
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

// ComputeContentHash computes a hash of the user-editable content (mods list)
func (p *Profile) ComputeContentHash() string {
	// Create a stable representation of the content
	content := fmt.Sprintf("v%d:%v:%v", p.Version, p.Mods, p.PassthroughEnv)
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)[:12] // Short hash is sufficient
}

// UpdateContentHash computes and stores the content hash
func (p *Profile) UpdateContentHash() {
	p.Build.ContentHash = p.ComputeContentHash()
}

// WasManuallyEdited checks if the profile content differs from the stored hash.
// Returns true if the content was edited after glovebox generated it.
// Returns false if no hash is stored (legacy profiles) or content matches.
func (p *Profile) WasManuallyEdited() bool {
	if p.Build.ContentHash == "" {
		return false // No hash stored, assume not edited (legacy profile)
	}
	return p.ComputeContentHash() != p.Build.ContentHash
}

// ImageName returns the Docker image name for this profile
func (p *Profile) ImageName() string {
	if p.Build.ImageName != "" {
		return p.Build.ImageName
	}

	if p.IsGlobal {
		return "glovebox:base"
	}

	// Generate project image name from directory
	dir := filepath.Dir(filepath.Dir(p.Path)) // Go up from .glovebox/profile.yaml
	return GenerateImageName(dir)
}

// GenerateImageName creates a Docker image name from a directory path
// Format: glovebox:<dirname>-<shorthash>
func GenerateImageName(dir string) string {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		absPath = dir
	}

	dirName := filepath.Base(absPath)
	hash := sha256.Sum256([]byte(absPath))
	shortHash := fmt.Sprintf("%x", hash)[:7]

	return fmt.Sprintf("glovebox:%s-%s", dirName, shortHash)
}

// GlobalDir returns the global glovebox directory path
func GlobalDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, GlobalProfileDir), nil
}

// ProjectDir returns the project glovebox directory path
func ProjectDir(dir string) string {
	return filepath.Join(dir, ProjectProfileDir)
}

// DockerfilePath returns the path where the Dockerfile should be generated
func (p *Profile) DockerfilePath() string {
	if p.IsGlobal {
		globalDir, _ := GlobalDir()
		return filepath.Join(globalDir, "Dockerfile")
	}
	// Project Dockerfile lives in .glovebox/Dockerfile
	return filepath.Join(filepath.Dir(p.Path), "Dockerfile")
}

// LoadGlobal loads the global profile (for base image)
func LoadGlobal() (*Profile, error) {
	globalPath, err := GlobalPath()
	if err != nil {
		return nil, fmt.Errorf("loading global profile: %w", err)
	}
	return Load(globalPath)
}

// LoadProject loads the project profile from a directory
func LoadProject(dir string) (*Profile, error) {
	projectPath := ProjectPath(dir)
	return Load(projectPath)
}

// EffectivePassthroughEnv returns the combined passthrough env vars from both
// global and project profiles. Project profile vars are appended to global ones,
// with duplicates removed (project takes precedence).
func EffectivePassthroughEnv(projectDir string) ([]string, error) {
	seen := make(map[string]bool)
	var result []string

	// Load global profile first
	globalProfile, err := LoadGlobal()
	if err != nil {
		return nil, fmt.Errorf("loading global profile: %w", err)
	}
	if globalProfile != nil {
		for _, env := range globalProfile.PassthroughEnv {
			if !seen[env] {
				seen[env] = true
				result = append(result, env)
			}
		}
	}

	// Load project profile and add its vars (deduped)
	projectProfile, err := LoadProject(projectDir)
	if err != nil {
		return nil, fmt.Errorf("loading project profile: %w", err)
	}
	if projectProfile != nil {
		for _, env := range projectProfile.PassthroughEnv {
			if !seen[env] {
				seen[env] = true
				result = append(result, env)
			}
		}
	}

	return result, nil
}
