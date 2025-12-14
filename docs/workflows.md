# Workflows

Common usage patterns and workflows for Glovebox.

## Daily Development

### Starting a Session

```bash
cd ~/projects/my-app
glovebox run
```

You're dropped into a shell inside the container with your project at `/workspace`.

### Resuming Work

Glovebox containers are persistent. When you `exit` and later run `glovebox run` again, you resume where you left off—including any tools you installed or configuration changes you made.

### Checking Status

```bash
glovebox status
```

Shows your profile configuration, image state, and container state.

## Project-Specific Environments

When a project needs tools beyond your base image:

### 1. Create a Project Profile

```bash
cd ~/projects/special-project
glovebox init
```

This creates `.glovebox/profile.yaml` that extends your base image.

### 2. Add Project-Specific Mods

```bash
glovebox add languages/nodejs
glovebox add tools/tmux
```

### 3. Build and Run

```bash
glovebox build
glovebox run
```

The project image inherits everything from `glovebox:base` and adds the project-specific tools.

## Ad-Hoc Tool Installation

You don't always know what tools you'll need. Glovebox lets you install things during a session and optionally persist them.

### Install During Session

```bash
# Inside the container
brew install jq
brew install httpie
```

### Persist Changes

When you exit, Glovebox shows what changed:

```
dev@glovebox /workspace (main)> exit

  ┃ Session ended · container has uncommitted changes:
  ┃
  ┃   brew install jq
  ┃   brew install httpie
  ┃   ...and 35 more changes
  ┃
  ┃ To persist: glovebox commit
  ┃ To discard: glovebox reset
```

Run `glovebox commit` to bake these tools into your image permanently.

## Working with AI Coding Assistants

Glovebox is ideal for running AI coding assistants that execute arbitrary code.

### Setup

Add an AI assistant to your base profile (edit `~/.glovebox/profile.yaml` or run from a directory without a project profile):

```bash
glovebox add ai/claude-code
glovebox build --base
```

### Configure API Keys

Add your API key to passthrough in your profile (`~/.glovebox/profile.yaml`):

```yaml
passthrough_env:
  - ANTHROPIC_API_KEY
```

Rebuild isn't needed—this takes effect on next `glovebox run`.

### Use

```bash
cd ~/projects/my-app
glovebox run

# Inside container
claude
```

The AI assistant runs sandboxed. It can modify files in `/workspace` (your project), but cannot access your host system, SSH keys, or other projects.

## Testing Untrusted Code

Glovebox provides a safe environment for:

- Evaluating npm packages before adding them to production
- Running scripts from the internet
- Testing code from unfamiliar sources

### Example: Testing an npm Package

```bash
glovebox run

# Inside container
cd /workspace
npm install sketchy-package
node -e "require('sketchy-package')"
```

If the package does something malicious, the damage is contained to the container. Your host is protected.

### Clean Slate Testing

For maximum isolation, erase changes after testing:

```bash
exit
# Choose [e]rase when prompted
```

Next session starts fresh.

## Cloning and Running

For quick exploration of a repository:

```bash
glovebox clone https://github.com/user/repo
```

This clones the repo and immediately starts a Glovebox session in it.

## Multiple Projects

Each project gets its own container and (optionally) its own image:

```
~/projects/
├── app-a/           → container: glovebox-app-a-abc123
├── app-b/           → container: glovebox-app-b-def456
└── special-project/ → container: glovebox-special-project-789xyz
                       image: glovebox:special-project-789xyz
```

Projects without a `.glovebox/profile.yaml` use the base image directly. Projects with a profile get their own extended image.

## Clean Up

### Single Project

```bash
cd ~/projects/my-app
glovebox clean
```

Removes the project's container and image (if any). Base image preserved.

### Full Reset

```bash
glovebox clean --all
```

Removes all Glovebox containers and images, including base. Use this when:

- Something is broken beyond repair
- You want to rebuild everything from scratch
- You're done with Glovebox entirely

After `clean --all`, you'll need to `glovebox build --base` again.

## Workflow Summary

| Scenario | Commands |
|----------|----------|
| Daily use | `gb run` |
| Project needs extra tools | `gb init`, `gb add <mod>`, `gb build`, `gb run` |
| Install tool during session | Install normally, choose [y]es on exit |
| Test untrusted code | `gb run`, test, `exit`, choose [e]rase |
| Start fresh | `gb clean` or `gb clean --all` |
| Quick repo exploration | `gb clone <url>` |
