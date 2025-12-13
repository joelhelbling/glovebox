# Help Screens

All help output from the `glovebox` app.

## Root
Command: `gb --help`

```
Glovebox creates sandboxed Docker containers for running untrusted or
experimental code. It uses a mod-based system to compose your perfect
development environment from modular, reusable pieces.

Usage:
  glovebox [command]

Available Commands:
  add         Add a mod to your profile
  build       Generate Dockerfile and build Docker image
  clean       Remove glovebox Docker container (and optionally image)
  clone       Clone a git repository and start glovebox in it
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  init        Initialize a new glovebox profile
  mod         Manage custom mods
  remove      Remove a mod from your profile
  run         Run glovebox container with a mounted directory
  status      Show profile and Dockerfile status

Flags:
  -h, --help      help for glovebox
  -v, --version   version for glovebox

Use "glovebox [command] --help" for more information about a command.
```

## Add
Command: `gb add --help`

```
Add a mod to your glovebox profile.

Run 'glovebox mod list' to see available mods.

To create your own custom mod, run:
  glovebox mod create <name>

Examples:
  glovebox add shells/fish
  glovebox add ai/claude-code
  glovebox add custom/my-tool

Usage:
  glovebox add <mod> [flags]

Flags:
  -h, --help   help for add
```

## Build
Command: `gb build --help`

```
Generate a Dockerfile from your profile mods and build the Docker image.

For the global profile (~/.glovebox/profile.yaml), this builds glovebox:base.
For project profiles (.glovebox/profile.yaml), this builds a project-specific
image that extends glovebox:base.

Use --base to explicitly build only the base image.

If the Dockerfile has been modified since last generation, you'll be prompted
to choose how to proceed.

Usage:
  glovebox build [flags]

Flags:
      --base            Build only the base image (from global profile)
  -f, --force           Force regeneration without prompts
      --generate-only   Only generate Dockerfile, don't build image
  -h, --help            help for build
```

## Clean
Command: `gb clean --help`

```
Remove glovebox Docker container for the current project.

By default, removes only the container (preserving the image):
  - Discards any uncommitted changes in the container
  - Next run creates a fresh container from the existing image
  - Safe: committed changes in the image are preserved

With --image, also removes the project image:
  - Removes both container and image
  - Next run triggers a full image rebuild
  - Warning: any user-committed changes will be lost

With --all, removes everything glovebox-related (requires confirmation):
  - All glovebox:* images
  - All glovebox-* containers

Use --force to skip confirmation prompts.

Usage:
  glovebox clean [directory] [flags]

Flags:
      --all     Remove all glovebox images and containers (requires confirmation)
  -f, --force   Skip confirmation prompts
  -h, --help    help for clean
      --image   Also remove the project image (loses committed changes)
```

## Clone
Command: `gb clone --help`

```
Clone a git repository and start glovebox in the cloned directory.

Repository can be:
  - user/repo    (assumes GitHub, e.g., joelhelbling/glovebox)
  - Full URL     (GitHub, GitLab, Bitbucket, or any git URL)

Examples:
  glovebox clone rails/rails
  glovebox clone https://gitlab.com/user/repo.git

Usage:
  glovebox clone <repository> [flags]

Flags:
  -h, --help   help for clone
```

## Completion
Command: `gb completion --help`

```
Generate the autocompletion script for glovebox for the specified shell.
See each sub-command's help for details on how to use the generated script.

Usage:
  glovebox completion [command]

Available Commands:
  bash        Generate the autocompletion script for bash
  fish        Generate the autocompletion script for fish
  powershell  Generate the autocompletion script for powershell
  zsh         Generate the autocompletion script for zsh

Flags:
  -h, --help   help for completion

Use "glovebox completion [command] --help" for more information about a command.
```

## Completion: Bash
Command: `gb completion bash --help`

```
Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(glovebox completion bash)

To load completions for every new session, execute once:

#### Linux:

	glovebox completion bash > /etc/bash_completion.d/glovebox

#### macOS:

	glovebox completion bash > $(brew --prefix)/etc/bash_completion.d/glovebox

You will need to start a new shell for this setup to take effect.

Usage:
  glovebox completion bash

Flags:
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

## Completion: Fish
Command: `gb completion fish --help`

```
Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	glovebox completion fish | source

To load completions for every new session, execute once:

	glovebox completion fish > ~/.config/fish/completions/glovebox.fish

You will need to start a new shell for this setup to take effect.

Usage:
  glovebox completion fish [flags]

Flags:
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

## Completion: PowerShell
Command: `gb completion powershell --help`

```
Generate the autocompletion script for powershell.

To load completions in your current shell session:

	glovebox completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.

Usage:
  glovebox completion powershell [flags]

Flags:
  -h, --help              help for powershell
      --no-descriptions   disable completion descriptions
```

## Completion: Zsh
Command: `gb completion zsh --help`

```
Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(glovebox completion zsh)

To load completions for every new session, execute once:

#### Linux:

	glovebox completion zsh > "${fpath[1]}/_glovebox"

#### macOS:

	glovebox completion zsh > $(brew --prefix)/share/zsh/site-functions/_glovebox

You will need to start a new shell for this setup to take effect.

Usage:
  glovebox completion zsh [flags]

Flags:
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions
```

## Help
Command: `gb help --help`

```
Help provides help for any command in the application.
Simply type glovebox help [path to command] for full details.

Usage:
  glovebox help [command] [flags]

Flags:
  -h, --help   help for help
```

## Init
Command: `gb init --help`

```
Initialize a new glovebox profile interactively.

Use --base to create the base image profile (~/.glovebox/profile.yaml).
This defines your standard development environment with your preferred
shell, editor, and tools. Build it once with 'glovebox build --base'.

Without --base, creates a project-specific profile (.glovebox/profile.yaml)
that extends the base image with additional tools for that project.

Usage:
  glovebox init [flags]

Flags:
  -b, --base   Create base profile instead of project-local
  -h, --help   help for init
```

## Mod
Command: `gb mod --help`

```
Manage custom mods for your glovebox environment.

Mods are YAML files that define tools, packages, and configurations
to include in your Docker image. Custom mods can be created in:

  ~/.glovebox/mods/       Global mods (available everywhere)
  .glovebox/mods/         Project-local mods (this project only)

Local mods take precedence over embedded ones, so you can also
override built-in mods if needed.

Usage:
  glovebox mod [command]

Available Commands:
  cat         Output a mod's raw YAML content
  create      Create a new custom mod
  list        List available mods

Flags:
  -h, --help   help for mod

Use "glovebox mod [command] --help" for more information about a command.
```

## Mod: Cat
Command: `gb mod cat --help`

```
Output the raw YAML content of a mod to stdout.

This is useful for inspecting mods or creating custom overrides:

  # View a mod
  glovebox mod cat ai/claude-code

  # Copy to local mods and customize
  glovebox mod cat ai/claude-code > .glovebox/mods/ai/claude-code.yaml

The command respects the mod load order (local > global > embedded),
so it shows the version that would actually be used.

Usage:
  glovebox mod cat <mod-id> [flags]

Flags:
  -h, --help   help for cat
```

## Mod: Create
Command: `gb mod create --help`

```
Create a new custom mod with a starter template.

The mod name can include a category prefix (e.g., "tools/mytool").
Without --global, creates in .glovebox/mods/ (project-local).
With --global, creates in ~/.glovebox/mods/ (available everywhere).

Examples:
  glovebox mod create my-tool           # Creates custom/my-tool.yaml
  glovebox mod create tools/my-tool     # Creates tools/my-tool.yaml
  glovebox mod create my-tool --global  # Creates in ~/.glovebox/mods/

Usage:
  glovebox mod create <name> [flags]

Flags:
  -g, --global   Create in global mods directory
  -h, --help     help for create
```

## Mod: List
Command: `gb mod list --help`

```
List all available mods that can be added to your glovebox profile.

This shows built-in mods plus any custom mods found in:
  ~/.glovebox/mods/       Global custom mods
  .glovebox/mods/         Project-local custom mods

To create a custom mod, run:
  glovebox mod create <name>

Usage:
  glovebox mod list [flags]

Aliases:
  list, ls

Flags:
  -h, --help   help for list
```

## Remove
Command: `gb remove --help`

```
Remove a mod from your glovebox profile.

Example:
  glovebox remove ai/opencode
  glovebox rm shells/zsh

Usage:
  glovebox remove <mod> [flags]

Aliases:
  remove, rm

Flags:
  -h, --help   help for remove
```

## Run
Command: `gb run --help`

```
Run the glovebox container with the specified directory mounted as workspace.

If no directory is specified, the current directory is used.

The command will:
1. Check for a project profile (.glovebox/profile.yaml) and use that image
2. Fall back to glovebox:base if no project profile exists
3. Build images automatically if they don't exist

Each project gets its own persistent container. Any changes you make to the
container (installing tools, configuring editors, etc.) are preserved in the
container's writable layer. After exiting, you'll be prompted to commit
changes to the image if any were detected.

Usage:
  glovebox run [directory] [flags]

Flags:
  -h, --help   help for run
```

## Status
Command: `gb status --help`

```
Show the current status of your glovebox profiles, images, and Dockerfiles.

Usage:
  glovebox status [flags]

Flags:
  -h, --help   help for status
```

