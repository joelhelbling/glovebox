# Glovebox

A dockerized development sandbox for working with dangerous things like agentic coding tools and npm packages.

## Prerequisites

- Docker

## Installation

Clone this repository and add the `glovebox` script to your PATH:

```bash
ln -s /path/to/glovebox/glovebox ~/.local/bin/glovebox
```

## Usage

```bash
# Start glovebox in the current directory
glovebox

# Start glovebox in a specific directory
glovebox /path/to/project

# Force rebuild the image
glovebox --build
```

The container mounts your project directory and drops you into a Fish shell.

## What's Included

**Shell & Terminal**
- Fish shell (v4)
- tmux + tmuxp

**Editors**
- neovim (latest stable)

**AI Coding Tools**
- Claude Code
- Gemini CLI
- OpenCode

**Version Management**
- mise (for managing language runtimes)

**Build Tools**
- Node.js 22
- build-essential (gcc, make, etc.)
- git, curl, wget, jq

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
