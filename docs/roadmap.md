# Roadmap

Features under consideration for future releases.

## Dotfiles Integration

Automatically sync your dotfiles into the container. This would allow seamless use of your existing shell configuration, git aliases, and tool settings.

Potential approaches:
- Mount dotfiles repository
- Copy dotfiles at container creation
- Git-based sync on container start

## SSH Key Forwarding

Securely access your SSH keys for git operations without copying private keys into the container.

Potential approaches:
- SSH agent forwarding
- Mounted SSH socket
- Per-session key injection

## Networking Affordances

Better integration with host services and other containers:

- Connect to services running on the host (Ollama, LM Studio, databases)
- Link multiple Glovebox containers for microservices development
- Configurable network isolation levels

## GPU Passthrough

Access host GPU for local AI model inference. This would enable running local LLMs inside Glovebox containers.

Considerations:
- NVIDIA Container Toolkit integration
- AMD ROCm support
- Apple Silicon GPU access (limited by Docker)

## Additional OS Support

Expand beyond Ubuntu, Fedora, and Alpine:

- Debian
- Arch Linux
- NixOS

## Resource Limits

Optional CPU and memory limits for containers:

- Prevent runaway processes from affecting the host
- Simulate resource-constrained environments
- Per-project resource profiles

---

Have a feature request? [Open an issue](https://github.com/joelhelbling/glovebox/issues) on GitHub.
