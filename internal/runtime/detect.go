package runtime

import (
	"fmt"
	"os/exec"
	goruntime "runtime"
	"strings"
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
	case "apple":
		if !appleAvailable() {
			return DetectResult{}, fmt.Errorf("Apple Containers not available: install with 'brew install --cask container'")
		}
		return DetectResult{Runtime: NewApple(io)}, nil
	default:
		return DetectResult{}, fmt.Errorf("unknown runtime %q (available: apple, docker)", name)
	}
}

const dockerFallbackMsg = `Note: Using Docker runtime. For better isolation, install Apple Containers:
  brew install --cask container
Docker containers share a kernel with the host system. Apple Containers
run each container in its own virtual machine for hardware-level isolation.`

func detectAuto(io Stdio) (DetectResult, error) {
	if appleAvailable() {
		return DetectResult{Runtime: NewApple(io)}, nil
	}

	if dockerAvailable() {
		result := DetectResult{Runtime: NewDocker(io)}
		// If we're on macOS, let the user know about Apple Containers
		if isMacOS() {
			result.FellBack = true
			result.FallbackMsg = dockerFallbackMsg
		}
		return result, nil
	}

	return DetectResult{}, fmt.Errorf("no container runtime found\n" +
		"Install Apple Containers: brew install --cask container\n" +
		"Install Docker: https://docs.docker.com/get-docker/")
}

// appleAvailable checks if Apple Containers CLI exists and responds to --version.
// This avoids false positives from other binaries named "container".
func appleAvailable() bool {
	path, err := exec.LookPath("container")
	if err != nil {
		return false
	}
	// Verify it's actually Apple Containers by checking --version output
	out, err := exec.Command(path, "--version").Output()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(out)), "container")
}

// dockerAvailable checks if the docker CLI exists and the daemon is responsive.
func dockerAvailable() bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	// Verify daemon is running
	return exec.Command("docker", "info").Run() == nil
}

// isMacOS reports whether the current OS is macOS.
func isMacOS() bool {
	return goruntime.GOOS == "darwin"
}
