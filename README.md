# Glovebox

![Glovebox](glovebox-1.jpg)

A composable, dockerized development sandbox for working with dangerous things like agentic coding tools and npm packages.

## Prerequisites

- Docker
- Go 1.23+ (for building from source)

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
# Initialize a profile (interactive snippet selection)
glovebox init

# Generate Dockerfile and build the image
glovebox build

# Run glovebox in the current directory
glovebox run

# Or run in a specific directory
glovebox run /path/to/project
```

## Commands

| Command | Description |
|---------|-------------|
| `glovebox init` | Initialize a new profile (interactive) |
| `glovebox init --global` | Create a global profile (~/.glovebox/profile.yaml) |
| `glovebox list` | List all available snippets |
| `glovebox add <snippet>` | Add a snippet to your profile |
| `glovebox remove <snippet>` | Remove a snippet from your profile |
| `glovebox build` | Generate Dockerfile and build Docker image |
| `glovebox build --generate-only` | Only generate Dockerfile, don't build |
| `glovebox status` | Show profile and Dockerfile status |
| `glovebox run [directory]` | Run glovebox container |
| `glovebox clone <repo>` | Clone a repo and start glovebox in it |

## Composable Snippets

Glovebox uses a snippet-based system to compose your perfect development environment. Instead of a monolithic Dockerfile, you select the components you need:

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

### Profile Management

Your profile is stored in `.glovebox/profile.yaml` (project-local) or `~/.glovebox/profile.yaml` (global):

```yaml
version: 1
snippets:
  - base
  - shells/fish
  - editors/neovim
  - tools/tmux
  - tools/mise
  - ai/claude-code
```

### Modifying Your Environment

```bash
# Add a snippet
glovebox add ai/gemini-cli

# Remove a snippet
glovebox remove ai/opencode

# Regenerate Dockerfile after changes
glovebox build
```

### Dockerfile Tracking

Glovebox tracks the generated Dockerfile with a digest. If you manually edit the Dockerfile, glovebox will detect it and offer options:

```
$ glovebox build
⚠ Dockerfile has been modified since last generation

--- Current Dockerfile
+++ Generated Dockerfile
@@ -143,5 +143,3 @@
...

To preserve your manual changes:
  1. Create a snippet file in snippets/custom/<name>.yaml
  2. Add your changes to the appropriate section
  3. Run: glovebox add custom/<name>
  4. Run: glovebox build

Proceed? [r]egenerate / [k]eep changes / [a]bort:
```

## Persistence

Glovebox containers are ephemeral - most changes are lost when you exit. However, **mise installations persist** via Docker volumes.

Each project gets its own mise volume named `glovebox-<dirname>-<hash>`, so languages/tools you install with mise will be available next time you run glovebox in that directory.

To install a runtime:

```bash
mise use node@22
mise use ruby@3.4
mise use python@3.12
```

## API Keys

The following environment variables are passed through to the container:

- `ANTHROPIC_API_KEY`
- `OPENAI_API_KEY`
- `GOOGLE_API_KEY`
- `GEMINI_API_KEY`

Additionally, these config directories are mounted from your host:

- `~/.anthropic` → `/home/ubuntu/.anthropic`
- `~/.config/gemini` → `/home/ubuntu/.config/gemini`

## Creating Custom Snippets

Create a YAML file in `snippets/custom/` with the following structure:

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

Then add it to your profile:

```bash
glovebox add custom/my-tool
glovebox build
```
