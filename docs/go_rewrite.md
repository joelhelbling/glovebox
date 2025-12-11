# Glovebox: Bash to Go Rewrite

## What the Bash Version Had

The original ~150 lines of bash provided:

- Basic `glovebox` and `glovebox clone` commands
- Single hardcoded Dockerfile
- `--build` flag to rebuild
- Volume persistence for mise only

## What the Go Version Added

The Go rewrite expanded the codebase to ~2,700 lines, transforming a simple "run Docker" script into a proper composable development environment builder.

### 1. Composable Snippet System

- **14 built-in snippets** across 5 categories:
  - Shells: bash, zsh, fish
  - Editors: vim, neovim, helix
  - Tools: mise, tmux, homebrew
  - Languages: nodejs
  - AI: claude-code, gemini-cli, opencode
- YAML-based snippet definitions with apt repos, packages, root/user commands, and environment variables
- Snippets embedded in binary via `//go:embed` for single-binary distribution

### 2. Three-Tier Snippet Loading

Snippets are loaded from multiple locations in priority order:

1. **Project-local**: `.glovebox/snippets/`
2. **User global**: `~/.glovebox/snippets/`
3. **Embedded** (bundled in binary)

This allows users to override or extend built-in snippets without rebuilding from source.

### 3. Layered Image Architecture

- **Base image** (`glovebox:base`): User's standard environment built from `~/.glovebox/profile.yaml`
- **Project images**: Extend the base with project-specific tools from `.glovebox/profile.yaml`
- Smart auto-detection of which image to use when running

### 4. Full CLI with 8 Commands

| Command | Description |
|---------|-------------|
| `glovebox init [--base]` | Initialize profiles with interactive snippet selection |
| `glovebox build [--base]` | Generate Dockerfiles and build images |
| `glovebox run` | Run containers with proper volume mounts |
| `glovebox add <snippet>` | Add snippets to profiles |
| `glovebox remove <snippet>` | Remove snippets from profiles |
| `glovebox list` | List available snippets with descriptions |
| `glovebox status` | Show current profile configurations |
| `glovebox clone <repo>` | Clone repos and initialize glovebox |
| `glovebox snippet create <name>` | Scaffold custom snippets |

### 5. Profile Management

- YAML profiles track selected snippets
- Automatic dependency resolution (snippets can require other snippets)
- Content-based image tagging with SHA hashes for cache invalidation

### 6. Home Directory Persistence

- Full `~/.local` volume persistence (not just mise)
- Credentials and config files preserved across sessions
