# Glovebox

![Glovebox](glovebox-1.jpg)

A composable, dockerized development sandbox for working with dangerous things like agentic coding tools and npm packages.

## Why Glovebox?

AI coding assistants are powerful, but they run code. So do npm packages, pip installs, and that sketchy shell script you found on Stack Overflow. Running untrusted code on your development machine is a risk—but constantly spinning up VMs or fighting with container configs kills your flow.

Glovebox gives you a sandboxed Docker environment that actually feels like home. Your shell, your editor, your tools—all running safely inside a container with your project mounted. Think of it as glamping on Jurassic Island: even in mortal danger, you still get your Nespresso.

**What makes it different:**

- **Composable mods** - Mix and match shells, editors, languages, and AI tools
- **Layered images** - Build once, extend per-project
- **Persistent volumes** - Your shell history, tool configs, and installed runtimes survive rebuilds
- **First-run provisioning** - Heavy tools install once, then persist

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
glovebox init --base
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

### Build-time vs Post-install Mods

Mods can be installed at two different phases:

| Phase | When | Use Case |
|-------|------|----------|
| **build** (default) | During `docker build` | Core tools, shells, package managers |
| **post_install** | First container run | Tools that benefit from volume persistence |

Post-install mods (like AI coding assistants and editors) are installed on first container start:

```
===========================================
Glovebox: First-run provisioning
===========================================

Installing tools you selected. This only
happens on first run - subsequent starts
will be instant.

[1/1] Installing claude-code...
🍺  claude-code was successfully installed!

===========================================
Provisioning complete!
===========================================
```

**Benefits of post-install:**
- Smaller Docker images (tools not baked into layers)
- Installations persist in home volume across image rebuilds
- Faster iteration on base image without reinstalling tools

**Implementation details:**

- Post-install script location: `/usr/local/lib/glovebox/post-install.sh`
- First-run marker file: `~/.glovebox-initialized`

## Design Philosophy

Glovebox balances two competing concerns:

1. **Don't nom my SSD** - Docker images and volumes can balloon quickly. We minimize image size by using layered builds and deferring tool installation to first-run when it makes sense.

2. **Don't waste my time** - Nobody wants to wait for homebrew to install every time they start a container. Persistent volumes cache your installed tools, shell history, and configurations.

**The balance:**

| What | Where | Why |
|------|-------|-----|
| Shells, package managers | Baked into image | Fast, rarely change |
| Editors, AI tools | Post-install (volume) | Large, benefit from persistence |
| Language runtimes | Mise on volume | Project-specific versions |
| Your code | Mounted from host | Always current, never copied |

**Source of truth:** Your profile and mods define what *should* be installed. The volume is a cache. If you delete the volume and rebuild, you should get a fully functional environment—it just might take a minute on first boot.

## Commands

| Command | Description |
|---------|-------------|
| `glovebox init --base` | Create base profile (~/.glovebox/profile.yaml) |
| `glovebox init` | Create project-specific profile (.glovebox/profile.yaml) |
| `glovebox add <mod>` | Add a mod to your profile |
| `glovebox remove <mod>` | Remove a mod from your profile |
| `glovebox build --base` | Build the base image from base profile |
| `glovebox build` | Build project image (or base if no project profile) |
| `glovebox build --generate-only` | Only generate Dockerfile, don't build |
| `glovebox status` | Show profile and image status |
| `glovebox run [directory]` | Run glovebox container |
| `glovebox clone <repo>` | Clone a repo and start glovebox in it |
| `glovebox mod list` | List all available mods (alias: `ls`) |
| `glovebox mod cat <id>` | Output a mod's raw YAML to stdout |
| `glovebox mod create <name>` | Create a new custom mod from template |

## Composable Mods

Glovebox uses a mod-based system to compose your development environment:

```bash
$ glovebox mod list

ai:
  ai/claude-code       Anthropic's Claude Code CLI assistant
  ai/gemini-cli        Google's Gemini CLI assistant
  ai/opencode          OpenCode AI coding assistant

core:
  base                 Core dependencies required by all glovebox environments

editors:
  editors/emacs        GNU Emacs - the extensible text editor
  editors/helix        Helix - a post-modern modal text editor
  editors/neovim       Neovim - hyperextensible Vim-based text editor
  editors/vim          Vim - the ubiquitous text editor

languages:
  languages/nodejs     Node.js JavaScript runtime (LTS)

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
glovebox init --base

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

# Add project-specific mods
glovebox add languages/python

# Build and run
glovebox build
glovebox run
```

## Persistence

### Home Directory Volume

Each project gets its own Docker volume for the container's home directory (`/home/ubuntu`). The volume is named `glovebox-<dirname>-<hash>-home`.

**What lives in the volume:**

| Path | Contents |
|------|----------|
| `~/.local/share/mise/` | Mise-installed language runtimes |
| `/home/linuxbrew/.linuxbrew/` | Homebrew and installed packages |
| `~/.config/` | Tool configurations |
| `~/.bash_history`, etc. | Shell history |
| `~/.glovebox-initialized` | First-run marker file |

**What lives in the image:**

| Path | Contents |
|------|----------|
| `/usr/bin/`, `/usr/local/bin/` | APT-installed tools, shells |
| `/usr/local/lib/glovebox/` | Post-install script |

### Working with Mise and Direnv

The container has mise installed (via mod), but specific language versions are installed on-demand and cached in the volume:

```bash
# In your project's .envrc
mise install      # Installs versions from mise.toml
mise activate     # Activates the environment
```

This means your first `cd` into a project directory might trigger installs, but subsequent runs are instant.

## API Keys

The following environment variables are passed through to the container:

- `ANTHROPIC_API_KEY`
- `OPENAI_API_KEY`
- `GOOGLE_API_KEY`
- `GEMINI_API_KEY`

Additionally, these config directories are mounted read-only from your host:

- `~/.anthropic` → `/home/ubuntu/.anthropic`
- `~/.config/gemini` → `/home/ubuntu/.config/gemini`

## Creating Custom Mods

Custom mods can be placed in two locations:

| Location | Scope | Path |
|----------|-------|------|
| Project-local | Only this project | `.glovebox/mods/<name>.yaml` |
| User base | All your projects | `~/.glovebox/mods/<name>.yaml` |

Local mods take precedence over embedded ones, so you can override built-in mods if needed.

### Mod Structure

Create a YAML file with the following structure:

```yaml
name: my-tool
description: My custom tool configuration
category: custom
install_phase: build  # "build" (default) or "post_install"
requires:
  - base  # dependencies on other mods

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

Use `install_phase: post_install` for tools installed via homebrew or mise that you want to persist in volumes.

### Examples

**Add to your base image** (available everywhere):
```bash
mkdir -p ~/.glovebox/mods/custom
# Create ~/.glovebox/mods/custom/my-tool.yaml
glovebox add custom/my-tool
glovebox build --base
```

**Add to a project** (only for this project):
```bash
mkdir -p .glovebox/mods/custom
# Create .glovebox/mods/custom/my-tool.yaml
glovebox add custom/my-tool
glovebox build
```

**Override a built-in mod** (e.g., customize neovim):
```bash
# Copy the built-in mod as a starting point
mkdir -p ~/.glovebox/mods/editors
glovebox mod cat editors/neovim > ~/.glovebox/mods/editors/neovim.yaml

# Edit to customize, then rebuild
glovebox build --base
```

## File Locations

| File | Purpose |
|------|---------|
| `~/.glovebox/profile.yaml` | Global profile (base image definition) |
| `~/.glovebox/Dockerfile` | Generated base Dockerfile |
| `~/.glovebox/mods/` | Custom global mods |
| `.glovebox/profile.yaml` | Project profile (extends base) |
| `.glovebox/Dockerfile` | Generated project Dockerfile |
| `.glovebox/mods/` | Custom project mods |

## Roadmap

Features under consideration:

- **Dotfiles integration** - Automatically sync your dotfiles into the container
- **SSH key forwarding** - Securely access your SSH keys for git operations
- **Networking affordances** - Connect to host services (Ollama, LM Studio) and other containers
- **GPU passthrough** - Access host GPU for local AI model inference
