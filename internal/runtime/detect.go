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
