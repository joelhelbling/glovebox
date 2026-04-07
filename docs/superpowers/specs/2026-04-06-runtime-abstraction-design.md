# Runtime Abstraction Layer: Apple Containers + Multi-Runtime Support

## Context

Docker containers share a kernel with the host system, which presents a security concern for glovebox's sandboxed development use case. Apple Containers (macOS) provide hardware-level isolation via micro-VMs -- each container gets its own kernel, with sub-second startup on Apple Silicon.

Glovebox currently hardcodes Docker with 28 `exec.Command("docker", ...)` calls across `cmd/` files. This design introduces a runtime abstraction layer that:

1. Enables Apple Containers as the preferred runtime on macOS (with Docker fallback)
2. Prepares the codebase for future runtimes (e.g., Podman, containerd)
3. Migrates incrementally with no behavior change until the abstraction is complete

## Apple Containers: Key Facts

- **Security**: Each container runs in its own micro-VM via Apple's Virtualization.framework. Hardware-level isolation, no shared kernel.
- **OCI compatible**: Uses standard Dockerfiles, pulls from any OCI registry.
- **CLI**: `container` command (install via `brew install --cask container`).
- **Build DNS bug**: Was a known issue (GitHub #656) -- DNS failed during `container build` RUN steps. Now fixed (PR #1370). **Needs physical testing** on macOS 15 and 26 to confirm.
- **Platform**: Apple Silicon only, macOS 15+ (best on macOS 26/Tahoe). Pre-1.0 (v0.11.0).
- **Limitations**: No Docker Compose equivalent, pre-1.0 stability. See "Verified Capabilities and Known Gaps" below.

## Verified Capabilities and Known Gaps

Based on research of the official command reference, GitHub issues, and documentation (verified 2026-04-07).

### Confirmed Working

| Capability | Docker equivalent | Apple Containers syntax | Notes |
|-----------|------------------|------------------------|-------|
| Build with tag | `docker build -t name -f path dir` | `container build -t name -f path dir` | Identical syntax. Builder runs as separate container. |
| Interactive run | `docker run -it` | `container run -it` | Identical flags |
| Named containers | `docker run --name n` | `container run --name n` | Also sets hostname implicitly |
| Bind mounts | `docker run -v h:c` | `container run -v h:c` | Identical syntax. Also supports `--mount` |
| Env vars | `docker run -e K=V` | `container run -e K=V` | Also supports `--env-file` |
| Working directory | `docker run -w /path` | `container run -w /path` | Also accepts `--cwd` |
| Container persistence | Containers survive exit | Same (unless `--rm`) | `container ls -a` shows stopped |
| Restart stopped | `docker start -ai name` | `container start -a -i name` | Same effect, slightly different flag style |
| Exec in running | `docker exec -it name cmd` | `container exec -it name cmd` | Identical |
| Image inspect | `docker image inspect name` | `container image inspect name` | Returns detailed JSON |
| Image list | `docker images` | `container image ls` | Shows NAME, TAG, DIGEST |
| Image remove | `docker rmi name` | `container image rm name` | Confirmed |
| Container remove | `docker container rm name` | `container rm name` | Confirmed |
| Version check | `docker --version` | `container --version` | Single-line output |
| Export filesystem | `docker export -o file.tar name` | `container export -o file.tar name` | Exports stopped container |

### Key Differences from Docker

| Feature | Docker | Apple Containers | Impact on Glovebox |
|---------|--------|-----------------|-------------------|
| **No `attach` command** | `docker attach name` | Does not exist. Use `container start -a -i` (stopped) or `container exec -it` (running). | `Attach()` method must use `exec -it name <shell>` |
| **No `--hostname` flag** | `docker run --hostname h` | Does not exist. `--name` implicitly sets hostname. | `RunConfig.Hostname` is a no-op; use container name as hostname |
| **No `--filter` on list** | `docker ps --filter name=X` | Does not exist. Use `--format json` and filter in code. | `ListContainers`/`ListImages` must fetch all + filter in Go |
| **No `diff` command** | `docker diff name` | Does not exist | Confirmed -- capability flag needed |
| **No `commit` command** | `docker commit name img` | Does not exist (issues #1019, #1399) | Confirmed -- capability flag needed. Workaround: `export` + rebuild |
| **Builder is separate** | BuildKit integrated | Must `container builder start` before first build. Default: 2 CPU, 2GB RAM. | Glovebox should auto-start builder if not running |
| **Resource allocation** | Shared kernel, no per-container limits by default | Each container is a VM: default 4 CPU, 1GB RAM | May need `--cpus`/`--memory` flags for heavy workloads |

### Known Issues (from GitHub)

| Issue | Severity | Detail |
|-------|----------|--------|
| **UID/GID mismatch on bind mounts** (issue #165, open) | **High** | Files in mounts appear as root inside container. No `--userns keep-id` yet. Glovebox runs as `dev` user -- mounted workspace files may have wrong ownership. |
| **Slow bind mount throughput** (issue #948) | Medium | Filesystem I/O through mounts is significantly slower than Docker. |
| **Single file mounts fail intermittently** (issue #1251, open) | Low | Glovebox mounts directories, not single files, so likely not affected. |
| **Relative paths in mounts** (issue #618, closed/fixed) | None | Was broken, now fixed. Glovebox uses absolute paths anyway. |

### Physical Testing Results

| # | Test | macOS 26 (Tahoe) | macOS 15 (Sequoia) |
|---|------|-------------------|-------------------|
| 1 | DNS during `container build` | PASS | Pending |
| 2 | Volume mount UID/GID as non-root user | PASS | Pending |
| 3 | `container start -a -i` reattaches TTY | PASS | Pending |
| 4 | Env var passthrough | PASS | Pending |
| 5 | Signal forwarding (Ctrl+C) | PASS | Pending |
| 6 | Builder auto-start behavior | PASS | Pending |
| 7 | Container name as hostname | PASS | Pending |

**macOS 26**: All 7 tests passed (2026-04-07). No blockers on Tahoe.
**macOS 15**: Testing in progress.

## Runtime Interface

File: `internal/runtime/runtime.go`

```go
package runtime

import "io"

// Runtime abstracts a container runtime (Docker, Apple Containers, etc.)
type Runtime interface {
    // Name returns the human-readable runtime name (e.g., "Docker", "Apple Containers")
    Name() string

    // --- Image operations ---
    ImageExists(name string) bool
    GetImageDigest(name string) (string, error)
    BuildImage(dockerfilePath, contextDir, imageName string) error
    RemoveImage(name string) error
    ListImages(filterRef string) ([]string, error)

    // --- Container lifecycle ---
    ContainerExists(name string) bool
    ContainerRunning(name string) bool
    RunInteractive(cfg RunConfig) error
    StartInteractive(name string) error
    Attach(name string) error
    RemoveContainer(name string) error
    ForceRemoveContainer(name string) error
    ListContainers(filterName string, all bool) ([]ContainerInfo, error)

    // --- Container state inspection ---
    Diff(name string) ([]FileDiff, error)
    Commit(containerName, imageName string) error

    // --- Capabilities ---
    Capabilities() Capabilities
}
```

### Supporting Types

```go
type RunConfig struct {
    ContainerName  string
    ImageName      string
    HostPath       string
    WorkspacePath  string
    Env            map[string]string // pre-resolved key=value pairs
    Hostname       string            // Docker uses --hostname; Apple Containers uses --name (which sets hostname implicitly)
}

type ContainerInfo struct {
    Name  string
    Image string
}

type FileDiff struct {
    ChangeType string // "A" (added), "C" (changed), "D" (deleted)
    Path       string
}

type Capabilities struct {
    SupportsDiff   bool
    SupportsCommit bool
    SupportsExport bool // `container export` -- available in both, useful as commit workaround
}
```

### Stdio Wiring

Interactive methods (`RunInteractive`, `StartInteractive`, `Attach`) need terminal access. Each runtime struct holds stdio references set at construction:

```go
func NewDocker(stdin io.Reader, stdout, stderr io.Writer) *DockerRuntime
func NewApple(stdin io.Reader, stdout, stderr io.Writer) *AppleRuntime
```

### Exit Code Handling

The `ignoreExitError` logic is Docker-specific (exit codes 125-127, 137). Each runtime handles exit code normalization internally -- interactive methods return `nil` for clean exits and meaningful errors for real failures.

## Package Structure

```
internal/
  runtime/
    runtime.go          # Interface + types (RunConfig, ContainerInfo, FileDiff, Capabilities)
    detect.go           # Auto-detection + factory
    docker.go           # Docker implementation
    apple.go            # Apple Containers implementation
    docker_test.go
    apple_test.go
    detect_test.go
  docker/
    docker.go           # Slimmed to: ContainerName(), ImageName() (pure functions only)
    docker_test.go
```

## Runtime Detection and Selection

File: `internal/runtime/detect.go`

```go
type DetectResult struct {
    Runtime     Runtime
    FellBack    bool
    FallbackMsg string
}

func Detect(override string) (DetectResult, error)
```

### Detection Order

1. **If `--runtime` flag or config override is set**: use that runtime. Error if unavailable.
2. **Check Apple Containers**: `exec.LookPath("container")` then verify `container --version` responds (avoids false positives from other binaries named `container`).
3. **Check Docker**: `exec.LookPath("docker")` then verify daemon is running via `docker info`.
4. **Neither available**: return clear error listing what to install.

### Fallback Message

When Apple Containers is not available and Docker is used:

```
Note: Using Docker runtime. For better isolation, install Apple Containers:
  brew install --cask container
Docker containers share a kernel with the host system. Apple Containers
run each container in its own virtual machine for hardware-level isolation.
```

### Integration Point

In `cmd/root.go`, add a `PersistentPreRunE` that calls `runtime.Detect()` and stores the result in a package-level `rt` variable. Add `--runtime` as a persistent flag on the root command.

## Handling the Diff/Commit Gap

Apple Containers may not support `diff` or `commit`. The interface handles this via `Capabilities()`:

| Feature | Docker | Apple Containers (initial) |
|---------|--------|---------------------------|
| `Diff` | Yes | No -- returns `ErrNotSupported` |
| `Commit` | Yes | No -- returns `ErrNotSupported` |

### Graceful Degradation

| Call site | With diff/commit | Without |
|-----------|-----------------|---------|
| `glovebox run` post-exit | Shows change summary | Shows "Session ended" (no summary) |
| `glovebox diff` | Shows changes | "diff not supported by {runtime name}" |
| `glovebox commit` | Commits to image | "commit not supported by {runtime name}" |
| `glovebox status` | Shows "N uncommitted changes" | Omits that line |

### Future Persistence for Apple Containers

Investigate later:
- Volume snapshots
- Filesystem export/import
- Rebuilding image with changes baked in

## Apple Containers CLI Mapping (Verified)

| Glovebox operation | Docker command | Apple Containers equivalent |
|-------------------|---------------|---------------------------|
| Build image | `docker build -t name -f path dir` | `container build -t name -f path dir` (must ensure builder is running) |
| Run container | `docker run -it --name n -v h:c -w w --hostname h -e K=V img` | `container run -it --name n -v h:c -w w -e K=V img` (no `--hostname`; `--name` sets hostname) |
| Attach to running | `docker attach name` | `container exec -it name /bin/sh` (no `attach` command) |
| Start stopped | `docker start -ai name` | `container start -a -i name` |
| Diff | `docker diff name` | Not available |
| Commit | `docker commit container image` | Not available (workaround: `container export` + rebuild) |
| Remove container | `docker container rm name` | `container rm name` |
| Remove image | `docker rmi name` | `container image rm name` |
| List containers | `docker container ls -a --filter name=X` | `container ls -a --format json` + filter in Go |
| List images | `docker images --filter reference=X` | `container image ls --format json` + filter in Go |
| Container exists | `docker container inspect name` | `container inspect name` (check exit code) |
| Container running | `docker container inspect -f {{.State.Running}} name` | `container inspect name --format json` (parse status field) |
| Image exists | `docker image inspect name` | `container image inspect name` (check exit code) |
| Export filesystem | `docker export -o file.tar name` | `container export -o file.tar name` |

## Migration Plan

### Phase 1: Create Interface + Docker Implementation

**Goal**: Establish the abstraction with zero behavior change.

1. Create `internal/runtime/runtime.go` with interface and types.
2. Create `internal/runtime/docker.go` implementing every method by wrapping existing `exec.Command("docker", ...)` patterns.
3. Create `internal/runtime/detect.go` returning Docker always (detection comes later).
4. Add `var rt runtime.Runtime` in `cmd/root.go` with `PersistentPreRunE`.
5. All existing tests pass unchanged.

### Phase 2: Migrate cmd/ Files to Runtime Interface

Migrate one file at a time, each as a self-contained commit:

1. **`cmd/run.go`** (largest): Replace `attachToContainer`, `startContainer`, `createAndStartContainerWithEnv`, `getContainerDiff`, `commitContainer`, `deleteContainer` with `rt.Attach()`, `rt.StartInteractive()`, `rt.RunInteractive()`, `rt.Diff()`, `rt.Commit()`, `rt.RemoveContainer()`.
2. **`cmd/build.go`**: Replace `runDockerBuild()` with `rt.BuildImage()`.
3. **`cmd/clean.go`**: Replace container/image listing and removal with runtime methods.
4. **`cmd/commit.go`**: Replace with `rt.Commit()` + `rt.RemoveContainer()`.
5. **`cmd/diff.go`**: Replace with `rt.Diff()`.
6. **`cmd/reset.go`**: Replace with `rt.RemoveContainer()`.
7. **`cmd/status.go`**: Replace `docker.ContainerExists/Running/ImageExists` and diff calls.

### Phase 3: Slim `internal/docker/`

Remove `ContainerExists`, `ContainerRunning`, `ImageExists`, `GetImageDigest` from `internal/docker/docker.go`. Keep only:
- `ContainerName(dir string) string` (pure function)
- `ImageName(dir string) string` (pure function)

Move `BuildRunArgs` into `runtime/docker.go` as a private helper.

### Phase 4: Add Apple Containers Implementation

**Pre-implementation validation** (see "Needs Physical Testing" section above):
- Run all 7 physical tests on macOS 15 and macOS 26.
- **Blockers**: If DNS during build fails, need Docker-build-then-transfer fallback. If UID/GID mismatch makes workspace unusable, need workaround before proceeding.

Implementation:
1. Create `internal/runtime/apple.go` implementing the interface.
2. `Capabilities()` returns `{SupportsDiff: false, SupportsCommit: false, SupportsExport: true}`.
3. `Attach()` uses `container exec -it name /bin/sh` (no `attach` command exists).
4. `RunInteractive()` uses `--name` for hostname (no `--hostname` flag).
5. `ListContainers()`/`ListImages()` use `--format json` + Go-side filtering (no `--filter` flag).
6. `BuildImage()` must ensure builder is running (`container builder start` if needed).
7. Update `detect.go` to check for `container` CLI with version verification.
8. Add `--runtime` persistent flag to root command.
9. Add capability checks in `cmd/run.go` (post-exit), `cmd/diff.go`, `cmd/commit.go`, `cmd/status.go`.

### Phase 5: Improve Apple Containers Support

- Investigate `container export` + rebuild as a `commit` workaround.
- Address UID/GID mismatch for bind mounts (may need `--user` flag or entrypoint `chown`).
- Test interactive terminal behavior (TTY allocation, signal forwarding).
- Benchmark bind mount throughput vs Docker for developer experience.

## Verification Plan

### Unit Tests
- `runtime/docker_test.go`: Test arg construction, exit code handling.
- `runtime/apple_test.go`: Test arg construction for Apple Containers CLI.
- `runtime/detect_test.go`: Test detection logic with mocked `LookPath`.

### Integration Tests
- Run `glovebox build --base` with each runtime.
- Run `glovebox run` and verify interactive session works.
- Verify fallback message appears when Apple Containers is not installed.
- Verify `--runtime docker` override works.
- Verify graceful degradation: `glovebox diff` with Apple Containers shows appropriate message.

### Pre-Phase 4 Physical Validation (macOS 15 + macOS 26)

Run these tests manually before implementing the Apple Containers runtime:

```bash
# 1. DNS during build
cat <<'EOF' > /tmp/test-dns.Dockerfile
FROM ubuntu:24.04
RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*
RUN curl -s https://httpbin.org/get | head -5
EOF
container build -t test-dns -f /tmp/test-dns.Dockerfile /tmp

# 2. Volume mount + UID/GID as non-root user
container run -it -v $(pwd):/workspace -w /workspace ubuntu:24.04 bash -c '
  useradd -m dev && su - dev -c "ls -la /workspace && touch /workspace/.test-write && rm /workspace/.test-write"
'

# 3. Container persistence + reattach
container run -it --name test-persist ubuntu:24.04 bash -c 'echo "first run"'
container start -a -i test-persist

# 4. Env var passthrough
container run --rm -e FOO=bar -e BAZ=qux alpine env | grep -E 'FOO|BAZ'

# 5. Hostname via --name
container run --rm --name glovebox alpine hostname

# 6. Builder auto-start
container builder stop 2>/dev/null
container build -t test-builder -f /tmp/test-dns.Dockerfile /tmp
# Does it fail or auto-start the builder?

# 7. Signal forwarding
container run --rm -it --name test-signal alpine sh -c 'trap "echo caught SIGINT" INT; echo "send Ctrl+C"; sleep 30'
# Press Ctrl+C -- does the trap fire?

# Cleanup
container rm test-persist 2>/dev/null
container image rm test-dns test-builder 2>/dev/null
```

**Blocking results**: Tests 1 (DNS) and 2 (UID/GID) are blockers. If DNS fails, we need a build fallback strategy. If UID/GID makes the workspace unusable for the `dev` user, we need a workaround (e.g., entrypoint `chown`, `--user` flag mapping, or documenting the limitation).

### Manual End-to-End Validation (after Phase 4)
- Full glovebox workflow on each runtime: `init` -> `build --base` -> `build` -> `run` -> exit -> diff/status
- Verify workspace is writable by `dev` user inside Apple Container
- Verify `glovebox clean` works with Apple Containers

## Critical Files

**New files:**
- `internal/runtime/runtime.go`
- `internal/runtime/detect.go`
- `internal/runtime/docker.go`
- `internal/runtime/apple.go`

**Modified files:**
- `cmd/root.go` -- PersistentPreRunE + --runtime flag
- `cmd/run.go` -- Replace 6 functions with runtime calls + capability checks
- `cmd/build.go` -- Replace `runDockerBuild()`
- `cmd/clean.go` -- Replace listing/removal functions
- `cmd/commit.go` -- Replace with runtime calls
- `cmd/diff.go` -- Replace with runtime call + capability check
- `cmd/reset.go` -- Replace with runtime call
- `cmd/status.go` -- Replace inspection calls + capability check
- `internal/docker/docker.go` -- Slim to pure functions only
