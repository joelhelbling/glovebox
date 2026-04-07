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
