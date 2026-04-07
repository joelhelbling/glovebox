package runtime

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// Compile-time check that DockerRuntime implements Runtime.
var _ Runtime = (*DockerRuntime)(nil)

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
