package runtime

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// Compile-time check that AppleRuntime implements Runtime.
var _ Runtime = (*AppleRuntime)(nil)

// AppleRuntime implements Runtime using Apple Containers (macOS).
type AppleRuntime struct {
	io Stdio
}

// NewApple creates an Apple Containers runtime with the given I/O streams.
func NewApple(io Stdio) *AppleRuntime {
	return &AppleRuntime{io: io}
}

func (a *AppleRuntime) Name() string { return "Apple Containers" }

func (a *AppleRuntime) ImageExists(name string) bool {
	cmd := exec.Command("container", "image", "inspect", name)
	return cmd.Run() == nil
}

func (a *AppleRuntime) GetImageDigest(name string) (string, error) {
	cmd := exec.Command("container", "image", "inspect", name)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	var images []struct {
		Index struct {
			Digest string `json:"digest"`
		} `json:"index"`
	}
	if err := json.Unmarshal(output, &images); err != nil {
		return "", fmt.Errorf("failed to parse image inspect output: %w", err)
	}
	if len(images) == 0 {
		return "", fmt.Errorf("no image found for %q", name)
	}
	return images[0].Index.Digest, nil
}

func (a *AppleRuntime) BuildImage(dockerfilePath, contextDir, imageName string) error {
	// Ensure builder is running before building
	if err := a.ensureBuilder(); err != nil {
		return fmt.Errorf("failed to start builder: %w", err)
	}

	cmd := exec.Command("container", "build", "-t", imageName, "-f", dockerfilePath, contextDir)
	cmd.Stdout = a.io.Stdout
	cmd.Stderr = a.io.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("container build failed: %w", err)
	}
	return nil
}

func (a *AppleRuntime) RemoveImage(name string) error {
	return exec.Command("container", "image", "rm", name).Run()
}

// appleImageEntry represents a single entry from `container image ls --format json`.
type appleImageEntry struct {
	Reference string `json:"reference"`
}

func (a *AppleRuntime) ListImages(filterRef string) ([]string, error) {
	cmd := exec.Command("container", "image", "ls", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var allImages []appleImageEntry
	if err := json.Unmarshal(output, &allImages); err != nil {
		return nil, fmt.Errorf("failed to parse image list: %w", err)
	}

	var images []string
	for _, img := range allImages {
		// Strip "docker.io/library/" prefix for matching against short names
		name := stripDockerHubPrefix(img.Reference)
		if matchesFilter(name, filterRef) {
			images = append(images, name)
		}
	}
	return images, nil
}

func (a *AppleRuntime) ContainerExists(name string) bool {
	cmd := exec.Command("container", "inspect", name)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	// Apple Containers returns [] (empty array) with exit 0 for non-existent containers.
	var results []json.RawMessage
	if err := json.Unmarshal(output, &results); err != nil {
		return false
	}
	return len(results) > 0
}

func (a *AppleRuntime) ContainerRunning(name string) bool {
	cmd := exec.Command("container", "inspect", name)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	var containers []struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(output, &containers); err != nil {
		return false
	}
	return len(containers) > 0 && containers[0].Status == "running"
}

// buildRunArgs constructs the argument list for `container run`.
func (a *AppleRuntime) buildRunArgs(cfg RunConfig) []string {
	args := []string{
		"run", "-it",
		"--name", cfg.ContainerName,
		"-v", fmt.Sprintf("%s:%s", cfg.HostPath, cfg.WorkspacePath),
		"-w", cfg.WorkspacePath,
	}
	// Apple Containers has no --hostname flag; --name implicitly sets hostname.

	keys := make([]string, 0, len(cfg.Env))
	for k := range cfg.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, cfg.Env[key]))
	}

	args = append(args, cfg.ImageName)
	return args
}

func (a *AppleRuntime) RunInteractive(cfg RunConfig) error {
	args := a.buildRunArgs(cfg)
	cmd := exec.Command("container", args...)
	cmd.Stdin = a.io.Stdin
	cmd.Stdout = a.io.Stdout
	cmd.Stderr = a.io.Stderr
	return a.normalizeExitError(cmd.Run())
}

func (a *AppleRuntime) StartInteractive(name string) error {
	cmd := exec.Command("container", "start", "-a", "-i", name)
	cmd.Stdin = a.io.Stdin
	cmd.Stdout = a.io.Stdout
	cmd.Stderr = a.io.Stderr
	return a.normalizeExitError(cmd.Run())
}

// Attach connects to a running container. Apple Containers has no `attach`
// command, so we use `container exec -it name /bin/sh` instead.
func (a *AppleRuntime) Attach(name string) error {
	cmd := exec.Command("container", "exec", "-it", name, "/bin/sh")
	cmd.Stdin = a.io.Stdin
	cmd.Stdout = a.io.Stdout
	cmd.Stderr = a.io.Stderr
	return a.normalizeExitError(cmd.Run())
}

func (a *AppleRuntime) RemoveContainer(name string) error {
	return exec.Command("container", "rm", name).Run()
}

func (a *AppleRuntime) ForceRemoveContainer(name string) error {
	return exec.Command("container", "rm", "-f", name).Run()
}

// appleContainerEntry represents a single entry from `container ls --format json`.
type appleContainerEntry struct {
	Configuration struct {
		ID    string `json:"id"`
		Image struct {
			Reference string `json:"reference"`
		} `json:"image"`
	} `json:"configuration"`
}

func (a *AppleRuntime) ListContainers(filterName string, all bool) ([]ContainerInfo, error) {
	args := []string{"ls", "--format", "json"}
	if all {
		args = append(args, "-a")
	}

	cmd := exec.Command("container", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var allContainers []appleContainerEntry
	if err := json.Unmarshal(output, &allContainers); err != nil {
		return nil, fmt.Errorf("failed to parse container list: %w", err)
	}

	var containers []ContainerInfo
	for _, c := range allContainers {
		name := c.Configuration.ID
		image := c.Configuration.Image.Reference
		if filterName == "" || strings.Contains(name, filterName) {
			containers = append(containers, ContainerInfo{Name: name, Image: image})
		}
	}
	return containers, nil
}

func (a *AppleRuntime) Diff(name string) ([]FileDiff, error) {
	return nil, ErrNotSupported
}

func (a *AppleRuntime) Commit(containerName, imageName string) error {
	return ErrNotSupported
}

func (a *AppleRuntime) Capabilities() Capabilities {
	return Capabilities{
		SupportsDiff:   false,
		SupportsCommit: false,
		SupportsExport: true,
	}
}

// ensureBuilder starts the Apple Containers builder if it's not already running.
func (a *AppleRuntime) ensureBuilder() error {
	// Check if builder is already running
	if exec.Command("container", "builder", "info").Run() == nil {
		return nil
	}
	// Start the builder
	return exec.Command("container", "builder", "start").Run()
}

// normalizeExitError filters out normal container exit codes.
// Apple Containers uses similar conventions to Docker for exit codes.
func (a *AppleRuntime) normalizeExitError(err error) error {
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
		return fmt.Errorf("container runtime error (exit %d): %w", code, err)
	case code == 137:
		return fmt.Errorf("container was killed (exit 137, possibly out of memory)")
	default:
		return nil
	}
}

// stripDockerHubPrefix removes the "docker.io/library/" prefix that Apple Containers
// adds to locally-built images, so names match the short form used by glovebox.
func stripDockerHubPrefix(ref string) string {
	ref = strings.TrimPrefix(ref, "docker.io/library/")
	return ref
}

// matchesFilter checks if an image name matches a Docker-style reference filter.
// Supports simple prefix/glob matching (e.g., "glovebox*" matches "glovebox:base").
func matchesFilter(name, filter string) bool {
	if filter == "" {
		return true
	}
	// Handle glob-style filter (e.g., "glovebox*")
	if strings.HasSuffix(filter, "*") {
		prefix := strings.TrimSuffix(filter, "*")
		return strings.HasPrefix(name, prefix)
	}
	// Exact match or name-only match
	return name == filter || strings.HasPrefix(name, filter+":")
}
