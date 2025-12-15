# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Git Commits

**NEVER skip GPG signing.** All commits must be signed. If GPG signing times out, alert the user to touch their 2FA key and retry the commit. Do not use `--no-gpg-sign` under any circumstances.

## Development Commands

Use `make help` to see all available targets. Key commands:

```bash
make build      # Build binary with version from git tags
make test       # Run tests
make lint       # Run fmt and vet
make install    # Build and install to /usr/local/bin
make version    # Show current version string
make all        # Run lint, test, and build
```

### Testing Glovebox Docker Images

```bash
./bin/glovebox build --base    # Build base image from ~/.glovebox/profile.yaml
./bin/glovebox build           # Build project image from .glovebox/profile.yaml
./bin/glovebox clean           # Remove project container and image
./bin/glovebox clean --all     # Remove all glovebox containers and images
```

## Versioning and Releases

Glovebox uses git tag-based versioning. Version is injected automatically at build time.

### Version Format

- On a tag: `v0.2.0`
- After commits: `v0.2.0-3-g1a2b3c4` (3 commits after v0.2.0, at commit 1a2b3c4)
- With uncommitted changes: `v0.2.0-3-g1a2b3c4-dirty`

### Creating a Release

```bash
# Ensure all changes are committed first, then:
make release V=v0.3.0

# This will:
# 1. Verify working directory is clean
# 2. Create annotated tag v0.3.0
# 3. Build binary with that version
# 4. Print instructions for pushing

# Push commit and tag:
git push origin main && git push origin v0.3.0
```

### Checking Current Version

```bash
make version           # Show version that would be built
./bin/glovebox -v      # Show version of built binary
```

## Architecture

Glovebox is a Go CLI that generates and runs Docker containers for sandboxed development environments. It uses a **mod-based composition system** where YAML mods define reusable pieces of Dockerfile configuration.

### Key Packages

- `cmd/` - Cobra CLI commands (init, build, run, add, remove, status, clean, clone, mod)
- `internal/mod/` - Mod loading with embedded filesystem (`//go:embed`) and local override support
- `internal/profile/` - Profile management (global `~/.glovebox/` and project `.glovebox/`)
- `internal/generator/` - Dockerfile generation from mods (`GenerateBase`, `GenerateProject`)
- `internal/assets/` - Embedded assets like the entrypoint script

### Mod System

Mods are YAML files embedded in the binary at `internal/mod/mods/`. Categories:
- `os/` - Base OS images (ubuntu, fedora, alpine)
- `shells/` - Shell configurations
- `editors/` - Text editors
- `tools/` - Development tools
- `languages/` - Programming language runtimes
- `ai/` - AI coding assistants

Some mods are OS-agnostic (use homebrew/mise), others have OS-specific variants (e.g., `zsh-ubuntu.yaml`, `zsh-fedora.yaml`). Run `glovebox mod list` to see all available mods.

Load priority: project-local `.glovebox/mods/` → user global `~/.glovebox/mods/` → embedded

### Layered Image Architecture

1. **Base image** (`glovebox:base`) - Built from `~/.glovebox/profile.yaml`, contains user's standard environment
2. **Project images** - Built from `.glovebox/profile.yaml`, extend base with project-specific tools

The generator (`internal/generator/generator.go`) collects run_as_root commands, run_as_user commands, and env vars from mods and outputs a Dockerfile.

### Container Persistence

Glovebox uses persistent containers (not `--rm`). Each project gets its own named container (`glovebox-<dirname>-<hash>`). On exit, `docker diff` detects filesystem changes and prompts the user to commit them back to the image.

Key functions in `cmd/run.go`:
- `checkContainerExists()` / `checkContainerRunning()` - Container state detection
- `createAndStartContainer()` - First run
- `startContainer()` - Subsequent runs
- `handlePostExit()` - Diff detection and commit prompt

## Adding New Mods

Mods requiring OS-specific package installation need OS variants (e.g., `toolname-ubuntu.yaml`, `toolname-fedora.yaml`, `toolname-alpine.yaml`). Mods using homebrew or mise can remain OS-agnostic.

Create a YAML file in `internal/mod/mods/<category>/<name>.yaml`:

```yaml
name: toolname-ubuntu
description: Brief description (Ubuntu)
category: tool
requires:
  - ubuntu
provides:
  - toolname

run_as_root: |
  apt-get update && apt-get install -y some-package && rm -rf /var/lib/apt/lists/*

run_as_user: |
  # Commands run as dev user

env:
  PATH: /some/path:$PATH
```

For OS-agnostic mods (using homebrew/mise):

```yaml
name: toolname
description: Brief description
category: tool
requires:
  - tools/homebrew
provides:
  - toolname

run_as_user: |
  brew install toolname
```

After adding, rebuild the binary to embed the new mod.
