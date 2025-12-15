//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/joelhelbling/glovebox/internal/mod"
)

// TestModCompatibility verifies that our compatibility calculation is working
func TestModCompatibility(t *testing.T) {
	for _, osName := range mod.KnownOSNames {
		t.Run(osName, func(t *testing.T) {
			mods, err := ModsCompatibleWithOS(osName)
			if err != nil {
				t.Fatalf("Failed to get compatible mods for %s: %v", osName, err)
			}

			if len(mods) == 0 {
				t.Errorf("Expected at least some mods compatible with %s, got none", osName)
			}

			t.Logf("Found %d mods compatible with %s:", len(mods), osName)
			sort.Strings(mods)
			for _, m := range mods {
				t.Logf("  - %s", m)
			}
		})
	}
}

// TestDependencyResolution verifies that dependencies are correctly resolved
func TestDependencyResolution(t *testing.T) {
	// Test that gemini-cli pulls in nodejs
	deps, err := ResolveDependencies("ai/gemini-cli", "ubuntu")
	if err != nil {
		t.Fatalf("Failed to resolve dependencies: %v", err)
	}

	t.Logf("Dependencies for ai/gemini-cli on ubuntu: %v", deps)

	// Should include nodejs-ubuntu (provides nodejs which gemini-cli requires)
	hasNodejs := false
	hasGeminiCli := false
	for _, dep := range deps {
		if dep == "languages/nodejs-ubuntu" {
			hasNodejs = true
		}
		if dep == "ai/gemini-cli" {
			hasGeminiCli = true
		}
	}

	if !hasNodejs {
		t.Error("Expected nodejs-ubuntu to be included as dependency")
	}
	if !hasGeminiCli {
		t.Error("Expected gemini-cli to be included")
	}

	// Test that claude-code pulls in bash
	deps2, err := ResolveDependencies("ai/claude-code", "alpine")
	if err != nil {
		t.Fatalf("Failed to resolve dependencies: %v", err)
	}

	t.Logf("Dependencies for ai/claude-code on alpine: %v", deps2)

	hasBash := false
	for _, dep := range deps2 {
		if dep == "shells/bash" {
			hasBash = true
		}
	}

	if !hasBash {
		t.Error("Expected bash to be included as dependency for claude-code")
	}
}

// TestModBuilds tests that mods can be built successfully for each OS.
// It creates temporary profiles and runs `glovebox build --base` for each combination.
func TestModBuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping build tests in short mode")
	}

	// Find the glovebox binary
	gloveboxBin := findGloveboxBinary(t)

	for _, osName := range mod.KnownOSNames {
		osName := osName // capture for parallel
		t.Run(osName, func(t *testing.T) {
			// Get mods compatible with this OS
			compatibleMods, err := LeafModsForOS(osName)
			if err != nil {
				t.Fatalf("Failed to get compatible mods: %v", err)
			}

			t.Logf("Testing %d mods on %s", len(compatibleMods), osName)

			for _, modID := range compatibleMods {
				modID := modID // capture for parallel
				testName := strings.ReplaceAll(modID, "/", "-")

				t.Run(testName, func(t *testing.T) {
					// Create a temp directory for this test
					tmpDir, err := os.MkdirTemp("", fmt.Sprintf("glovebox-test-%s-%s-*", osName, testName))
					if err != nil {
						t.Fatalf("Failed to create temp dir: %v", err)
					}
					defer os.RemoveAll(tmpDir)

					// Create .glovebox directory and profile
					gloveboxDir := filepath.Join(tmpDir, ".glovebox")
					if err := os.MkdirAll(gloveboxDir, 0755); err != nil {
						t.Fatalf("Failed to create .glovebox dir: %v", err)
					}

					// Resolve all dependencies for this mod
					deps, err := ResolveDependencies(modID, osName)
					if err != nil {
						t.Fatalf("Failed to resolve dependencies for %s: %v", modID, err)
					}

					// Build profile with OS and all required mods
					// Use a unique image name to avoid clobbering user's real base image
					testImageName := fmt.Sprintf("glovebox-test:%s-%s", osName, testName)

					var modsYaml strings.Builder
					modsYaml.WriteString(fmt.Sprintf("  - os/%s\n", osName))
					for _, dep := range deps {
						modsYaml.WriteString(fmt.Sprintf("  - %s\n", dep))
					}

					profileContent := fmt.Sprintf("version: 1\nmods:\n%s\nbuild:\n  image_name: %s\n", modsYaml.String(), testImageName)

					profilePath := filepath.Join(gloveboxDir, "profile.yaml")
					if err := os.WriteFile(profilePath, []byte(profileContent), 0644); err != nil {
						t.Fatalf("Failed to write profile: %v", err)
					}

					t.Logf("Testing %s with dependencies: %v", modID, deps)

					// Run glovebox build --base from the temp directory
					// We use --base because we're testing base image builds, not project builds
					cmd := exec.Command(gloveboxBin, "build", "--base")
					cmd.Dir = tmpDir
					cmd.Env = append(os.Environ(),
						fmt.Sprintf("HOME=%s", tmpDir), // Use temp dir as HOME so it finds our profile
					)

					output, err := cmd.CombinedOutput()
					if err != nil {
						t.Errorf("Build failed for %s on %s:\n%s\nError: %v", modID, osName, string(output), err)
						return
					}

					t.Logf("Successfully built %s on %s", modID, osName)

					// Clean up the Docker image
					cleanupImage(t, testImageName)
				})
			}
		})
	}
}

// TestSingleModBuild is a helper for testing a specific mod during development
func TestSingleModBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping build tests in short mode")
	}

	// Set these via environment variables for targeted testing
	osName := os.Getenv("TEST_OS")
	modID := os.Getenv("TEST_MOD")

	if osName == "" || modID == "" {
		t.Skip("Set TEST_OS and TEST_MOD environment variables to run this test")
	}

	gloveboxBin := findGloveboxBinary(t)

	tmpDir, err := os.MkdirTemp("", "glovebox-test-single-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gloveboxDir := filepath.Join(tmpDir, ".glovebox")
	if err := os.MkdirAll(gloveboxDir, 0755); err != nil {
		t.Fatalf("Failed to create .glovebox dir: %v", err)
	}

	// Resolve all dependencies for this mod
	deps, err := ResolveDependencies(modID, osName)
	if err != nil {
		t.Fatalf("Failed to resolve dependencies for %s: %v", modID, err)
	}

	// Build profile with OS and all required mods
	// Use a unique image name to avoid clobbering user's real base image
	testName := strings.ReplaceAll(modID, "/", "-")
	testImageName := fmt.Sprintf("glovebox-test:%s-%s", osName, testName)

	var modsYaml strings.Builder
	modsYaml.WriteString(fmt.Sprintf("  - os/%s\n", osName))
	for _, dep := range deps {
		modsYaml.WriteString(fmt.Sprintf("  - %s\n", dep))
	}

	profileContent := fmt.Sprintf("version: 1\nmods:\n%s\nbuild:\n  image_name: %s\n", modsYaml.String(), testImageName)
	t.Logf("Testing %s with dependencies: %v (image: %s)", modID, deps, testImageName)

	profilePath := filepath.Join(gloveboxDir, "profile.yaml")
	if err := os.WriteFile(profilePath, []byte(profileContent), 0644); err != nil {
		t.Fatalf("Failed to write profile: %v", err)
	}

	cmd := exec.Command(gloveboxBin, "build", "--base")
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("HOME=%s", tmpDir),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Build failed:\n%s\nError: %v", string(output), err)
	}

	t.Logf("Build output:\n%s", string(output))

	cleanupImage(t, testImageName)
}

// findGloveboxBinary locates the glovebox binary to use for tests
func findGloveboxBinary(t *testing.T) string {
	t.Helper()

	// Check for binary in common locations
	candidates := []string{
		"./bin/glovebox",                                         // Built binary in project
		filepath.Join(os.Getenv("GOPATH"), "bin", "glovebox"),    // Installed via go install
		"/usr/local/bin/glovebox",                                // System-wide install
	}

	// Also check relative to test directory
	if cwd, err := os.Getwd(); err == nil {
		// Walk up to find project root
		for dir := cwd; dir != "/"; dir = filepath.Dir(dir) {
			candidate := filepath.Join(dir, "bin", "glovebox")
			candidates = append([]string{candidate}, candidates...)
		}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			absPath, _ := filepath.Abs(candidate)
			t.Logf("Using glovebox binary: %s", absPath)
			return absPath
		}
	}

	t.Fatal("Could not find glovebox binary. Run 'make build' first.")
	return ""
}

// cleanupImage removes a Docker image
func cleanupImage(t *testing.T, imageName string) {
	t.Helper()

	cmd := exec.Command("docker", "rmi", "-f", imageName)
	if err := cmd.Run(); err != nil {
		// Don't fail test on cleanup errors
		t.Logf("Warning: failed to remove image %s: %v", imageName, err)
	}
}
