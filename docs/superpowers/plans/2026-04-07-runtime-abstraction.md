# Runtime Abstraction Layer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce a `Runtime` interface so glovebox can use Apple Containers (preferred) or Docker (fallback), with extensibility for future runtimes.

**Architecture:** New `internal/runtime/` package defines a `Runtime` interface. Docker and Apple Containers each implement it. A `Detect()` function auto-selects the best available runtime at startup. All 28 direct `exec.Command("docker", ...)` calls in cmd/ files migrate to runtime method calls.

**Tech Stack:** Go 1.25, Cobra CLI, exec.Command for subprocess invocation

**Spec:** `docs/superpowers/specs/2026-04-06-runtime-abstraction-design.md`

---

## File Structure

**New files:**
- `internal/runtime/runtime.go` — Interface, types, sentinel errors
- `internal/runtime/docker.go` — Docker implementation of Runtime
- `internal/runtime/docker_test.go` — Unit tests for Docker runtime arg construction + exit code handling
- `internal/runtime/detect.go` — Runtime auto-detection and factory
- `internal/runtime/detect_test.go` — Detection logic tests with mocked LookPath

**Modified files:**
- `cmd/root.go` — Add `PersistentPreRunE` for runtime detection, `--runtime` flag, package-level `rt` variable
- `cmd/run.go` — Replace 6 direct docker functions with `rt` calls + capability checks
- `cmd/build.go` — Replace `runDockerBuild()` with `rt.BuildImage()`
- `cmd/clean.go` — Replace listing/removal functions with runtime calls
- `cmd/commit.go` — Replace with `rt.Commit()` + `rt.RemoveContainer()` + capability check
- `cmd/diff.go` — Replace with `rt.Diff()` + capability check
- `cmd/reset.go` — Replace with `rt.RemoveContainer()`
- `cmd/status.go` — Replace inspection/diff calls + capability check
- `internal/docker/docker.go` — Remove functions that moved to runtime package
- `internal/docker/docker_test.go` — Remove tests for moved functions, keep naming tests

---

## Task 1: Define the Runtime Interface and Types

**Files:**
- Create: `internal/runtime/runtime.go`

- [ ] **Step 1: Create the runtime package with interface and types**

```go
// Package runtime abstracts container runtimes (Docker, Apple Containers, etc.)
package runtime

import (
	"errors"
	"io"
)

// ErrNotSupported is returned when an operation is not supported by the runtime.
var ErrNotSupported = errors.New("operation not supported by this runtime")

// Runtime abstracts a container runtime.
type Runtime interface {
	// Name returns the human-readable runtime name (e.g., "Docker", "Apple Containers").
	Name() string

	// Image operations
	ImageExists(name string) bool
	GetImageDigest(name string) (string, error)
	BuildImage(dockerfilePath, contextDir, imageName string) error
	RemoveImage(name string) error
	ListImages(filterRef string) ([]string, error)

	// Container lifecycle
	ContainerExists(name string) bool
	ContainerRunning(name string) bool
	RunInteractive(cfg RunConfig) error
	StartInteractive(name string) error
	Attach(name string) error
	RemoveContainer(name string) error
	ForceRemoveContainer(name string) error
	ListContainers(filterName string, all bool) ([]ContainerInfo, error)

	// Container state inspection
	Diff(name string) ([]FileDiff, error)
	Commit(containerName, imageName string) error

	// Capabilities reports which optional features this runtime supports.
	Capabilities() Capabilities
}

// RunConfig holds the parameters for creating and running a new container.
type RunConfig struct {
	ContainerName string
	ImageName     string
	HostPath      string
	WorkspacePath string
	Env           map[string]string // Pre-resolved key=value pairs
	Hostname      string            // Docker: --hostname flag. Apple Containers: ignored (--name sets hostname).
}

// ContainerInfo represents a container returned by list operations.
type ContainerInfo struct {
	Name  string
	Image string
}

// FileDiff represents a single filesystem change in a container.
type FileDiff struct {
	ChangeType string // "A" (added), "C" (changed), "D" (deleted)
	Path       string
}

// Capabilities describes which optional features a runtime supports.
type Capabilities struct {
	SupportsDiff   bool
	SupportsCommit bool
	SupportsExport bool
}

// Stdio holds the I/O streams for interactive container operations.
type Stdio struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/joelhelbling/code/ai/glovebox && go build ./internal/runtime/`
Expected: Clean compilation, no errors

- [ ] **Step 3: Commit**

```bash
git add internal/runtime/runtime.go
git commit -m "feat: define Runtime interface and types for multi-runtime support"
```

---

## Task 2: Implement Docker Runtime

**Files:**
- Create: `internal/runtime/docker.go`
- Create: `internal/runtime/docker_test.go`

- [ ] **Step 1: Write tests for Docker runtime arg construction**

```go
package runtime

import (
	"strings"
	"testing"
)

func TestDockerRuntime_Name(t *testing.T) {
	rt := NewDocker(Stdio{})
	if rt.Name() != "Docker" {
		t.Errorf("expected 'Docker', got %q", rt.Name())
	}
}

func TestDockerRuntime_buildRunArgs(t *testing.T) {
	rt := NewDocker(Stdio{})

	t.Run("basic args", func(t *testing.T) {
		args := rt.buildRunArgs(RunConfig{
			ContainerName: "my-container",
			ImageName:     "my-image:latest",
			HostPath:      "/home/user/project",
			WorkspacePath: "/project",
			Hostname:      "glovebox",
		})

		argsStr := strings.Join(args, " ")
		for _, want := range []string{
			"run", "-it",
			"--name my-container",
			"-v /home/user/project:/project",
			"-w /project",
			"--hostname glovebox",
		} {
			if !strings.Contains(argsStr, want) {
				t.Errorf("expected %q in args, got: %s", want, argsStr)
			}
		}

		// Image should be last
		if args[len(args)-1] != "my-image:latest" {
			t.Errorf("expected image as last arg, got %q", args[len(args)-1])
		}
	})

	t.Run("env vars", func(t *testing.T) {
		args := rt.buildRunArgs(RunConfig{
			ContainerName: "test",
			ImageName:     "test:latest",
			HostPath:      "/path",
			WorkspacePath: "/workspace",
			Env:           map[string]string{"API_KEY": "secret", "FOO": "bar"},
		})

		argsStr := strings.Join(args, " ")
		if !strings.Contains(argsStr, "-e API_KEY=secret") {
			t.Error("expected API_KEY env var in args")
		}
		if !strings.Contains(argsStr, "-e FOO=bar") {
			t.Error("expected FOO env var in args")
		}
	})

	t.Run("empty hostname omitted", func(t *testing.T) {
		args := rt.buildRunArgs(RunConfig{
			ContainerName: "test",
			ImageName:     "test:latest",
			HostPath:      "/path",
			WorkspacePath: "/workspace",
			Hostname:      "",
		})

		argsStr := strings.Join(args, " ")
		if strings.Contains(argsStr, "--hostname") {
			t.Error("empty hostname should not produce --hostname flag")
		}
	})
}

func TestDockerRuntime_Capabilities(t *testing.T) {
	rt := NewDocker(Stdio{})
	caps := rt.Capabilities()

	if !caps.SupportsDiff {
		t.Error("Docker should support diff")
	}
	if !caps.SupportsCommit {
		t.Error("Docker should support commit")
	}
	if !caps.SupportsExport {
		t.Error("Docker should support export")
	}
}

func TestDockerRuntime_normalizeExitError(t *testing.T) {
	rt := NewDocker(Stdio{})

	t.Run("nil error passes through", func(t *testing.T) {
		if err := rt.normalizeExitError(nil); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/joelhelbling/code/ai/glovebox && go test ./internal/runtime/ -v -run TestDocker`
Expected: Compilation error — `NewDocker` not defined

- [ ] **Step 3: Implement the Docker runtime**

```go
package runtime

import (
	"fmt"
	"os/exec"
	"strings"
)

// DockerRuntime implements Runtime using the Docker CLI.
type DockerRuntime struct {
	io Stdio
}

// NewDocker creates a Docker runtime with the given I/O streams.
func NewDocker(io Stdio) *DockerRuntime {
	return &DockerRuntime{io: io}
}

func (d *DockerRuntime) Name() string { return "Docker" }

func (d *DockerRuntime) ImageExists(name string) bool {
	cmd := exec.Command("docker", "image", "inspect", name)
	return cmd.Run() == nil
}

func (d *DockerRuntime) GetImageDigest(name string) (string, error) {
	cmd := exec.Command("docker", "image", "inspect", "--format", "{{.Id}}", name)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (d *DockerRuntime) BuildImage(dockerfilePath, contextDir, imageName string) error {
	cmd := exec.Command("docker", "build", "-t", imageName, "-f", dockerfilePath, contextDir)
	cmd.Stdout = d.io.Stdout
	cmd.Stderr = d.io.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}
	return nil
}

func (d *DockerRuntime) RemoveImage(name string) error {
	return exec.Command("docker", "rmi", name).Run()
}

func (d *DockerRuntime) ListImages(filterRef string) ([]string, error) {
	cmd := exec.Command("docker", "images", "--filter", fmt.Sprintf("reference=%s", filterRef), "--format", "{{.Repository}}:{{.Tag}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var images []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			images = append(images, line)
		}
	}
	return images, nil
}

func (d *DockerRuntime) ContainerExists(name string) bool {
	cmd := exec.Command("docker", "container", "inspect", name)
	return cmd.Run() == nil
}

func (d *DockerRuntime) ContainerRunning(name string) bool {
	cmd := exec.Command("docker", "container", "inspect", "-f", "{{.State.Running}}", name)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

// buildRunArgs constructs the argument list for `docker run`.
func (d *DockerRuntime) buildRunArgs(cfg RunConfig) []string {
	args := []string{
		"run", "-it",
		"--name", cfg.ContainerName,
		"-v", fmt.Sprintf("%s:%s", cfg.HostPath, cfg.WorkspacePath),
		"-w", cfg.WorkspacePath,
	}

	if cfg.Hostname != "" {
		args = append(args, "--hostname", cfg.Hostname)
	}

	for key, val := range cfg.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, val))
	}

	args = append(args, cfg.ImageName)
	return args
}

func (d *DockerRuntime) RunInteractive(cfg RunConfig) error {
	args := d.buildRunArgs(cfg)
	cmd := exec.Command("docker", args...)
	cmd.Stdin = d.io.Stdin
	cmd.Stdout = d.io.Stdout
	cmd.Stderr = d.io.Stderr
	return d.normalizeExitError(cmd.Run())
}

func (d *DockerRuntime) StartInteractive(name string) error {
	cmd := exec.Command("docker", "start", "-ai", name)
	cmd.Stdin = d.io.Stdin
	cmd.Stdout = d.io.Stdout
	cmd.Stderr = d.io.Stderr
	return d.normalizeExitError(cmd.Run())
}

func (d *DockerRuntime) Attach(name string) error {
	cmd := exec.Command("docker", "attach", name)
	cmd.Stdin = d.io.Stdin
	cmd.Stdout = d.io.Stdout
	cmd.Stderr = d.io.Stderr
	return d.normalizeExitError(cmd.Run())
}

func (d *DockerRuntime) RemoveContainer(name string) error {
	return exec.Command("docker", "container", "rm", name).Run()
}

func (d *DockerRuntime) ForceRemoveContainer(name string) error {
	return exec.Command("docker", "container", "rm", "-f", name).Run()
}

func (d *DockerRuntime) ListContainers(filterName string, all bool) ([]ContainerInfo, error) {
	args := []string{"container", "ls"}
	if all {
		args = append(args, "-a")
	}
	if filterName != "" {
		args = append(args, "--filter", fmt.Sprintf("name=%s", filterName))
	}
	args = append(args, "--format", "{{.Names}}\t{{.Image}}")

	cmd := exec.Command("docker", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var containers []ContainerInfo
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			containers = append(containers, ContainerInfo{Name: parts[0], Image: parts[1]})
		}
	}
	return containers, nil
}

func (d *DockerRuntime) Diff(name string) ([]FileDiff, error) {
	cmd := exec.Command("docker", "diff", name)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var diffs []FileDiff
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			diffs = append(diffs, FileDiff{ChangeType: parts[0], Path: parts[1]})
		}
	}
	return diffs, nil
}

func (d *DockerRuntime) Commit(containerName, imageName string) error {
	return exec.Command("docker", "commit", containerName, imageName).Run()
}

func (d *DockerRuntime) Capabilities() Capabilities {
	return Capabilities{
		SupportsDiff:   true,
		SupportsCommit: true,
		SupportsExport: true,
	}
}

// normalizeExitError filters out normal container exit codes while preserving
// Docker-specific errors that indicate real problems.
//
// Exit codes:
//   - 125: Docker daemon error (failed to create/start container)
//   - 126: Command cannot be invoked (permission denied)
//   - 127: Command not found in container
//   - 137: Container killed by SIGKILL (often OOM killer)
//   - Other: Normal exit (including non-zero from last shell command)
func (d *DockerRuntime) normalizeExitError(err error) error {
	if err == nil {
		return nil
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return err
	}

	code := exitErr.ExitCode()
	switch {
	case code >= 125 && code <= 127:
		return fmt.Errorf("docker error (exit %d): %w", code, err)
	case code == 137:
		return fmt.Errorf("container was killed (exit 137, possibly out of memory)")
	default:
		return nil
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/joelhelbling/code/ai/glovebox && go test ./internal/runtime/ -v -run TestDocker`
Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/runtime/docker.go internal/runtime/docker_test.go
git commit -m "feat: implement Docker runtime wrapping all docker CLI operations"
```

---

## Task 3: Implement Runtime Detection

**Files:**
- Create: `internal/runtime/detect.go`
- Create: `internal/runtime/detect_test.go`

- [ ] **Step 1: Write tests for detection logic**

```go
package runtime

import (
	"testing"
)

func TestDetect_withOverride(t *testing.T) {
	t.Run("docker override", func(t *testing.T) {
		result, err := Detect("docker", Stdio{})
		if err != nil {
			// Docker may not be available in test env; skip if so
			t.Skipf("Docker not available: %v", err)
		}
		if result.Runtime.Name() != "Docker" {
			t.Errorf("expected Docker runtime, got %q", result.Runtime.Name())
		}
		if result.FellBack {
			t.Error("explicit override should not be a fallback")
		}
	})

	t.Run("unknown override errors", func(t *testing.T) {
		_, err := Detect("nonexistent-runtime", Stdio{})
		if err == nil {
			t.Error("expected error for unknown runtime")
		}
	})
}

func TestDetect_autoDetection(t *testing.T) {
	// Auto-detection with empty override
	result, err := Detect("", Stdio{})
	if err != nil {
		t.Skipf("No runtime available in test environment: %v", err)
	}
	// Should find at least one runtime
	if result.Runtime == nil {
		t.Error("expected a runtime to be detected")
	}
	if result.Runtime.Name() == "" {
		t.Error("expected runtime to have a name")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/joelhelbling/code/ai/glovebox && go test ./internal/runtime/ -v -run TestDetect`
Expected: Compilation error — `Detect` not defined

- [ ] **Step 3: Implement the detection function**

```go
package runtime

import (
	"fmt"
	"os/exec"
)

// DetectResult holds the outcome of runtime auto-detection.
type DetectResult struct {
	Runtime     Runtime
	FellBack    bool
	FallbackMsg string
}

// Detect selects the best available container runtime.
// If override is non-empty, that specific runtime is required.
// Otherwise, prefers Apple Containers, falls back to Docker.
func Detect(override string, io Stdio) (DetectResult, error) {
	if override != "" {
		return detectOverride(override, io)
	}
	return detectAuto(io)
}

func detectOverride(name string, io Stdio) (DetectResult, error) {
	switch name {
	case "docker":
		if !dockerAvailable() {
			return DetectResult{}, fmt.Errorf("Docker not available: ensure 'docker' is installed and the daemon is running")
		}
		return DetectResult{Runtime: NewDocker(io)}, nil
	// Apple Containers override will be added in Phase 4
	default:
		return DetectResult{}, fmt.Errorf("unknown runtime %q (available: docker)", name)
	}
}

func detectAuto(io Stdio) (DetectResult, error) {
	// Phase 4 will add Apple Containers detection here, before Docker.

	if dockerAvailable() {
		return DetectResult{Runtime: NewDocker(io)}, nil
	}

	return DetectResult{}, fmt.Errorf("no container runtime found\nInstall Docker: https://docs.docker.com/get-docker/")
}

// dockerAvailable checks if the docker CLI exists and the daemon is responsive.
func dockerAvailable() bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	// Verify daemon is running
	return exec.Command("docker", "info").Run() == nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/joelhelbling/code/ai/glovebox && go test ./internal/runtime/ -v -run TestDetect`
Expected: PASS (or SKIP if no Docker in CI)

- [ ] **Step 5: Commit**

```bash
git add internal/runtime/detect.go internal/runtime/detect_test.go
git commit -m "feat: add runtime auto-detection with Docker support"
```

---

## Task 4: Wire Runtime Into cmd/root.go

**Files:**
- Modify: `cmd/root.go`

- [ ] **Step 1: Add runtime initialization to root command**

Replace the entire contents of `cmd/root.go` with:

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/joelhelbling/glovebox/internal/runtime"
	"github.com/spf13/cobra"
)

var (
	// rt is the active container runtime, set during PersistentPreRunE.
	rt runtime.Runtime

	// runtimeOverride is set via the --runtime flag.
	runtimeOverride string
)

var rootCmd = &cobra.Command{
	Use:   "glovebox",
	Short: "A composable, sandboxed development environment",
	Long: `Glovebox creates sandboxed containers for running untrusted or
experimental code. It uses a mod-based system to compose your perfect
development environment from modular, reusable pieces.`,
	Version: Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip runtime detection for commands that don't need it
		switch cmd.Name() {
		case "help", "version", "init", "mod":
			return nil
		}
		// Also skip if this is a child of "mod" (e.g., "mod list")
		if cmd.Parent() != nil && cmd.Parent().Name() == "mod" {
			return nil
		}

		result, err := runtime.Detect(runtimeOverride, runtime.Stdio{
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		})
		if err != nil {
			return err
		}
		if result.FellBack {
			colorYellow.Println(result.FallbackMsg)
			fmt.Println()
		}
		rt = result.Runtime
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&runtimeOverride, "runtime", "", "Container runtime to use (e.g., docker)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/joelhelbling/code/ai/glovebox && go build ./...`
Expected: Clean compilation

- [ ] **Step 3: Run existing tests**

Run: `cd /Users/joelhelbling/code/ai/glovebox && make test`
Expected: All existing tests PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/root.go
git commit -m "feat: wire runtime detection into root command with --runtime flag"
```

---

## Task 5: Migrate cmd/run.go to Runtime Interface

**Files:**
- Modify: `cmd/run.go`

This is the largest migration. Replace all 6 direct docker functions with `rt` method calls.

- [ ] **Step 1: Replace docker imports and helper functions**

In `cmd/run.go`, remove the `"os/exec"` import (it will no longer be needed in this file after migration). Replace `"github.com/joelhelbling/glovebox/internal/docker"` with `"github.com/joelhelbling/glovebox/internal/runtime"` in the import block. Keep the `"github.com/joelhelbling/glovebox/internal/docker"` import for now (still needed for `docker.ContainerName` and `docker.ImageName`).

Replace `attachToContainer` (lines 146-152):

```go
// attachToContainer attaches to a running container
func attachToContainer(name string) error {
	return rt.Attach(name)
}
```

Replace `startContainer` (lines 187-194):

```go
// startContainer starts an existing stopped container
func startContainer(name, hostPath, workspacePath string) error {
	return rt.StartInteractive(name)
}
```

Replace `createAndStartContainerWithEnv` (lines 197-222):

```go
// createAndStartContainerWithEnv creates a new container with pre-computed env vars
func createAndStartContainerWithEnv(name, imageName, hostPath, workspacePath string, _ []string) error {
	// Get passthrough env config from profiles
	passthroughEnv, err := profile.EffectivePassthroughEnv(hostPath)
	if err != nil {
		// Non-fatal: continue without passthrough vars
		passthroughEnv = nil
	}

	// Build env map
	env := make(map[string]string)
	for _, envName := range passthroughEnv {
		if val := os.Getenv(envName); val != "" {
			env[envName] = val
		}
	}
	// Always add mise trusted config path
	env["MISE_TRUSTED_CONFIG_PATHS"] = fmt.Sprintf("%s:%s/**", workspacePath, workspacePath)

	return rt.RunInteractive(runtime.RunConfig{
		ContainerName: name,
		ImageName:     imageName,
		HostPath:      hostPath,
		WorkspacePath: workspacePath,
		Env:           env,
		Hostname:      "glovebox",
	})
}
```

Replace `getContainerDiff` (lines 246-261):

```go
// getContainerDiff returns the filesystem changes in a container
func getContainerDiff(name string) ([]string, error) {
	caps := rt.Capabilities()
	if !caps.SupportsDiff {
		return nil, nil
	}

	diffs, err := rt.Diff(name)
	if err != nil {
		return nil, err
	}

	var changes []string
	for _, d := range diffs {
		changes = append(changes, fmt.Sprintf("%s %s", d.ChangeType, d.Path))
	}
	return changes, nil
}
```

Replace `commitContainer` (lines 471-474):

```go
// commitContainer commits container changes to its image
func commitContainer(containerName, imageName string) error {
	return rt.Commit(containerName, imageName)
}
```

Replace `deleteContainer` (lines 477-480):

```go
// deleteContainer removes a container without printing
func deleteContainer(containerName string) error {
	return rt.RemoveContainer(containerName)
}
```

Also update `runRun` to replace the `docker.BuildRunArgs` call (lines 100-109) used only for computing `passthroughVars` for the banner. Replace it with direct env lookup:

```go
	var passthroughVars []string
	if !containerExists {
		passthroughEnv, err := profile.EffectivePassthroughEnv(absPath)
		if err != nil {
			colorYellow.Printf("Warning: could not load passthrough env: %v\n", err)
		} else {
			for _, env := range passthroughEnv {
				if os.Getenv(env) != "" {
					passthroughVars = append(passthroughVars, env)
				}
			}
		}
	}
```

Now remove the `"os/exec"` import since it is no longer used in this file. Keep `"github.com/joelhelbling/glovebox/internal/docker"` (still used for `docker.ContainerName`, `docker.ContainerExists`, `docker.ContainerRunning`).

Then replace the remaining `docker.ContainerExists` and `docker.ContainerRunning` calls (lines 74-75) with runtime calls:

```go
	containerExists := rt.ContainerExists(containerName)
	containerRunning := rt.ContainerRunning(containerName)
```

And the `docker.ImageExists` call inside `determineImage` (line 518):

```go
		if !rt.ImageExists(imageName) {
```

And line 530:

```go
	if !rt.ImageExists("glovebox:base") {
```

After all replacements, the only remaining `docker` import usage should be `docker.ContainerName` and `docker.ImageName`.

Remove the `ignoreExitError` function (lines 163-184) — this logic now lives inside `DockerRuntime.normalizeExitError`.

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/joelhelbling/code/ai/glovebox && go build ./...`
Expected: Clean compilation

- [ ] **Step 3: Run existing tests**

Run: `cd /Users/joelhelbling/code/ai/glovebox && make test`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/run.go
git commit -m "refactor: migrate cmd/run.go to runtime interface"
```

---

## Task 6: Migrate cmd/build.go to Runtime Interface

**Files:**
- Modify: `cmd/build.go`

- [ ] **Step 1: Replace `runDockerBuild` with runtime call**

Replace the `runDockerBuild` function (lines 325-344) with:

```go
func runDockerBuild(dockerfilePath, imageName string) error {
	fmt.Printf("\nBuilding image %s...\n", imageName)

	dockerfileDir := dockerfilePath[:len(dockerfilePath)-len("Dockerfile")]
	if dockerfileDir == "" {
		dockerfileDir = "."
	}

	if err := rt.BuildImage(dockerfilePath, dockerfileDir, imageName); err != nil {
		return fmt.Errorf("image build failed: %w", err)
	}

	colorGreen.Printf("\n✓ Image %s built successfully\n", imageName)
	return nil
}
```

Replace the `docker.ImageExists` calls in `buildProjectImage` (lines 105, 115) with `rt.ImageExists`. Replace `docker.GetImageDigest` (line 118) with `rt.GetImageDigest`.

Remove the `"os/exec"` import. Remove the `"github.com/joelhelbling/glovebox/internal/docker"` import (only needed for `ImageExists`/`GetImageDigest`, which now come from `rt`).

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/joelhelbling/code/ai/glovebox && go build ./...`
Expected: Clean compilation

- [ ] **Step 3: Run tests**

Run: `cd /Users/joelhelbling/code/ai/glovebox && make test`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/build.go
git commit -m "refactor: migrate cmd/build.go to runtime interface"
```

---

## Task 7: Migrate cmd/clean.go to Runtime Interface

**Files:**
- Modify: `cmd/clean.go`

- [ ] **Step 1: Replace all docker exec calls with runtime methods**

Replace `findRunningGloveboxContainers` (lines 119-152) with:

```go
func findRunningGloveboxContainers() ([]containerInfo, error) {
	containers, err := rt.ListContainers("", false) // running only
	if err != nil {
		return nil, err
	}

	var result []containerInfo
	for _, c := range containers {
		if strings.HasPrefix(c.Image, "glovebox:") {
			result = append(result, containerInfo{name: c.Name, image: c.Image})
		}
	}
	return result, nil
}
```

Replace `findGloveboxImages` (lines 211-226) with:

```go
func findGloveboxImages() ([]string, error) {
	return rt.ListImages("glovebox:*")
}
```

Replace `findGloveboxContainers` (lines 228-243) with:

```go
func findGloveboxContainers() ([]string, error) {
	containers, err := rt.ListContainers("glovebox-", true) // all, including stopped
	if err != nil {
		return nil, err
	}

	var names []string
	for _, c := range containers {
		if strings.HasPrefix(c.Name, "glovebox-") {
			names = append(names, c.Name)
		}
	}
	return names, nil
}
```

Replace `removeContainer` (lines 245-253) with:

```go
func removeContainer(name string, green *color.Color) error {
	if err := rt.ForceRemoveContainer(name); err != nil {
		return err
	}
	green.Printf("Removed container: %s\n", name)
	return nil
}
```

Replace `removeImage` (lines 255-262) with:

```go
func removeImage(name string, green *color.Color) error {
	if err := rt.RemoveImage(name); err != nil {
		return err
	}
	green.Printf("Removed image: %s\n", name)
	return nil
}
```

Replace `docker.ImageExists` (line 87) with `rt.ImageExists`. Replace `docker.ContainerExists` (line 88) with `rt.ContainerExists`.

Remove `"os/exec"` import. Remove `"github.com/joelhelbling/glovebox/internal/docker"` import — replace `docker.ImageName(targetDir)` and `docker.ContainerName(targetDir)` with imports from the docker package (keep this import for naming functions only).

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/joelhelbling/code/ai/glovebox && go build ./...`
Expected: Clean compilation

- [ ] **Step 3: Run tests**

Run: `cd /Users/joelhelbling/code/ai/glovebox && make test`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/clean.go
git commit -m "refactor: migrate cmd/clean.go to runtime interface"
```

---

## Task 8: Migrate cmd/commit.go to Runtime Interface

**Files:**
- Modify: `cmd/commit.go`

- [ ] **Step 1: Replace docker exec calls with runtime methods + capability check**

Replace lines 49-74 (the docker calls in `runCommit`) with:

```go
	// Check if container exists
	if !rt.ContainerExists(containerName) {
		return fmt.Errorf("no container found for this project\nRun 'glovebox run' first to create a container")
	}

	// Check if runtime supports commit
	caps := rt.Capabilities()
	if !caps.SupportsCommit {
		return fmt.Errorf("commit is not supported by %s runtime", rt.Name())
	}

	// Determine image name
	imageName, err := getImageNameForCommit(absPath)
	if err != nil {
		return err
	}

	// Commit the container
	prompt := ui.NewPrompt()
	fmt.Printf("Committing container to %s...\n", imageName)

	if err := rt.Commit(containerName, imageName); err != nil {
		return fmt.Errorf("committing container: %w", err)
	}

	// Remove the container
	if err := rt.RemoveContainer(containerName); err != nil {
		fmt.Print(prompt.RenderWarning(fmt.Sprintf("could not remove container: %v", err)))
	}

	fmt.Print(prompt.RenderCommitSuccess(imageName))
	fmt.Println("Next 'glovebox run' will start fresh from the updated image.")
```

Remove `"os/exec"` import. Remove `docker` from import (keep for `docker.ContainerName`).

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/joelhelbling/code/ai/glovebox && go build ./...`
Expected: Clean compilation

- [ ] **Step 3: Commit**

```bash
git add cmd/commit.go
git commit -m "refactor: migrate cmd/commit.go to runtime interface with capability check"
```

---

## Task 9: Migrate cmd/diff.go to Runtime Interface

**Files:**
- Modify: `cmd/diff.go`

- [ ] **Step 1: Replace docker exec call with runtime method + capability check**

Replace lines 54-64 (the container exists check and docker diff call) with:

```go
	// Check if container exists
	if !rt.ContainerExists(containerName) {
		fmt.Println("No container found for this project.")
		return nil
	}

	// Check if runtime supports diff
	caps := rt.Capabilities()
	if !caps.SupportsDiff {
		return fmt.Errorf("diff is not supported by %s runtime", rt.Name())
	}

	// Get the diff
	diffs, err := rt.Diff(containerName)
	if err != nil {
		return fmt.Errorf("getting container diff: %w", err)
	}

	// Convert to string format for existing processing logic
	var lines []string
	for _, d := range diffs {
		lines = append(lines, fmt.Sprintf("%s %s", d.ChangeType, d.Path))
	}
```

Update the rest of the function to use the `lines` variable instead of parsing raw output. Remove `"os/exec"` import. Replace `docker.ContainerExists` with `rt.ContainerExists` and `docker.ContainerName` stays.

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/joelhelbling/code/ai/glovebox && go build ./...`
Expected: Clean compilation

- [ ] **Step 3: Commit**

```bash
git add cmd/diff.go
git commit -m "refactor: migrate cmd/diff.go to runtime interface with capability check"
```

---

## Task 10: Migrate cmd/reset.go to Runtime Interface

**Files:**
- Modify: `cmd/reset.go`

- [ ] **Step 1: Replace docker exec calls**

Replace lines 47-57 (container exists check and rm call):

```go
	// Check if container exists
	if !rt.ContainerExists(containerName) {
		fmt.Println("No container found for this project. Nothing to reset.")
		return nil
	}

	// Remove the container
	prompt := ui.NewPrompt()
	if err := rt.RemoveContainer(containerName); err != nil {
		return fmt.Errorf("removing container: %w", err)
	}
```

Remove `"os/exec"` import. Keep `docker` import for `docker.ContainerName`.

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/joelhelbling/code/ai/glovebox && go build ./...`
Expected: Clean compilation

- [ ] **Step 3: Commit**

```bash
git add cmd/reset.go
git commit -m "refactor: migrate cmd/reset.go to runtime interface"
```

---

## Task 11: Migrate cmd/status.go to Runtime Interface

**Files:**
- Modify: `cmd/status.go`

- [ ] **Step 1: Replace docker calls in buildBaseSection, buildProjectSection, and buildContainerSection**

In `buildBaseSection` (line 82), replace:
```go
	if !rt.ImageExists("glovebox:base") {
```

In `buildProjectSection` (line 130), replace:
```go
	if !rt.ImageExists(imageName) {
```

In `buildContainerSection` (lines 182-196), replace:
```go
	if rt.ContainerExists(containerName) {
		if rt.ContainerRunning(containerName) {
			section.Items = append(section.Items,
				ui.StatusItem{Label: "Status", Value: "Running", Status: ui.StatusOK},
			)
		} else {
			section.Items = append(section.Items,
				ui.StatusItem{Label: "Status", Value: "Stopped (will resume on next run)", Status: ui.StatusOK},
			)
			// Check for uncommitted changes (only if runtime supports diff)
			caps := rt.Capabilities()
			if caps.SupportsDiff {
				changes, err := getContainerChanges(containerName)
				if err == nil && len(changes) > 0 {
					section.Items = append(section.Items,
						ui.StatusItem{Label: "Changes", Value: fmt.Sprintf("%d uncommitted", len(changes)), Status: ui.StatusWarning},
					)
				}
			}
		}
```

Replace `getContainerChanges` (lines 269-283):

```go
func getContainerChanges(name string) ([]string, error) {
	diffs, err := rt.Diff(name)
	if err != nil {
		return nil, err
	}
	var changes []string
	for _, d := range diffs {
		changes = append(changes, fmt.Sprintf("%s %s", d.ChangeType, d.Path))
	}
	return changes, nil
}
```

Remove `"os/exec"` import. Keep `docker` import for `docker.ContainerName`.

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/joelhelbling/code/ai/glovebox && go build ./...`
Expected: Clean compilation

- [ ] **Step 3: Run all tests**

Run: `cd /Users/joelhelbling/code/ai/glovebox && make test`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/status.go
git commit -m "refactor: migrate cmd/status.go to runtime interface with capability check"
```

---

## Task 12: Slim internal/docker/ Package

**Files:**
- Modify: `internal/docker/docker.go`
- Modify: `internal/docker/docker_test.go`

- [ ] **Step 1: Remove functions that moved to runtime package**

In `internal/docker/docker.go`, remove these functions (they now live in `internal/runtime/docker.go`):
- `ContainerExists` (lines 13-16)
- `ContainerRunning` (lines 19-26)
- `ImageExists` (lines 29-32)
- `GetImageDigest` (lines 35-42)
- `RunArgsConfig` type (lines 78-85)
- `RunArgsResult` type (lines 88-92)
- `BuildRunArgs` function (lines 96-134)

Keep only:
- `ContainerName` (lines 48-59)
- `ImageName` (lines 64-75)

The file should become:

```go
// Package docker provides helper functions for container naming.
package docker

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
)

// ContainerName generates a deterministic container name for a given directory.
// Format: glovebox-<dirname>-<shorthash>
func ContainerName(dir string) string {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		absPath = dir
	}

	hash := sha256.Sum256([]byte(absPath))
	shortHash := fmt.Sprintf("%x", hash)[:7]
	dirName := filepath.Base(absPath)

	return fmt.Sprintf("glovebox-%s-%s", dirName, shortHash)
}

// ImageName generates a deterministic image name for a given directory.
// Format: glovebox:<dirname>-<shorthash>
func ImageName(dir string) string {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		absPath = dir
	}

	hash := sha256.Sum256([]byte(absPath))
	shortHash := fmt.Sprintf("%x", hash)[:7]
	dirName := filepath.Base(absPath)

	return fmt.Sprintf("glovebox:%s-%s", dirName, shortHash)
}
```

- [ ] **Step 2: Remove tests for deleted functions from docker_test.go**

Remove `TestBuildRunArgs` and the helper functions `containsArg` and `containsString` from `internal/docker/docker_test.go`. Keep only:
- `TestContainerName`
- `TestImageName`
- `TestContainerNameAndImageNameConsistency`

Remove the `"strings"` import if no longer needed (it is — it's used in the remaining tests). Remove the comment block at the bottom about integration tests (lines 344-358) since those functions no longer exist in this package.

- [ ] **Step 3: Verify everything compiles and tests pass**

Run: `cd /Users/joelhelbling/code/ai/glovebox && make all`
Expected: lint, test, and build all PASS

- [ ] **Step 4: Commit**

```bash
git add internal/docker/docker.go internal/docker/docker_test.go
git commit -m "refactor: slim internal/docker/ to naming functions only"
```

---

## Task 13: Update Root Command Long Description

**Files:**
- Modify: `cmd/root.go`

- [ ] **Step 1: Update description to be runtime-agnostic**

In `cmd/root.go`, update the `Long` description:

```go
	Long: `Glovebox creates sandboxed containers for running untrusted or
experimental code. It uses a mod-based system to compose your perfect
development environment from modular, reusable pieces.

Supports multiple container runtimes. Use --runtime to override auto-detection.`,
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/joelhelbling/code/ai/glovebox && go build ./...`
Expected: Clean compilation

- [ ] **Step 3: Commit**

```bash
git add cmd/root.go
git commit -m "docs: update root command description to be runtime-agnostic"
```

---

## Task 14: Full Verification

- [ ] **Step 1: Run full lint + test + build**

Run: `cd /Users/joelhelbling/code/ai/glovebox && make all`
Expected: All pass

- [ ] **Step 2: Verify no remaining direct docker exec calls in cmd/**

Run: `grep -rn 'exec.Command("docker"' cmd/`
Expected: No matches

- [ ] **Step 3: Verify runtime interface is the only path to docker CLI**

Run: `grep -rn 'exec.Command("docker"' internal/`
Expected: Only matches in `internal/runtime/docker.go`

- [ ] **Step 4: Run glovebox commands manually (smoke test)**

```bash
./bin/glovebox --version
./bin/glovebox status
./bin/glovebox --runtime docker status
```
Expected: All work, `--runtime docker` explicitly selects Docker

- [ ] **Step 5: Commit any remaining fixes**

If any issues were found and fixed in previous steps, ensure they're committed.

---

## Notes for Phase 4 (Apple Containers — separate plan)

Phase 4 is deliberately excluded from this plan. It requires:
1. Physical validation tests on macOS 15 and 26 (see spec)
2. Results may change the implementation approach
3. Should be planned after validation results are known

When ready, the Phase 4 plan will add:
- `internal/runtime/apple.go` implementing the Runtime interface
- Apple Containers detection in `detect.go`
- Fallback messaging
- Capability checks are already in place from this plan
