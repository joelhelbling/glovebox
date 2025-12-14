# Commands Reference

Glovebox provides a `gb` shorthand alias for all commands. Both `glovebox <command>` and `gb <command>` work identically.

## Quick Reference

| Command | Description |
|---------|-------------|
| `glovebox init --base` | Create base profile |
| `glovebox init` | Create project profile |
| `glovebox build --base` | Build base image |
| `glovebox build` | Build project image |
| `glovebox run` | Start sandboxed session |
| `glovebox status` | Show current state |
| `glovebox add <mod>` | Add a mod to profile |
| `glovebox remove <mod>` | Remove a mod from profile |
| `glovebox commit` | Persist container changes to image |
| `glovebox reset` | Discard container changes |
| `glovebox diff` | Show changes in container filesystem |
| `glovebox clean` | Remove project container/image |
| `glovebox clone <repo>` | Clone and start glovebox |
| `glovebox mod list` | List available mods |

## Initialization

### `glovebox init --base`

Creates your base profile at `~/.glovebox/profile.yaml`. Launches an interactive wizard to select:

- Base OS (Ubuntu, Fedora, or Alpine)
- Shell (bash, zsh, fish)
- Editor (vim, neovim, helix, emacs)
- Tools (mise, tmux, homebrew)
- AI assistants (claude-code, gemini-cli, opencode)

Only mods compatible with your selected OS are shown.

### `glovebox init`

Creates a project-specific profile at `.glovebox/profile.yaml` in the current directory. Use this when a project needs tools beyond your base image.

## Building

### `glovebox build --base`

Builds the `glovebox:base` image from your global profile (`~/.glovebox/profile.yaml`). This is your standard development environment used across all projects.

### `glovebox build`

Builds a project-specific image that extends `glovebox:base`. If no project profile exists, falls back to building/rebuilding the base image.

The project image is tagged as `glovebox:<dirname>-<hash>` where the hash is derived from the project path.

### `glovebox build --generate-only`

Generates the Dockerfile without building the image. Useful for debugging or customization.

## Running

### `glovebox run [directory]`

Starts a Glovebox session. If no directory is specified, uses the current directory.

Behavior:
- **First run**: Creates a new container from the appropriate image
- **Subsequent runs**: Starts the existing container, preserving any changes
- **On exit**: Shows a summary of filesystem changes (if any)

The project directory is mounted at `/workspace` inside the container.

### `glovebox clone <repo>`

Clones a git repository and immediately starts a Glovebox session in it.

Repository can be:
- `user/repo` - GitHub shorthand (e.g., `rails/rails`)
- Full URL - GitHub, GitLab, Bitbucket, or any git URL

```bash
glovebox clone rails/rails
glovebox clone https://gitlab.com/user/repo.git
```

## Profile Management

### `glovebox add <mod>`

Adds a mod to your profile. If a project profile exists (`.glovebox/profile.yaml`), adds to that; otherwise adds to the global profile.

For OS-specific mods, you can use the base name and Glovebox will resolve it automatically:

```bash
glovebox add editors/vim       # Resolves to editors/vim-ubuntu on Ubuntu
glovebox add ai/claude-code    # OS-agnostic, adds as-is
```

### `glovebox remove <mod>`

Removes a mod from your profile. Alias: `glovebox rm`

Like `add`, you can use base names for OS-specific mods:

```bash
glovebox remove editors/vim    # Removes editors/vim-ubuntu if installed
glovebox rm shells/zsh         # Removes shells/zsh-ubuntu if installed
```

## Status and Information

### `glovebox status`

Shows the current state of your Glovebox environment:

- Profile locations and contents
- Image status (built, needs rebuild)
- Container status (exists, running)
- Mods in use

## Container Management

### `glovebox commit`

Commits changes from the current project's container to its image. The container is removed and a fresh one will be created on the next `glovebox run`.

Use this after installing tools or making configuration changes you want to keep permanently.

### `glovebox reset`

Discards all changes in the current project's container. The container is removed and a fresh one will be created from the original image on the next `glovebox run`.

Use this when you want to return to a clean state.

### `glovebox diff`

Shows changes in the container's filesystem compared to the original image. Useful for seeing what's been modified before deciding whether to commit or reset.

## Cleanup

### `glovebox clean`

Removes the project container for the current directory. The image is preserved, so any committed changes remain.

### `glovebox clean --image`

Removes both the project container and image. Any user-committed changes to the image will be lost. The base image is preserved.

### `glovebox clean --all`

Removes all Glovebox containers and images, including the base image. Requires confirmation. Use this for a complete reset.

## Mod Commands

### `glovebox mod list`

Lists all available mods organized by category. Alias: `glovebox mod ls`

```
$ glovebox mod list

  ┃ Available Mods
  ┃
  ┃ os/
  ┃   alpine       Alpine Linux base image (lightweight, musl-based)
  ┃   fedora       Fedora base image with core dependencies
  ┃   ubuntu       Ubuntu base image with core dependencies
  ┃
  ┃ shells/
  ┃   bash         Bash shell (default, minimal configuration)
  ┃   fish         Fish shell - the friendly interactive shell (for Alpine, Fedora, Ubuntu)
  ┃   zsh          Z shell with sensible defaults (for Alpine, Fedora, Ubuntu)
  ...
```

Mods marked with `(for Alpine, Fedora, Ubuntu)` have OS-specific variants. When you `add` or `remove` these mods, Glovebox automatically resolves to the correct variant for your profile's OS.

### `glovebox mod cat <id>`

Outputs a mod's raw YAML to stdout. Useful for understanding what a mod does or as a starting point for custom mods.

```bash
glovebox mod cat editors/neovim > ~/.glovebox/mods/editors/neovim.yaml
```

### `glovebox mod create <name>`

Creates a new custom mod from a template. The mod is created in your project's `.glovebox/mods/` directory (or `~/.glovebox/mods/` with `--global`).

```bash
glovebox mod create my-tool          # Creates .glovebox/mods/custom/my-tool.yaml
glovebox mod create tools/my-tool    # Creates .glovebox/mods/tools/my-tool.yaml
glovebox mod create my-tool --global # Creates ~/.glovebox/mods/custom/my-tool.yaml
```
