# Architecture

Glovebox uses a layered image approach with persistent containers to balance reproducibility with flexibility.

## Layered Images

```
┌─────────────────────────────┐
│     Project Image           │  ← Project-specific tools
│  (glovebox:myproject-abc123)│     FROM glovebox:base
├─────────────────────────────┤
│     Base Image              │  ← Your standard environment
│     (glovebox:base)         │     Shell, editor, mise, etc.
├─────────────────────────────┤
│  Ubuntu / Fedora / Alpine   │  ← Choose your base OS
└─────────────────────────────┘
```

### Base Image (`glovebox:base`)

Your standard development environment, defined in `~/.glovebox/profile.yaml`. Contains your preferred:

- Shell (bash, zsh, fish)
- Editor (vim, neovim, helix, emacs)
- Common tools (mise, tmux, homebrew)
- AI assistants (claude-code, gemini-cli, opencode)

Build once with `glovebox build --base`, then use across all projects.

### Project Images

Optional project-specific extensions defined in `.glovebox/profile.yaml`. These images:

- Extend `glovebox:base` (using `FROM glovebox:base`)
- Add only the tools needed for that specific project
- Are tagged as `glovebox:<dirname>-<hash>`

Most projects don't need a project image—the base image is sufficient.

## Container Persistence

Each project gets its own persistent container. Glovebox does not use `--rm`.

### Container Lifecycle

1. **First run**: Creates a new container from your image
2. **Subsequent runs**: Starts the existing container (preserving all changes)
3. **On exit**: Shows a summary of any filesystem changes

This means you can install tools, configure editors, and customize your environment during a session—then decide later whether to persist those changes.

### Exit Workflow

When you exit a Glovebox session, it detects changes and shows a summary:

```
dev@glovebox /workspace (main)> brew install yq
🍺  /home/linuxbrew/.linuxbrew/Cellar/yq/4.50.1: 10 files, 13MB

dev@glovebox /workspace (main)> exit

  ┃ Session ended · container has uncommitted changes:
  ┃
  ┃   brew install yq
  ┃   added /home/linuxbrew/.linuxbrew/var/homebrew/tmp
  ┃   modified /home/linuxbrew/.linuxbrew/var/homebrew/linked
  ┃   added /home/linuxbrew/.linuxbrew/var/homebrew/linked/yq
  ┃   ...and 20 more changes
  ┃
  ┃ To persist: glovebox commit
  ┃ To discard: glovebox reset
```

Your changes remain in the container until you decide what to do with them.

**Commands:**

| Command | Container | Image | Next Run |
|---------|-----------|-------|----------|
| `glovebox commit` | Removed | Updated | Fresh container from updated image |
| `glovebox reset` | Removed | Unchanged | Fresh container from original image |
| *(do nothing)* | Preserved | Unchanged | Resume same container with changes |

### When to Commit vs. Reset

**Commit** when you've installed something you want permanently:
- New CLI tools (`brew install jq`)
- Editor plugins
- Configuration changes you want to keep

**Do nothing** when you're still experimenting:
- Trying out a tool you're not sure about
- Making temporary configuration changes
- Debugging with extra packages installed

**Reset** when something went wrong:
- Broke your configuration
- Installed something that conflicts
- Want a clean slate

## Design Philosophy

Glovebox balances two concerns:

1. **Declarative base** - Mods in your profile define your standard environment, baked into images at build time
2. **Flexible runtime** - Ad-hoc changes during sessions can be committed back to the image

| What | How | When |
|------|-----|------|
| Shells, editors, tools | Mods in profile | Build time |
| Language runtimes | Mise (via mod) | Build time |
| Ad-hoc installs | Commit workflow | Runtime |
| Your code | Mounted from host | Always current |

### Source of Truth

Your profile defines what *should* be installed. The container's writable layer captures any runtime additions.

If you `glovebox clean --all` and rebuild, you get exactly what your profile specifies—nothing more, nothing less.

### Why Not Just Use Mods for Everything?

Mods are great for tools you know you want. But development is exploratory:

- You discover you need `jq` to parse some API response
- You want to try `htop` to debug a performance issue
- You install a language server for a file type you rarely edit

The commit workflow lets you capture these discoveries without editing YAML files mid-session. If the tool proves useful, you can later add a proper mod for it.

## Security Model

Glovebox provides isolation, not invulnerability:

- **Container boundary**: Your host filesystem is protected from rogue code
- **Mounted workspace**: Your project files are accessible (read/write)
- **Network access**: Containers have normal network access by default

Think of it as a blast radius limiter. If an npm package or AI agent goes rogue, the damage is contained to the container. Your host system, SSH keys, and other projects remain safe.

What Glovebox does *not* provide:
- Protection against container escape exploits
- Network isolation
- Resource limits (CPU, memory)

For higher security needs, consider running Glovebox inside a VM.
