# Creating Custom Mods

Custom mods let you extend Glovebox with your own tools, configurations, and workflows.

## Mod Locations

| Location | Scope | Path |
|----------|-------|------|
| Project-local | Only this project | `.glovebox/mods/<category>/<name>.yaml` |
| User global | All your projects | `~/.glovebox/mods/<category>/<name>.yaml` |

Local mods take precedence over embedded ones, so you can override built-in mods if needed.

## Creating a Mod

### Using the Template

The easiest way to start:

```bash
glovebox mod create my-tool
# Creates .glovebox/mods/custom/my-tool.yaml
```

For a global mod:

```bash
glovebox mod create --global my-tool
# Creates ~/.glovebox/mods/custom/my-tool.yaml
```

### Mod Structure

A mod is a YAML file with these fields:

```yaml
name: my-tool
description: Brief description of what this mod provides
category: custom

# What this mod provides (optional, defaults to the mod name)
provides:
  - some-capability

# Dependencies on other mods
requires:
  - tools/homebrew  # Use full mod IDs for concrete dependencies

# Commands run as root during image build
run_as_root: |
  apt-get update && apt-get install -y some-package

# Commands run as the ubuntu user during image build
run_as_user: |
  echo "export MY_VAR=value" >> ~/.bashrc

# Environment variables set in the container
env:
  MY_VAR: value
  PATH: /some/path:$PATH

# Set as default shell (optional)
user_shell: /usr/bin/zsh
```

### Field Reference

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Mod identifier (should match filename) |
| `description` | Yes | Brief description shown in `mod list` |
| `category` | Yes | Grouping for organization |
| `provides` | No | Abstract capabilities this mod provides |
| `requires` | No | Dependencies (other mod IDs or abstract capabilities) |
| `run_as_root` | No | Shell commands run as root |
| `run_as_user` | No | Shell commands run as ubuntu user |
| `env` | No | Environment variables to set |
| `user_shell` | No | Set as default shell |

### Package Installation

For packages, use the appropriate package manager based on your target OS:

**Ubuntu (apt):**
```yaml
run_as_root: |
  apt-get update && apt-get install -y ripgrep fd-find
```

**Fedora (dnf):**
```yaml
run_as_root: |
  dnf install -y ripgrep fd-find
```

**Alpine (apk):**
```yaml
run_as_root: |
  apk add --no-cache ripgrep fd
```

**Homebrew (cross-platform):**
```yaml
requires:
  - tools/homebrew

run_as_user: |
  brew install ripgrep fd
```

## Examples

### Simple Tool Installation

```yaml
name: ripgrep
description: Fast recursive grep alternative
category: tools

run_as_root: |
  apt-get update && apt-get install -y ripgrep
```

### Tool with Homebrew Dependency

```yaml
name: github-cli
description: GitHub's official CLI
category: tools

requires:
  - tools/homebrew

run_as_user: |
  brew install gh
```

### Configuration Mod

```yaml
name: git-config
description: Standard git configuration
category: config

run_as_user: |
  git config --global init.defaultBranch main
  git config --global pull.rebase true
  git config --global core.editor nvim
```

### OS-Specific Mod

```yaml
name: docker-ubuntu
description: Docker CLI for Ubuntu
category: tools

requires:
  - ubuntu  # Only works on Ubuntu

run_as_root: |
  apt-get update
  apt-get install -y ca-certificates curl gnupg
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
  apt-get update
  apt-get install -y docker-ce-cli
```

### Abstract Dependency

```yaml
name: my-shell-config
description: Custom shell configuration
category: config

requires:
  - zsh  # Works with any zsh mod (zsh-ubuntu, zsh-fedora, etc.)

run_as_user: |
  echo 'alias ll="ls -la"' >> ~/.zshrc
```

## Overriding Built-in Mods

To customize a built-in mod:

```bash
# Copy the built-in mod as a starting point
mkdir -p ~/.glovebox/mods/editors
glovebox mod cat editors/neovim > ~/.glovebox/mods/editors/neovim.yaml

# Edit to customize
vim ~/.glovebox/mods/editors/neovim.yaml

# Rebuild to apply changes
glovebox build --base
```

Your local version takes precedence over the embedded one.

## Testing Mods

After creating or modifying a mod:

1. Add it to your profile:
   ```bash
   glovebox add custom/my-tool
   ```

2. Generate the Dockerfile to inspect:
   ```bash
   glovebox build --generate-only
   cat .glovebox/Dockerfile  # or ~/.glovebox/Dockerfile for base
   ```

3. Build and test:
   ```bash
   glovebox build
   glovebox run
   ```

4. If something breaks, clean up and iterate:
   ```bash
   glovebox clean
   # Edit your mod
   glovebox build
   ```

## Best Practices

1. **Choose the right installation method**:
   - For OS-specific mods, prefer native package managers (apt, dnf, apk) for lighter-weight images
   - Homebrew is still useful for tools not available in native repos, or when you want one mod that works across all OSes

2. **Create OS-specific variants when needed** - If your mod uses OS-specific commands, create separate variants (e.g., `my-tool-ubuntu.yaml`, `my-tool-fedora.yaml`) with:
   - `requires: [ubuntu]` (or fedora, alpine)
   - `provides: [my-tool]` so other mods can depend on the abstract capability

3. **Declare dependencies explicitly** - Don't assume other mods are present

4. **Keep mods focused** - One tool or configuration per mod is easier to manage

5. **Test on clean builds** - Use `glovebox clean` before testing to ensure your mod works from scratch

6. **Document with description** - A good description helps when browsing `mod list`
