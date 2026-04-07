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
