# Getting Started

This guide covers installation options and getting your first Glovebox environment running.

## Prerequisites

- Docker
- Go 1.25+ (only if building from source)

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap joelhelbling/glovebox
brew install glovebox
```

This installs the `glovebox` command and a `gb` shorthand alias.

### From Source

```bash
git clone https://github.com/joelhelbling/glovebox.git
cd glovebox
make build
```

Then add the `bin` directory to your PATH, or install system-wide:

```bash
make install  # installs to /usr/local/bin with gb symlink
```

## First Run

### 1. Create Your Base Environment

Run the interactive setup to choose your OS, shell, editor, and tools:

```bash
glovebox init --base
```

You'll be prompted to select:
- **Base OS**: Ubuntu, Fedora, or Alpine
- **Shell**: bash, zsh, or fish
- **Editor**: vim, neovim, helix, or emacs
- **Tools**: mise (runtime manager), tmux, homebrew, etc.
- **AI assistants**: claude-code, gemini-cli, opencode

Your selections are saved to `~/.glovebox/profile.yaml`.

### 2. Build the Base Image

```bash
glovebox build --base
```

This generates a Dockerfile and builds your `glovebox:base` image. This step takes a few minutes the first time, but you only need to do it once (or when you change your base profile).

### 3. Run Glovebox

Navigate to any project directory and start a sandboxed session:

```bash
cd ~/projects/my-app
glovebox run
```

You're now inside a Docker container with your project mounted at `/workspace`. Your shell, editor, and tools are all available.

### 4. Exit and Clean Up

When you're done, just exit the shell:

```bash
exit
```

Glovebox will detect any filesystem changes and ask if you want to commit them to the image. See [Architecture](architecture.md) for details on container persistence.

To remove all Glovebox containers and images:

```bash
glovebox clean --all
```

## Next Steps

- [Commands Reference](commands.md) - Full command documentation
- [Composable Mods](mods.md) - Available mods and how they work
- [Workflows](workflows.md) - Common usage patterns
- [Custom Mods](custom-mods.md) - Create your own mods
