// Package integration contains integration tests that verify the interaction
// between base (global) and project profiles in glovebox.
//
// These tests document and enforce the following behavioral assumptions:
//
//  1. Passthrough variables defined in base are passed through to all glovebox containers
//  2. Passthrough variables defined in project are passed through only to that project's container
//  3. Mods selected in the base are installed in the base image
//  4. Mods selected in the project which are not in the base are installed in the project
//  5. Mods selected in the project which are also in the base are NOT installed in the project
package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joelhelbling/glovebox/internal/docker"
	"github.com/joelhelbling/glovebox/internal/generator"
	"github.com/joelhelbling/glovebox/internal/mod"
	"github.com/joelhelbling/glovebox/internal/profile"
)

// TestPassthroughEnvFromBase verifies that passthrough variables defined in the
// global (base) profile are included when running any glovebox container.
//
// Assumption #1: Passthrough variables defined in base are passed through to all containers
func TestPassthroughEnvFromBase(t *testing.T) {
	// Create temp directories for global and project profiles
	tmpHome := t.TempDir()
	tmpProject := t.TempDir()

	// Create global profile with passthrough env vars
	globalProfilePath := filepath.Join(tmpHome, ".glovebox", "profile.yaml")
	globalProfile := profile.NewProfile()
	globalProfile.PassthroughEnv = []string{"GLOBAL_API_KEY", "GLOBAL_SECRET"}
	if err := globalProfile.SaveTo(globalProfilePath); err != nil {
		t.Fatalf("failed to save global profile: %v", err)
	}

	// Create project profile with its own passthrough env vars
	projectProfilePath := profile.ProjectPath(tmpProject)
	projectProfile := profile.NewProfile()
	projectProfile.PassthroughEnv = []string{"PROJECT_TOKEN"}
	if err := projectProfile.SaveTo(projectProfilePath); err != nil {
		t.Fatalf("failed to save project profile: %v", err)
	}

	// Override global path for testing
	origGlobalPath := overrideGlobalPath(t, globalProfilePath)
	defer restoreGlobalPath(origGlobalPath)

	// Get effective passthrough env
	result, err := profile.EffectivePassthroughEnv(tmpProject)
	if err != nil {
		t.Fatalf("EffectivePassthroughEnv() error: %v", err)
	}

	// Verify global vars are included
	if !containsString(result, "GLOBAL_API_KEY") {
		t.Error("expected GLOBAL_API_KEY from base profile to be included")
	}
	if !containsString(result, "GLOBAL_SECRET") {
		t.Error("expected GLOBAL_SECRET from base profile to be included")
	}
}

// TestPassthroughEnvFromProject verifies that passthrough variables defined in
// a project profile are included when running that project's container.
//
// Assumption #2: Passthrough variables defined in project are passed through to that project
func TestPassthroughEnvFromProject(t *testing.T) {
	tmpProject := t.TempDir()

	// Create project profile with passthrough env vars
	projectProfilePath := profile.ProjectPath(tmpProject)
	projectProfile := profile.NewProfile()
	projectProfile.PassthroughEnv = []string{"PROJECT_VAR1", "PROJECT_VAR2"}
	if err := projectProfile.SaveTo(projectProfilePath); err != nil {
		t.Fatalf("failed to save project profile: %v", err)
	}

	// Get effective passthrough env (may include global vars if they exist)
	result, err := profile.EffectivePassthroughEnv(tmpProject)
	if err != nil {
		t.Fatalf("EffectivePassthroughEnv() error: %v", err)
	}

	// Verify project vars are included
	if !containsString(result, "PROJECT_VAR1") {
		t.Error("expected PROJECT_VAR1 from project profile to be included")
	}
	if !containsString(result, "PROJECT_VAR2") {
		t.Error("expected PROJECT_VAR2 from project profile to be included")
	}
}

// TestPassthroughEnvMerging verifies that both base and project passthrough
// variables are merged, with deduplication.
func TestPassthroughEnvMerging(t *testing.T) {
	tmpHome := t.TempDir()
	tmpProject := t.TempDir()

	// Create global profile
	globalProfilePath := filepath.Join(tmpHome, ".glovebox", "profile.yaml")
	globalProfile := profile.NewProfile()
	globalProfile.PassthroughEnv = []string{"SHARED_VAR", "GLOBAL_ONLY"}
	if err := globalProfile.SaveTo(globalProfilePath); err != nil {
		t.Fatalf("failed to save global profile: %v", err)
	}

	// Create project profile with overlapping var
	projectProfilePath := profile.ProjectPath(tmpProject)
	projectProfile := profile.NewProfile()
	projectProfile.PassthroughEnv = []string{"SHARED_VAR", "PROJECT_ONLY"}
	if err := projectProfile.SaveTo(projectProfilePath); err != nil {
		t.Fatalf("failed to save project profile: %v", err)
	}

	// Override global path
	origGlobalPath := overrideGlobalPath(t, globalProfilePath)
	defer restoreGlobalPath(origGlobalPath)

	result, err := profile.EffectivePassthroughEnv(tmpProject)
	if err != nil {
		t.Fatalf("EffectivePassthroughEnv() error: %v", err)
	}

	// Count occurrences of SHARED_VAR - should only appear once
	count := 0
	for _, v := range result {
		if v == "SHARED_VAR" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("SHARED_VAR should appear exactly once (deduped), got %d occurrences", count)
	}

	// Both unique vars should be present
	if !containsString(result, "GLOBAL_ONLY") {
		t.Error("expected GLOBAL_ONLY from base profile")
	}
	if !containsString(result, "PROJECT_ONLY") {
		t.Error("expected PROJECT_ONLY from project profile")
	}
}

// TestPassthroughEnvInDockerArgs verifies that passthrough env vars are correctly
// translated into docker run -e arguments.
func TestPassthroughEnvInDockerArgs(t *testing.T) {
	// Create a mock environment lookup
	mockEnv := map[string]string{
		"API_KEY":   "secret123",
		"OTHER_VAR": "value456",
		"UNSET_VAR": "", // This one is "unset"
	}
	envLookup := func(key string) string {
		return mockEnv[key]
	}

	result := docker.BuildRunArgs(docker.RunArgsConfig{
		ContainerName:  "test-container",
		ImageName:      "test-image",
		HostPath:       "/host/path",
		WorkspacePath:  "/workspace",
		PassthroughEnv: []string{"API_KEY", "OTHER_VAR", "UNSET_VAR"},
		EnvLookup:      envLookup,
	})

	// Check that set vars are passed through
	if !containsString(result.PassedVars, "API_KEY") {
		t.Error("expected API_KEY to be in PassedVars")
	}
	if !containsString(result.PassedVars, "OTHER_VAR") {
		t.Error("expected OTHER_VAR to be in PassedVars")
	}

	// Check that unset var is in MissingVars
	if !containsString(result.MissingVars, "UNSET_VAR") {
		t.Error("expected UNSET_VAR to be in MissingVars")
	}

	// Check actual docker args contain the -e flags
	argsStr := strings.Join(result.Args, " ")
	if !strings.Contains(argsStr, "-e API_KEY=secret123") {
		t.Error("expected docker args to contain '-e API_KEY=secret123'")
	}
	if !strings.Contains(argsStr, "-e OTHER_VAR=value456") {
		t.Error("expected docker args to contain '-e OTHER_VAR=value456'")
	}
	// Unset var should NOT be in args
	if strings.Contains(argsStr, "UNSET_VAR") {
		t.Error("unset var should not appear in docker args")
	}
}

// TestBaseModsInBaseImage verifies that mods selected in the global profile
// are included in the generated base Dockerfile.
//
// Assumption #3: Mods selected in base are installed in the base image
func TestBaseModsInBaseImage(t *testing.T) {
	// Generate a base Dockerfile with specific mods
	// Note: os/ubuntu provides "base" so homebrew's requirement is satisfied
	baseMods := []string{"os/ubuntu", "shells/bash", "tools/homebrew"}

	dockerfile, err := generator.GenerateBase(baseMods)
	if err != nil {
		t.Fatalf("GenerateBase() error: %v", err)
	}

	// Verify it's a standalone image (FROM ubuntu, not FROM glovebox:base)
	if !strings.Contains(dockerfile, "FROM ubuntu:") {
		t.Error("base image should start FROM ubuntu, not extend another image")
	}

	// Verify OS mod content is included (curl is installed by os/ubuntu)
	if !strings.Contains(dockerfile, "curl") {
		t.Error("expected OS mod packages (curl) to be included")
	}

	// Verify homebrew setup is included
	if !strings.Contains(dockerfile, "homebrew") || !strings.Contains(dockerfile, "Homebrew") {
		t.Error("expected homebrew setup to be included")
	}

	// Verify the mods are listed in the header comment
	if !strings.Contains(dockerfile, "# Mods:") {
		t.Error("expected mods to be listed in header")
	}
}

// TestProjectModsNotInBase verifies that mods selected in the project profile
// which are NOT in the base are installed in the project image.
//
// Assumption #4: Mods in project but not in base are installed in project
func TestProjectModsNotInBase(t *testing.T) {
	// Base has: os/ubuntu (provides base), homebrew
	// Project has: mise (which requires homebrew, but homebrew is already in base)
	baseMods := []string{"os/ubuntu", "tools/homebrew"}
	projectMods := []string{"tools/mise"}

	dockerfile, err := generator.GenerateProject(projectMods, baseMods)
	if err != nil {
		t.Fatalf("GenerateProject() error: %v", err)
	}

	// Verify it extends the base image
	if !strings.Contains(dockerfile, "FROM glovebox:base") {
		t.Error("project image should extend FROM glovebox:base")
	}

	// Verify mise is included (it's project-only)
	if !strings.Contains(dockerfile, "mise") {
		t.Error("expected mise to be included in project Dockerfile")
	}
}

// TestOverlappingModsExcludedFromProject verifies that mods which are in BOTH
// the base and project profiles are NOT duplicated in the project image.
//
// Assumption #5: Mods in both base and project are NOT installed in project
func TestOverlappingModsExcludedFromProject(t *testing.T) {
	// Base has: os/ubuntu (provides base), homebrew
	// Project has: homebrew (duplicate), mise
	baseMods := []string{"os/ubuntu", "tools/homebrew"}
	projectMods := []string{"tools/homebrew", "tools/mise"}

	dockerfile, err := generator.GenerateProject(projectMods, baseMods)
	if err != nil {
		t.Fatalf("GenerateProject() error: %v", err)
	}

	// Verify homebrew is NOT reinstalled (it's already in base)
	// Look for homebrew-specific installation steps that would only appear
	// if homebrew were being installed (not just used)
	lines := strings.Split(dockerfile, "\n")
	for _, line := range lines {
		// The homebrew mod has a specific run_as_user that installs homebrew
		// This should NOT appear in project Dockerfile
		if strings.Contains(line, "NONINTERACTIVE=1") && strings.Contains(line, "install.sh") {
			t.Error("homebrew installation should not appear in project Dockerfile (already in base)")
		}
	}

	// Verify mise IS included
	if !strings.Contains(dockerfile, "mise") {
		t.Error("expected mise to be included in project Dockerfile")
	}
}

// TestModDependencyResolutionWithExclusions verifies that the mod loading
// correctly excludes mods that are already satisfied by the base.
func TestModDependencyResolutionWithExclusions(t *testing.T) {
	// mise requires homebrew which requires base (provided by os/ubuntu)
	// If os/ubuntu and homebrew are in baseModIDs, only mise should be loaded
	baseMods := []string{"os/ubuntu", "tools/homebrew"}
	projectMods := []string{"tools/mise"}

	mods, err := mod.LoadMultipleExcluding(projectMods, baseMods)
	if err != nil {
		t.Fatalf("LoadMultipleExcluding() error: %v", err)
	}

	// Should only have mise
	if len(mods) != 1 {
		names := make([]string, len(mods))
		for i, m := range mods {
			names[i] = m.Name
		}
		t.Errorf("expected exactly 1 mod (mise), got %d: %v", len(mods), names)
	}

	if len(mods) > 0 && mods[0].Name != "mise" {
		t.Errorf("expected mise, got %s", mods[0].Name)
	}

	// Verify ubuntu and homebrew are NOT included
	for _, m := range mods {
		if m.Name == "ubuntu" {
			t.Error("ubuntu should be excluded (already in base image)")
		}
		if m.Name == "homebrew" {
			t.Error("homebrew should be excluded (already in base image)")
		}
	}
}

// TestTransitiveDependenciesExcluded verifies that transitive dependencies
// of base mods are also excluded from project builds.
func TestTransitiveDependenciesExcluded(t *testing.T) {
	// If base has os/ubuntu and mise (which requires base),
	// then a project adding neovim-ubuntu (which also requires ubuntu)
	// should not re-include ubuntu
	baseMods := []string{"os/ubuntu", "tools/mise"}
	projectMods := []string{"editors/neovim-ubuntu"}

	mods, err := mod.LoadMultipleExcluding(projectMods, baseMods)
	if err != nil {
		t.Fatalf("LoadMultipleExcluding() error: %v", err)
	}

	// Should only have neovim-ubuntu (not ubuntu)
	for _, m := range mods {
		if m.Name == "ubuntu" {
			t.Error("ubuntu should be excluded (transitive dependency of base mods)")
		}
	}

	// neovim-ubuntu should be included
	found := false
	for _, m := range mods {
		if m.Name == "neovim-ubuntu" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected neovim-ubuntu to be included in project mods")
	}
}

// Helper functions

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// overrideGlobalPath is a test helper that temporarily overrides the global profile path.
// This is a workaround since we can't easily inject the path into EffectivePassthroughEnv.
// Returns the original value to restore later.
//
// Note: This approach works because the test creates files at the expected global path
// location within the test's temp directory structure. For a cleaner approach, consider
// refactoring profile.LoadGlobal to accept a path parameter or use an interface.
func overrideGlobalPath(t *testing.T, newPath string) string {
	t.Helper()

	// We can't easily override the global path without modifying the profile package.
	// Instead, we'll use a different approach: set HOME env var to our temp dir.
	origHome := os.Getenv("HOME")

	// Extract the temp home from the profile path (path is like /tmp/xxx/.glovebox/profile.yaml)
	tmpHome := filepath.Dir(filepath.Dir(newPath))
	os.Setenv("HOME", tmpHome)

	return origHome
}

func restoreGlobalPath(origHome string) {
	os.Setenv("HOME", origHome)
}
