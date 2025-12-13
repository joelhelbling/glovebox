// Package docker provides helper functions for Docker container and image operations.
package docker

import (
	"crypto/sha256"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ContainerExists checks if a container with the given name exists (running or stopped).
func ContainerExists(name string) bool {
	cmd := exec.Command("docker", "container", "inspect", name)
	return cmd.Run() == nil
}

// ContainerRunning checks if a container is currently running.
func ContainerRunning(name string) bool {
	cmd := exec.Command("docker", "container", "inspect", "-f", "{{.State.Running}}", name)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

// ImageExists checks if a Docker image with the given name exists.
func ImageExists(name string) bool {
	cmd := exec.Command("docker", "image", "inspect", name)
	return cmd.Run() == nil
}

// GetImageDigest returns the digest (ID) of a Docker image.
func GetImageDigest(name string) (string, error) {
	cmd := exec.Command("docker", "image", "inspect", "--format", "{{.Id}}", name)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// ContainerName generates a deterministic container name for a given directory.
// Format: glovebox-<dirname>-<shorthash>
// The hash is based on the absolute path to ensure uniqueness across
// directories with the same name in different locations.
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
// Uses the same hashing logic as ContainerName for consistency.
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
