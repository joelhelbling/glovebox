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
| `shells/zsh` | Z shell with sensible defaults | Alpine, Fedora, Ubuntu |
| `shells/fish` | Fish shell - the friendly interactive shell | Alpine, Fedora, Ubuntu |

### Editors (`editors/`)

| Mod | Description | OS |
|-----|-------------|-----|
| `editors/vim` | Vim - the ubiquitous text editor | Alpine, Fedora, Ubuntu |
| `editors/neovim` | Neovim - hyperextensible Vim-based text editor | Alpine, Fedora, Ubuntu |
| `editors/helix` | Helix - a post-modern modal text editor | Alpine, Fedora, Ubuntu |
| `editors/emacs` | Emacs - the extensible text editor | Alpine, Fedora, Ubuntu |

### Tools (`tools/`)

| Mod | Description | OS |
|-----|-------------|-----|
| `tools/homebrew` | The Missing Package Manager for macOS (or Linux) | Any |
| `tools/mise` | Polyglot runtime version manager | Any |
| `tools/tmux` | Terminal multiplexer | Alpine, Fedora, Ubuntu |

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

Many mods have OS-specific variants that use native package managers (apt, dnf, apk) for lighter-weight installations. When you run `glovebox mod list`, these are shown consolidated:

```
  ┃ editors/
  ┃   emacs   Emacs - the extensible text editor (for Alpine, Fedora, Ubuntu)
  ┃   helix   Helix - a post-modern modal text editor (for Alpine, Fedora, Ubuntu)
  ┃   neovim  Neovim - hyperextensible Vim-based text editor (for Alpine, Fedora, Ubuntu)
  ┃   vim     Vim - the ubiquitous text editor (for Alpine, Fedora, Ubuntu)
```

### Automatic Resolution

When adding mods, Glovebox automatically resolves base names to the correct OS-specific variant:

```bash
# If your profile uses Ubuntu:
glovebox add editors/vim
# ✓ Added 'editors/vim-ubuntu' to profile
```

The same works for removal:

```bash
glovebox remove editors/vim
# ✓ Removed 'editors/vim-ubuntu' from profile
```

Your profile stores the full mod name (e.g., `editors/vim-ubuntu`) so you can always see exactly what's installed.

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
name: gemini-cli
requires:
  - languages/nodejs  # Needs Node.js for npm install
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
glovebox mod cat editors/vim-ubuntu
```

Output:

```yaml
name: vim-ubuntu
description: Vim - the ubiquitous text editor (Ubuntu)
category: editor
requires:
  - ubuntu
provides:
  - vim

run_as_root: |
  apt-get update && apt-get install -y vim && rm -rf /var/lib/apt/lists/*

env:
  EDITOR: vim
```

## Adding and Removing Mods

Add a mod to your profile (adds to project profile if one exists, otherwise to base profile):

```bash
glovebox add editors/vim      # Resolves to editors/vim-ubuntu on Ubuntu
glovebox add ai/claude-code   # OS-agnostic, adds as-is
```

Remove a mod:

```bash
glovebox remove editors/vim   # Removes editors/vim-ubuntu if that's installed
```

After changing mods, rebuild:

```bash
glovebox build        # Rebuild project image
glovebox build --base # Rebuild base image
```

## Next Steps

- [Custom Mods](custom-mods.md) - Create your own mods
- [Configuration](configuration.md) - Profile structure and options
