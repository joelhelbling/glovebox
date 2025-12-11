# Glovebox

![Glovebox](glovebox-1.jpg)

A composable, dockerized development sandbox for working with dangerous things like agentic coding tools and npm packages.

## Prerequisites

- Docker
- Go 1.25+ (for building from source)

## Installation

### From Source

```bash
git clone https://github.com/joelhelbling/glovebox.git
cd glovebox
go build -o bin/glovebox .
```

Then add the `bin` directory to your PATH, or symlink the binary:

```bash
ln -s /path/to/glovebox/bin/glovebox ~/.local/bin/glovebox
```

## Quick Start

```bash
# Create your base environment (one time setup)
glovebox init --global
glovebox build --base

# Run glovebox in any project directory
cd /path/to/your/project
glovebox run
```

## Architecture

Glovebox uses a **layered image approach**:

1. **Base Image (`glovebox:base`)**: Your standard development environment defined in `~/.glovebox/profile.yaml`. Contains your preferred shell, editor, and common tools. Build once, use everywhere.

2. **Project Images**: Optional project-specific extensions defined in `.glovebox/profile.yaml`. Extends the base image with additional tools needed for that project.

```
┌─────────────────────────────┐
│     Project Image           │  ← Project-specific tools
│  (glovebox:myproject-abc123)│     FROM glovebox:base
├─────────────────────────────┤
│     Base Image              │  ← Your standard environment
│     (glovebox:base)         │     Shell, editor, mise, etc.
├─────────────────────────────┤
│     Ubuntu 24.04            │
└─────────────────────────────┘
```

## Commands

| Command | Description |
|---------|-------------|
| `glovebox init --global` | Create global base profile (~/.glovebox/profile.yaml) |
| `glovebox init` | Create project-specific profile (.glovebox/profile.yaml) |
| `glovebox list` | List all available snippets |
| `glovebox add <snippet>` | Add a snippet to your profile |
| `glovebox remove <snippet>` | Remove a snippet from your profile |
| `glovebox build --base` | Build the base image from global profile |
| `glovebox build` | Build project image (or base if no project profile) |
| `glovebox build --generate-only` | Only generate Dockerfile, don't build |
| `glovebox status` | Show profile and image status |
| `glovebox run [directory]` | Run glovebox container |
| `glovebox clone <repo>` | Clone a repo and start glovebox in it |

## Composable Snippets

Glovebox uses a snippet-based system to compose your development environment:

```bash
$ glovebox list

ai:
  ai/claude-code       Anthropic's Claude Code CLI assistant
  ai/gemini-cli        Google's Gemini CLI assistant
  ai/opencode          OpenCode AI coding assistant

core:
  base                 Core dependencies required by all glovebox environments

editors:
  editors/helix        Helix - a post-modern modal text editor
  editors/neovim       Neovim - hyperextensible Vim-based text editor
  editors/vim          Vim - the ubiquitous text editor

languages:
  languages/nodejs     Node.js JavaScript runtime (v22 LTS)

shells:
  shells/bash          Bash shell (default, minimal configuration)
  shells/fish          Fish shell - the friendly interactive shell
  shells/zsh           Z shell with sensible defaults

tools:
  tools/mise           Polyglot runtime version manager
  tools/tmux           Terminal multiplexer with tmuxp session manager
```

## Workflow

### Initial Setup (One Time)

```bash
# Create your base environment with your preferred tools
glovebox init --global

# Build the base image
glovebox build --base
```

### Daily Use

```bash
# Run glovebox in any project directory
cd ~/projects/my-app
glovebox run
```

### Project-Specific Tools

If a project needs additional tools not in your base image:

```bash
cd ~/projects/special-project

# Create a project profile
glovebox init

# Add project-specific snippets
glovebox add languages/python

# Build and run
glovebox build
glovebox run
```

## Persistence

### Home Directory Volume

Each project gets its own Docker volume for the container's home directory. This persists:

- Shell history
- Mise-installed language versions
- Tool configurations
- Any files you create in `~`

The volume is named `glovebox-<dirname>-<hash>-home`.

### Philosophy: Dockerfile as Source of Truth

While the home volume provides persistence, **treat your Dockerfile as the source of truth**:

- If you want a tool to always be available, add it via a snippet
- The home volume is a cache, not permanent storage
- Deleting the volume and re-running should give you a fully functional environment

This works well with tools like mise and direnv:

```bash
# In your project's .envrc
mise install      # Installs versions from mise.toml
mise activate     # Activates the environment
```

The container has mise/direnv installed (via snippets), but the specific language versions are installed on-demand and cached in the volume.

## API Keys

The following environment variables are passed through to the container:

- `ANTHROPIC_API_KEY`
- `OPENAI_API_KEY`
- `GOOGLE_API_KEY`
- `GEMINI_API_KEY`

Additionally, these config directories are mounted read-only from your host:

- `~/.anthropic` → `/home/ubuntu/.anthropic`
- `~/.config/gemini` → `/home/ubuntu/.config/gemini`

## Creating Custom Snippets

Custom snippets can be placed in two locations:

| Location | Scope | Path |
|----------|-------|------|
| Project-local | Only this project | `.glovebox/snippets/<name>.yaml` |
| User global | All your projects | `~/.glovebox/snippets/<name>.yaml` |

Local snippets take precedence over embedded ones, so you can override built-in snippets if needed.

### Snippet Structure

Create a YAML file with the following structure:

```yaml
name: my-tool
description: My custom tool configuration
category: custom
requires:
  - base  # dependencies on other snippets

apt_repos:
  - ppa:some/repo  # APT repositories to add

apt_packages:
  - some-package  # APT packages to install

run_as_root: |
  # Shell commands to run as root
  curl -fsSL https://example.com/install.sh | bash

run_as_user: |
  # Shell commands to run as the ubuntu user
  echo "setup complete"

env:
  MY_VAR: value  # Environment variables to set

user_shell: /usr/bin/bash  # Set as default shell (optional)
```

### Examples

**Add to your base image** (available everywhere):
```bash
mkdir -p ~/.glovebox/snippets/custom
# Create ~/.glovebox/snippets/custom/my-tool.yaml
glovebox add custom/my-tool
glovebox build --base
```

**Add to a project** (only for this project):
```bash
mkdir -p .glovebox/snippets/custom
# Create .glovebox/snippets/custom/my-tool.yaml
glovebox add custom/my-tool
glovebox build
```

**Override a built-in snippet** (e.g., customize neovim):
```bash
mkdir -p ~/.glovebox/snippets/editors
# Create ~/.glovebox/snippets/editors/neovim.yaml with your customizations
# This will be used instead of the built-in neovim snippet
glovebox build --base
```

## File Locations

| File | Purpose |
|------|---------|
| `~/.glovebox/profile.yaml` | Global profile (base image definition) |
| `~/.glovebox/Dockerfile` | Generated base Dockerfile |
| `~/.glovebox/snippets/` | Custom global snippets |
| `.glovebox/profile.yaml` | Project profile (extends base) |
| `.glovebox/Dockerfile` | Generated project Dockerfile |
| `.glovebox/snippets/` | Custom project snippets |
