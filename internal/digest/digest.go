package digest

import (
	"crypto/sha256"
	"fmt"
	"os"
)

// Calculate computes a SHA256 digest of the given content
func Calculate(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("sha256:%x", hash)
}

// CalculateFile computes a SHA256 digest of a file's contents
func CalculateFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file for digest: %w", err)
	}
	return Calculate(string(data)), nil
}

// Match checks if the given content matches the expected digest
func Match(content, expected string) bool {
	return Calculate(content) == expected
}

// MatchFile checks if a file's content matches the expected digest
func MatchFile(path, expected string) (bool, error) {
	actual, err := CalculateFile(path)
	if err != nil {
		return false, err
	}
	return actual == expected, nil
}

// Short returns a shortened version of the digest for display
func Short(digest string) string {
	// Remove "sha256:" prefix and return first 12 chars
	if len(digest) > 19 { // "sha256:" (7) + 12 chars
		return digest[7:19]
	}
	return digest
}
