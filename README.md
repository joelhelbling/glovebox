# Glovebox

A composable, dockerized development sandbox for working with dangerous things like agentic coding tools and npm packages.

![Glovebox](glovebox-1.jpg)

## Why Glovebox?

AI coding assistants are powerful, but they run code. So do npm packages, pip installs, and that sketchy shell script you found on Stack Overflow. Running untrusted code on your development machine is a risk—but constantly spinning up VMs or fighting with container configs kills your flow.

Glovebox gives you a sandboxed Docker environment that actually feels like home. Your shell, your editor, your tools—all running safely inside a container with your project mounted. Think of it as glamping on Jurassic Island: even in mortal danger, you still get your Nespresso.

**What makes it different:**

- **Composable mods** - Mix and match shells, editors, languages, and AI tools
- **Layered images** - Build once, extend per-project
- **Persistent containers** - Your changes survive between sessions
- **Commit workflow** - Optionally save ad-hoc changes back to the image

## Prerequisites

- Docker
- Go 1.25+ (for building from source)

## Installation

```bash
brew tap joelhelbling/glovebox
brew install glovebox
```

This installs the `glovebox` command and a `gb` shorthand alias.

For other installation options, see [Getting Started](docs/getting-started.md).

## Quick Start

### One-Time Setup

Create and build your base environment:

```bash
glovebox init --base    # Select OS, shell, editor, tools
glovebox build --base   # Build the base image
```

### Daily Use

Run glovebox in any project directory:

```bash
cd ~/projects/my-app
glovebox run
```

You're now inside a sandboxed container with your project mounted at `/workspace`.

### Clean Up

Remove all glovebox containers and images:

```bash
glovebox clean --all
```

For more commands like `status`, `add`, `remove`, and `clone`, see the [Commands Reference](docs/commands.md).

## Is This For Me?

**Glovebox is for you if:**

- You run AI coding assistants and want to limit the blast radius
- You evaluate npm packages, pip installs, or random scripts before trusting them
- You want a consistent dev environment across projects without VM overhead
- You're a hacker (in the good, MIT sense) who experiments with potentially hazardous stuff

**Glovebox is NOT:**

- Infrastructure for production environments
- A security solution for deployed code
- A replacement for proper sandboxing in CI/CD

Glovebox is a **personal workbench tool**. It doesn't go "in your code" and doesn't run on your production server. It's the toolbox on your workbench where you safely tinker with the unknown.

For secure infrastructure aimed at running AI-generated code in production, check out [Daytona](https://www.daytona.io).

## Documentation

- [Getting Started](docs/getting-started.md) - Installation and first run
- [Commands Reference](docs/commands.md) - All available commands
- [Architecture](docs/architecture.md) - How layered images and container persistence work
- [Composable Mods](docs/mods.md) - Available mods and how they work
- [Custom Mods](docs/custom-mods.md) - Create your own mods
- [Workflows](docs/workflows.md) - Common usage patterns
- [Configuration](docs/configuration.md) - Profiles and environment variables
- [Roadmap](docs/roadmap.md) - Future plans

## Contributing

Contributions are welcome! Here's how to get started:

### Development Setup

```bash
git clone https://github.com/joelhelbling/glovebox.git
cd glovebox
make build
```

### Common Commands

```bash
make build    # Build binary with version from git tags
make test     # Run tests
make lint     # Run fmt and vet
make all      # Run lint, test, and build
```

### Testing Changes

```bash
./bin/glovebox build --base    # Build base image
./bin/glovebox run             # Test in a project directory
./bin/glovebox clean --all     # Clean up
```

### Submitting Changes

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `make all` to ensure tests pass
5. Submit a pull request

For bug reports and feature requests, [open an issue](https://github.com/joelhelbling/glovebox/issues).

## License

[MIT](LICENSE)
