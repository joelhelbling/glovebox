# Composable Mods

Glovebox uses a mod-based system to compose your development environment. Each mod is a YAML file that defines packages, commands, and configuration to install.

## Available Mods

### Operating Systems (`os/`)

Choose one as your base. This determines which package manager and OS-specific mods are available.

| Mod | Description |
|-----|-------------|
| `os/ubuntu` | Ubuntu base image with core dependencies |
| `os/fedora` | Fedora base image with core dependencies |
| `os/alpine` | Alpine Linux base image (lightweight, musl-based) |

### Shells (`shells/`)

| Mod | Description | OS |
|-----|-------------|-----|
| `shells/bash` | Bash shell (default, minimal configuration) | Any |
| `shells/zsh-ubuntu` | Z shell with sensible defaults | Ubuntu |
| `shells/zsh-fedora` | Z shell with sensible defaults | Fedora |
| `shells/zsh-alpine` | Z shell with sensible defaults | Alpine |
| `shells/fish-ubuntu` | Fish shell - the friendly interactive shell | Ubuntu |
| `shells/fish-fedora` | Fish shell - the friendly interactive shell | Fedora |
| `shells/fish-alpine` | Fish shell - the friendly interactive shell | Alpine |

### Editors (`editors/`)

| Mod | Description |
|-----|-------------|
| `editors/vim` | Vim - the ubiquitous text editor |
| `editors/neovim` | Neovim - hyperextensible Vim-based text editor |
| `editors/helix` | Helix - a post-modern modal text editor |
| `editors/emacs` | Emacs - the extensible text editor |

### Tools (`tools/`)

| Mod | Description |
|-----|-------------|
| `tools/homebrew` | The Missing Package Manager for macOS (or Linux) |
| `tools/mise` | Polyglot runtime version manager |
| `tools/tmux` | Terminal multiplexer with tmuxp session manager |

### Languages (`languages/`)

| Mod | Description |
|-----|-------------|
| `languages/nodejs` | Node.js JavaScript runtime (LTS) |

### AI Assistants (`ai/`)

| Mod | Description |
|-----|-------------|
| `ai/claude-code` | Anthropic's Claude Code CLI assistant |
| `ai/gemini-cli` | Google's Gemini CLI assistant |
| `ai/opencode` | OpenCode AI coding assistant |

## OS-Specific Mods

Some mods are tied to a specific operating system because they use OS-specific package managers or configurations. These are marked in `glovebox mod list` output:

```
  ┃ shells/
  ┃   bash         Bash shell (default, minimal configuration)
  ┃   fish-alpine  Fish shell [alpine]
  ┃   fish-fedora  Fish shell [fedora]
  ┃   fish-ubuntu  Fish shell [ubuntu]
```

During `glovebox init`, only mods compatible with your selected OS are shown.

## How Mods Work

### Mod Resolution

Glovebox searches for mods in this order:

1. **Project-local**: `.glovebox/mods/<category>/<name>.yaml`
2. **User global**: `~/.glovebox/mods/<category>/<name>.yaml`
3. **Embedded**: Built into the glovebox binary

Local mods take precedence, so you can override built-in mods if needed.

### Dependency Resolution

Mods can declare dependencies using `requires`:

```yaml
name: claude-code
requires:
  - tools/homebrew  # Concrete dependency on specific mod
```

When you add a mod, Glovebox automatically includes its dependencies.

### Abstract Dependencies

Some mods provide abstract capabilities:

```yaml
name: zsh-ubuntu
requires:
  - ubuntu      # Only works on Ubuntu
provides:
  - zsh         # Satisfies abstract "zsh" dependency
```

This allows other mods to depend on "zsh" without caring which OS-specific variant is used.

### Dockerfile Generation

When you run `glovebox build`, the generator:

1. Loads all mods from your profile
2. Resolves dependencies
3. Collects from each mod:
   - APT/DNF/APK repositories
   - Packages to install
   - `run_as_root` commands
   - `run_as_user` commands
   - Environment variables
4. Outputs a complete Dockerfile

You can see the generated Dockerfile at `~/.glovebox/Dockerfile` (base) or `.glovebox/Dockerfile` (project).

## Viewing Mod Contents

To see what a mod does:

```bash
glovebox mod cat editors/neovim
```

Output:

```yaml
name: neovim
description: Neovim - hyperextensible Vim-based text editor
category: editors

run_as_root: |
  curl -LO https://github.com/neovim/neovim/releases/latest/download/nvim-linux-x86_64.tar.gz
  tar -C /opt -xzf nvim-linux-x86_64.tar.gz
  rm nvim-linux-x86_64.tar.gz
  ln -s /opt/nvim-linux-x86_64/bin/nvim /usr/local/bin/nvim

env:
  EDITOR: nvim
```

## Adding and Removing Mods

Add a mod to your profile (adds to project profile if one exists, otherwise to base profile):

```bash
glovebox add tools/tmux
glovebox add ai/claude-code
```

Remove a mod:

```bash
glovebox remove tools/tmux
```

After changing mods, rebuild:

```bash
glovebox build        # Rebuild project image
glovebox build --base # Rebuild base image
```

## Next Steps

- [Custom Mods](custom-mods.md) - Create your own mods
- [Configuration](configuration.md) - Profile structure and options
