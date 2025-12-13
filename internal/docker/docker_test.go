package docker

import (
	"strings"
	"testing"
)

func TestContainerName(t *testing.T) {
	t.Run("deterministic output", func(t *testing.T) {
		dir := "/some/project/path"
		first := ContainerName(dir)
		second := ContainerName(dir)

		if first != second {
			t.Errorf("ContainerName not deterministic: %q != %q", first, second)
		}
	})

	t.Run("format check", func(t *testing.T) {
		dir := "/home/user/myproject"
		got := ContainerName(dir)

		// Should start with "glovebox-"
		if !strings.HasPrefix(got, "glovebox-") {
			t.Errorf("expected format 'glovebox-<name>-<hash>', got %q", got)
		}

		// Should contain the directory name
		if !strings.Contains(got, "myproject") {
			t.Errorf("expected container name to contain 'myproject', got %q", got)
		}

		// Should have format: glovebox-<dirname>-<7char hash>
		parts := strings.Split(got, "-")
		if len(parts) < 3 {
			t.Errorf("expected at least 3 parts separated by '-', got %q", got)
		}

		// Last part should be 7 char hash
		lastPart := parts[len(parts)-1]
		if len(lastPart) != 7 {
			t.Errorf("expected 7-char hash suffix, got %q (len=%d)", lastPart, len(lastPart))
		}
	})

	t.Run("different paths produce different names", func(t *testing.T) {
		name1 := ContainerName("/path/one")
		name2 := ContainerName("/path/two")

		if name1 == name2 {
			t.Error("different paths should produce different container names")
		}
	})

	t.Run("same dirname different paths have different hashes", func(t *testing.T) {
		// Two projects named "app" in different locations
		name1 := ContainerName("/home/user1/app")
		name2 := ContainerName("/home/user2/app")

		// Both should contain "app" but have different hashes
		if name1 == name2 {
			t.Error("same dirname with different full paths should have different hashes")
		}

		// Both should contain "app"
		if !strings.Contains(name1, "app") || !strings.Contains(name2, "app") {
			t.Error("both names should contain 'app'")
		}
	})

	t.Run("handles relative path", func(t *testing.T) {
		// Should not panic on relative path
		got := ContainerName("relative/path")
		if got == "" {
			t.Error("expected non-empty container name for relative path")
		}
		if !strings.HasPrefix(got, "glovebox-") {
			t.Errorf("expected glovebox- prefix, got %q", got)
		}
	})
}

func TestImageName(t *testing.T) {
	t.Run("deterministic output", func(t *testing.T) {
		dir := "/some/project/path"
		first := ImageName(dir)
		second := ImageName(dir)

		if first != second {
			t.Errorf("ImageName not deterministic: %q != %q", first, second)
		}
	})

	t.Run("format check", func(t *testing.T) {
		dir := "/home/user/myproject"
		got := ImageName(dir)

		// Should start with "glovebox:"
		if !strings.HasPrefix(got, "glovebox:") {
			t.Errorf("expected format 'glovebox:<name>-<hash>', got %q", got)
		}

		// Should contain the directory name
		if !strings.Contains(got, "myproject") {
			t.Errorf("expected image name to contain 'myproject', got %q", got)
		}
	})

	t.Run("different paths produce different names", func(t *testing.T) {
		name1 := ImageName("/path/one")
		name2 := ImageName("/path/two")

		if name1 == name2 {
			t.Error("different paths should produce different image names")
		}
	})

	t.Run("same dirname different paths have different hashes", func(t *testing.T) {
		name1 := ImageName("/home/user1/app")
		name2 := ImageName("/home/user2/app")

		if name1 == name2 {
			t.Error("same dirname with different full paths should have different hashes")
		}
	})
}

func TestContainerNameAndImageNameConsistency(t *testing.T) {
	t.Run("use same hash for same directory", func(t *testing.T) {
		dir := "/home/user/project"
		containerName := ContainerName(dir)
		imageName := ImageName(dir)

		// Extract hash from container name (glovebox-project-<hash>)
		containerParts := strings.Split(containerName, "-")
		containerHash := containerParts[len(containerParts)-1]

		// Extract hash from image name (glovebox:project-<hash>)
		imageTag := strings.TrimPrefix(imageName, "glovebox:")
		imageParts := strings.Split(imageTag, "-")
		imageHash := imageParts[len(imageParts)-1]

		if containerHash != imageHash {
			t.Errorf("container and image should use same hash: container=%q, image=%q", containerHash, imageHash)
		}
	})

	t.Run("both contain directory name", func(t *testing.T) {
		dir := "/home/user/myapp"
		containerName := ContainerName(dir)
		imageName := ImageName(dir)

		if !strings.Contains(containerName, "myapp") {
			t.Errorf("container name should contain 'myapp': %q", containerName)
		}
		if !strings.Contains(imageName, "myapp") {
			t.Errorf("image name should contain 'myapp': %q", imageName)
		}
	})
}

// Note: ContainerExists, ContainerRunning, ImageExists, and GetImageDigest
// require a running Docker daemon to test properly. These would be better
// suited for integration tests. For unit tests, we verify the name generation
// functions which are pure and have no external dependencies.
//
// Integration tests could be added in a separate file with build tags:
//   //go:build integration
//
// Example integration test structure:
//   func TestContainerExists_Integration(t *testing.T) {
//       // Create a test container
//       // Verify ContainerExists returns true
//       // Remove container
//       // Verify ContainerExists returns false
//   }
