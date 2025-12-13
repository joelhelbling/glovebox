# Usage Screens

## Exiting the Container

```
ubuntu@glovebox /glovebox (main)> exit

Changes detected in container:
  6 changes in /home/ubuntu

What would you like to do?
  [y]es   - commit changes to image (fresh container next run)
  [n]o    - keep uncommitted changes (resume this container next run)
  [e]rase - discard changes (fresh container next run)
Choice:
```

### `[y]` Commit changes to image

```
Choice: y
Changes committed to glovebox:glovebox-e7b8fab
```

### `[n]` Keep uncommitted changes

```
Choice: n
Changes kept in container.
```

### `[e]` Erase uncommitted changes

```
Choice: e
Container removed. Next run will start fresh.
```

## Status
**Command:** `glovebox status`

```
Base Image:
  Image: glovebox:base ✓
  Profile: ~/.glovebox/profile.yaml
  Mods: 5
    - base
    - shells/fish
    - tools/homebrew
    - editors/neovim
    - tools/mise
  Dockerfile: ~/.glovebox/Dockerfile
    Status: Up to date ✓
    Last built: 2025-12-12 08:43:37 EST

Project Image:
  Image: glovebox:glovebox-e7b8fab ✓
  Profile: ~/code/devex/glovebox/.glovebox/profile.yaml
  Mods: 1
    - ai/claude-code
  Dockerfile: ~/code/devex/glovebox/.glovebox/Dockerfile
    Status: Up to date ✓
    Last built: 2025-12-12 08:44:46 EST

Container:
  Workspace: ~/code/devex/glovebox → /glovebox
  Container: glovebox-glovebox-e7b8fab
    Status: Will be created on first run
```
