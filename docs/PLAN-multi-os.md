# Multi-OS Support Plan

This document captures the design decisions and implementation plan for adding multi-OS support to Glovebox.

## Motivation

- Homebrew-based installs use significant storage
- Lighter base images exist (Alpine, etc.)
- Not every user wants Ubuntu
- Goal: Support multiple base OSes while providing escape hatches for custom configurations

## Design Decisions

### OS as a Mod

The OS is a first-class mod in the `os/` category. This leverages the existing dependency system:

```yaml
# mods/os/ubuntu.yaml
name: ubuntu
description: Ubuntu base image
category: os
dockerfile_from: ubuntu:24.04
provides: [base]

run_as_root: |
  apt-get update && apt-get install -y \
    git curl wget ca-certificates build-essential sudo
  # ... user setup
```

**Rules:**
- Only one mod from `os` category allowed per profile
- OS mods provide `dockerfile_from` which becomes the `FROM` line
- Other mods declare `requires: [ubuntu]` (or fedora, alpine)

### Supported OSes (Embedded)

Three OSes supported out of the box:
- **Ubuntu** (`ubuntu.yaml`)
- **Fedora** (`fedora.yaml`)
- **Alpine** (`alpine.yaml`)

Users can create custom OS mods for any other base image.

### File Naming Convention

OS-specific mods use a flat structure with suffix:

```
mods/os/ubuntu.yaml
mods/os/fedora.yaml
mods/os/alpine.yaml
mods/shells/zsh-ubuntu.yaml
mods/shells/zsh-fedora.yaml
mods/shells/zsh-alpine.yaml
mods/editors/vim-ubuntu.yaml
...
```

### The `provides` Field

Mods can declare what they provide, enabling abstract dependencies:

```yaml
# zsh-ubuntu.yaml
name: zsh-ubuntu
provides: [zsh]
requires: [ubuntu]
```

```yaml
# oh-my-zsh.yaml (OS-agnostic)
name: oh-my-zsh
requires: [zsh]  # satisfied by zsh-ubuntu, zsh-fedora, or zsh-alpine
```

**Resolution logic:**
- Every mod implicitly provides its own name
- Plus anything in explicit `provides:` list
- `requires:` checks against all provided values

### Simplified Mod Schema

**Remove** `apt_packages`, `apt_repos` fields. Use `run_as_root` for all package installation.

**Rationale:**
- Avoids needing `dnf_packages`, `apk_packages`, etc.
- Custom OS users just use `run_as_root`
- Layer count overhead is negligible
- OS mods can batch common packages for efficiency

**Final schema:**

| Field | Purpose |
|-------|---------|
| `name` | Mod identifier |
| `description` | Human-readable description |
| `category` | Category (`os`, `shells`, `editors`, `tools`, etc.) |
| `dockerfile_from` | Base image (only for `os` category mods) |
| `provides` | What this mod provides (defaults to `[name]`) |
| `requires` | Dependencies (checked against name + provides) |
| `run_as_root` | Commands run as root |
| `run_as_user` | Commands run as user |
| `env` | Environment variables |
| `user_shell` | Sets default shell (for shell mods) |

### Profile UX

- Default "happy path" for three embedded OSes via `init`
- Clear escape hatch: `mod cat` to template custom mods, edit `profile.yaml` directly
- Protect user edits: warn/confirm before overwriting modified profiles
- Enhanced init: offer to open profile in `$EDITOR`, create templated mods

---

## Implementation Phases

### Phase 1: Mod Schema & Dependency Resolution ✅

Update the core mod system to support the new fields and validation.

- [x] Add `dockerfile_from` field to `mod.Mod` struct
- [x] Add `provides` field to `mod.Mod` struct (slice of strings)
- [x] Update mod YAML parsing to handle new fields
- [x] Implement `provides` resolution:
  - [x] Mod provides its own name (implicit) - `EffectiveProvides()` method
  - [x] Mod provides explicit `provides` values
  - [x] Build provides map from all mods in profile - `BuildProvidesMap()` function
- [x] Update dependency checking to use provides map
- [x] Add validation: only one mod in `os` category allowed - `ValidateOSCategory()`
- [x] Add validation: error if mod requires something not provided - `ValidateRequires()`
- [x] Add validation: error if mod requires an OS that isn't the selected one - `ValidateCrossOSDependencies()`

**Tests:**
- [x] Test `provides` resolution (implicit name + explicit provides)
- [x] Test dependency checking against provides
- [x] Test OS category mutual exclusivity validation
- [x] Test cross-OS dependency error (e.g., `vim-fedora` with `ubuntu`)

**Files affected:**
- `internal/mod/mod.go` - struct changes, validation functions
- `internal/mod/mod_test.go` - new tests

### Phase 2: Generator Changes ✅

Update the Dockerfile generator to use the new schema.

- [x] Find `os` category mod and extract `dockerfile_from` for `FROM` line
- [x] Error if zero OS mods present
- [x] Error if multiple OS mods present (handled by `ValidateMods()`)
- [x] Remove `collectAptPackages()` function
- [x] Remove `collectAptRepos()` function
- [x] Remove apt-related Dockerfile generation logic
- [x] Update `GenerateBase()` for new schema
- [x] Update `GenerateProject()` for new schema

**Tests:**
- [x] Test FROM image extraction from OS mod
- [x] Test error on missing OS mod
- [x] Test error on multiple OS mods (via `ValidateMods()`)
- [x] Test Dockerfile generation without apt_packages
- [x] Integration test: full Dockerfile from OS mod + tool mods

**Files affected:**
- `internal/generator/generator.go`
- `internal/generator/generator_test.go`
- `internal/integration/profile_interaction_test.go` - updated to use `os/ubuntu`

**Note:** Created `mods/os/ubuntu.yaml` early (partial Phase 3 work) to enable testing.

### Phase 3: Create OS Mods & Migrate Existing Mods ✅

Create the embedded OS mods and convert mods that need OS-specific package installation.

**Create OS mods:**
- [x] Create `mods/os/ubuntu.yaml` with base Ubuntu setup (done in Phase 2)
- [x] Create `mods/os/fedora.yaml` with base Fedora setup
- [x] Create `mods/os/alpine.yaml` with base Alpine setup

**Create OS-specific shell variants (require package manager installation):**

Shells:
- [x] `bash.yaml` - kept OS-agnostic (bash pre-installed on all OSes), added `provides: [bash]`
- [x] `zsh.yaml` → `zsh-ubuntu.yaml`, `zsh-fedora.yaml`, `zsh-alpine.yaml` with `provides: [zsh]`
- [x] `fish.yaml` → `fish-ubuntu.yaml`, `fish-fedora.yaml`, `fish-alpine.yaml` with `provides: [fish]`

**Keep OS-agnostic mods (use homebrew/mise which works across all OSes):**

Editors (all use `brew install`, kept OS-agnostic with `provides:`):
- [x] `vim.yaml` - added `provides: [vim]`
- [x] `neovim.yaml` - added `provides: [neovim]`
- [x] `helix.yaml` - added `provides: [helix]`
- [x] `emacs.yaml` - added `provides: [emacs]`

Tools (all use homebrew, kept OS-agnostic with `provides:`):
- [x] `homebrew.yaml` - added `provides: [homebrew]`, kept OS-agnostic (uses curl)
- [x] `mise.yaml` - added `provides: [mise]`
- [x] `tmux.yaml` - added `provides: [tmux]`

Languages (use mise, kept OS-agnostic with `provides:`):
- [x] `nodejs.yaml` - added `provides: [nodejs]`

AI tools (all use homebrew, kept OS-agnostic with `provides:`):
- [x] `claude-code.yaml` - added `provides: [claude-code]`
- [x] `gemini-cli.yaml` - added `provides: [gemini-cli]`
- [x] `opencode.yaml` - added `provides: [opencode]`

**Cleanup:**
- [x] Remove old `base.yaml`
- [x] Remove `apt_packages` and `apt_repos` from mod struct
- [x] Remove old `zsh.yaml` and `fish.yaml` (replaced by OS variants)

**Design Decision:** Only mods that require OS-specific package installation (shells needing apt/dnf/apk) need OS variants. Mods using homebrew or mise remain OS-agnostic since those tools work across all Linux distros.

### Phase 4: Command Updates ✅

Update CLI commands to handle multi-OS properly.

**`init` command:**
- [x] Prompt user to select OS (ubuntu, fedora, alpine)
- [x] Show only mods compatible with selected OS
- [x] Auto-select OS mod when user picks OS
- [x] Update default profile generation

**`add` command:**
- [x] Validate mod is compatible with profile's OS
- [x] Suggest correct variant if user tries wrong one (e.g., "Did you mean zsh-ubuntu?")

**`mod list` command:**
- [x] Show OS compatibility info (shows `[ubuntu]`, `[fedora]`, `[alpine]` badges)
- [x] Show `provides` info in ModInfo struct
- [x] Added `os/` category at top of list with all OS options

**`mod create` template:**
- [x] Removed deprecated `apt_packages` and `apt_repos` fields
- [x] Added `provides` field documentation
- [x] Updated `requires` documentation for concrete vs abstract dependencies

**Profile protection:**
- [x] Detect if profile.yaml has been manually edited (ContentHash in BuildInfo)
- [x] Warn before overwriting user-edited profile (stronger warning with yellow color)
- [x] Confirmation required for all overwrites

**Files affected:**
- `cmd/init.go` - OS selection, compatible mod filtering, content hash
- `cmd/add.go` - OS compatibility validation, mod variant suggestions
- `cmd/mod.go` - OS category ordering, RequiresOS display, updated template
- `internal/profile/profile.go` - ContentHash field and detection methods
- `internal/ui/modlist.go` - RequiresOS display support

### Phase 5: Enhanced Init UX

Polish the init experience for power users.

- [ ] Offer to open profile in `$EDITOR` after generation
- [ ] Offer to create a templated custom mod
- [ ] Add `mod new` or `mod create` command for scaffolding custom mods
- [ ] Improve help text explaining customization escape hatches

---

## Migration Notes

### For existing users

Existing profiles reference `base` and mods like `vim`, `zsh`. After this change:
- `base` → `ubuntu` (or chosen OS)
- `vim` → `vim-ubuntu`
- `zsh` → `zsh-ubuntu`

Consider:
- [ ] Migration command or script?
- [ ] Deprecation warnings for old mod names?
- [ ] Keep aliases for transition period?

### Breaking changes

- `apt_packages` field removed from mod schema
- `apt_repos` field removed from mod schema
- Old mod names no longer exist (replaced by OS-suffixed variants)
- `base` mod replaced by OS mods

---

## Open Questions

1. **Homebrew mod** - Does it make sense on Linux? Remove or keep for specific use cases?
2. **Migration path** - How to handle existing profiles gracefully?
3. **Mod aliases** - Should `vim` resolve to `vim-<current-os>` automatically?

---

## Session Notes

*Use this section to track progress across sessions.*

- **2024-12-13**: Initial design discussion. Decided on OS-as-mod approach, `provides` for abstract dependencies, dropping `apt_packages` in favor of `run_as_root`, flat file naming with OS suffix.
- **2024-12-13**: Completed Phase 1. Added `dockerfile_from` and `provides` fields to mod struct. Implemented `EffectiveProvides()`, `BuildProvidesMap()`, `ValidateOSCategory()`, `ValidateRequires()`, `ValidateCrossOSDependencies()`, and `ValidateMods()`. Updated dependency resolution to use provides. All tests passing.
- **2024-12-13**: Completed Phase 2. Updated generator to use OS mod's `dockerfile_from` for FROM line. Removed `collectAptPackages()` and `collectAptRepos()`. Created `mods/os/ubuntu.yaml` (partial Phase 3) to enable testing. Updated generator and integration tests. All tests passing.
- **2024-12-13**: Completed Phase 3. Created `os/fedora.yaml` and `os/alpine.yaml`. Created OS-specific shell variants (`zsh-ubuntu`, `zsh-fedora`, `zsh-alpine`, `fish-ubuntu`, `fish-fedora`, `fish-alpine`). Kept homebrew-based mods OS-agnostic with `provides:` added. Removed `base.yaml` and `apt_packages`/`apt_repos` from mod struct. Updated all tests. All tests passing.
- **2024-12-13**: Completed Phase 4. Updated `init` command with OS selection prompt, compatible mod filtering, and content hash for detecting manual edits. Updated `add` command with OS compatibility validation and variant suggestions. Updated `mod list` to show OS category first and display OS requirement badges. Updated `mod create` template to remove deprecated apt fields. Added profile protection with ContentHash. All tests passing.
