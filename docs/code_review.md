# Glovebox Code Review

**Review Date:** December 2024
**Reviewer Perspective:** Senior Staff Go Developer
**Focus:** Clarity, Simplicity, Security, DRY, Maintainability by AI Agents

---

## Executive Summary

Glovebox is a well-structured Go CLI for creating sandboxed Docker development environments. The codebase demonstrates solid Go fundamentals and clean architecture. This review identifies opportunities for improvement to enhance maintainability, reduce duplication, and improve reliability—particularly important for AI-assisted development.

**Overall Assessment:** Good foundation with several areas for refinement.

---

## Critical Issues

### 1. No Test Coverage ✅ DONE

**Location:** Entire codebase
**Severity:** High
**Status:** Resolved - Added 139 test cases across 5 packages

The codebase has zero test files. This is the most significant issue for AI-maintained code.

**Impact:**
- AI agents cannot verify their changes work correctly
- Regressions are likely during refactoring
- No documentation of expected behavior through tests

**Recommendation:**
- Add unit tests for `internal/` packages (pure functions, easy to test)
- Add integration tests for command execution
- Priority order:
  1. `internal/digest/` - simple, pure functions
  2. `internal/mod/` - dependency resolution logic
  3. `internal/profile/` - serialization/deserialization
  4. `internal/generator/` - Dockerfile generation

---

### 2. Duplicated Container/Image Helper Functions ✅ DONE

**Locations:**
- `cmd/run.go:114-127` - `checkContainerExists`, `checkContainerRunning`
- `cmd/status.go:152-164` - `containerExists`, `containerRunning`
- `cmd/clean.go:254-257` - `containerExistsForClean`

**Status:** Resolved - Extracted to `internal/docker/` package

**Issue:** Three separate implementations of essentially the same Docker inspection logic.

```go
// run.go
func checkContainerExists(name string) bool {
    cmd := exec.Command("docker", "container", "inspect", name)
    return cmd.Run() == nil
}

// status.go
func containerExists(name string) bool {
    cmd := exec.Command("docker", "container", "inspect", name)
    return cmd.Run() == nil
}

// clean.go
func containerExistsForClean(name string) bool {
    cmd := exec.Command("docker", "container", "inspect", name)
    return cmd.Run() == nil
}
```

**Recommendation:** Create `internal/docker/` package with shared helpers:
```go
package docker

func ContainerExists(name string) bool
func ContainerRunning(name string) bool
func ImageExists(name string) bool
func GetImageDigest(name string) (string, error)
```

---

### 3. Duplicated Container Name Generation ✅ DONE

**Locations:**
- `cmd/run.go:74-77`
- `cmd/clean.go:88-92`
- `cmd/status.go:126-129`

**Status:** Resolved - Added `ContainerName()` and `ImageName()` to `internal/docker/` package

**Issue:** Container name calculation logic duplicated in three places.

```go
// Appears in multiple files with slight variations
hash := sha256.Sum256([]byte(absPath))
shortHash := fmt.Sprintf("%x", hash)[:7]
dirName := filepath.Base(absPath)
containerName := fmt.Sprintf("glovebox-%s-%s", dirName, shortHash)
```

**Recommendation:** Add to `internal/profile/profile.go`:
```go
func ContainerName(dir string) string {
    absPath, _ := filepath.Abs(dir)
    hash := sha256.Sum256([]byte(absPath))
    shortHash := fmt.Sprintf("%x", hash)[:7]
    dirName := filepath.Base(absPath)
    return fmt.Sprintf("glovebox-%s-%s", dirName, shortHash)
}
```

---

## Moderate Issues

### 4. Inconsistent Error Handling Patterns ✅ DONE

**Location:** Various command files
**Status:** Resolved - All errors now wrapped with context using `%w`

**Issue:** Mix of error handling styles:

```go
// Sometimes wraps errors
return fmt.Errorf("resolving path: %w", err)

// Sometimes returns raw errors
return err

// Sometimes silently ignores errors
if m, err := loadFromFile(fullPath); err == nil {
    return m, nil
}
```

**Recommendation:** Establish consistent pattern:
- Always wrap errors with context using `%w`
- Use sentinel errors for expected conditions
- Consider `errors.Is()` / `errors.As()` for error checking

---

### 5. Deprecated Function Still Present ✅ DONE

**Location:** `internal/generator/generator.go:277-279`
**Status:** Resolved - Removed deprecated `Generate` function (unused)

```go
// Generate creates a Dockerfile from a list of mod IDs (legacy, for base images)
// Deprecated: Use GenerateBase or GenerateProject instead
func Generate(modIDs []string) (string, error) {
    return GenerateBase(modIDs)
}
```

**Issue:** Deprecated code increases cognitive load and confusion.

**Recommendation:** Search for usages, remove if unused, or add `// Deprecated:` doc comment that Go tools recognize.

---

### 6. Magic Strings for Docker Commands

**Location:** Throughout `cmd/` package

**Issue:** Docker subcommands and arguments scattered as string literals:

```go
exec.Command("docker", "container", "inspect", name)
exec.Command("docker", "image", "inspect", "--format", "{{.Id}}", name)
exec.Command("docker", "ps", "--filter", "ancestor=glovebox", "--format", "{{.Names}}\t{{.Image}}")
```

**Recommendation:** Create constants or a thin Docker wrapper:
```go
const (
    dockerBin = "docker"
)

func dockerInspectContainer(name string) *exec.Cmd {
    return exec.Command(dockerBin, "container", "inspect", name)
}
```

---

### 7. Hardcoded Environment Variable List ✅ DONE

**Location:** `cmd/run.go:159-169`
**Status:** Resolved - Added `passthrough_env` field to Profile struct

```go
envVars := []string{
    "ANTHROPIC_API_KEY",
    "OPENAI_API_KEY",
    "GOOGLE_API_KEY",
    "GEMINI_API_KEY",
}
```

**Issue:** Adding new API keys requires code changes.

**Recommendation:** Consider making this configurable via:
- Profile YAML `pass_through_env` field
- Or pattern matching (e.g., `*_API_KEY`)

---

### 8. Unused Function ✅ DONE

**Location:** `cmd/status.go:182-190`
**Status:** Resolved - Removed during Docker helper extraction

```go
func volumeExists(name string) bool {
    cmd := exec.Command("docker", "volume", "inspect", name)
    output, err := cmd.CombinedOutput()
    // ...
}
```

**Issue:** Function defined but never called.

**Recommendation:** Remove dead code.

---

## Minor Issues

### 9. Deprecated `strings.Title` Usage ✅ DONE

**Location:** `cmd/init.go:133`
**Status:** Resolved - Replaced with `golang.org/x/text/cases`

```go
bold.Printf("%s:\n", strings.Title(category))
```

**Issue:** `strings.Title` is deprecated in favor of `golang.org/x/text/cases`.

**Recommendation:** Replace with:
```go
import "golang.org/x/text/cases"
import "golang.org/x/text/language"

caser := cases.Title(language.English)
bold.Printf("%s:\n", caser.String(category))
```

---

### 10. Inconsistent Color Variable Patterns ✅ DONE

**Locations:** Various command files
**Status:** Resolved - Created `cmd/colors.go` with package-level color definitions

**Issue:** Color objects created at different scopes:

```go
// Sometimes at function start
func runBuild(cmd *cobra.Command, args []string) error {
    yellow := color.New(color.FgYellow)
    green := color.New(color.FgGreen)
    // ...
}

// Sometimes passed as parameters
func handlePostExit(containerName, imageName string, green, yellow, dim *color.Color) error
```

**Recommendation:** Create package-level color constants:
```go
var (
    colorGreen  = color.New(color.FgGreen)
    colorYellow = color.New(color.FgYellow)
    colorDim    = color.New(color.Faint)
    colorBold   = color.New(color.Bold)
    colorRed    = color.New(color.FgRed)
)
```

---

### 11. go.mod Uses `// indirect` for Direct Dependencies ✅ DONE

**Location:** `go.mod`
**Status:** Resolved - `go mod tidy` properly separates direct and indirect dependencies

```go
require (
    github.com/fatih/color v1.18.0 // indirect
    github.com/spf13/cobra v1.10.2 // indirect
    gopkg.in/yaml.v3 v3.0.1 // indirect
)
```

**Issue:** `cobra`, `color`, and `yaml.v3` are direct imports but marked indirect.

**Recommendation:** Run `go mod tidy` to fix dependency declarations.

---

### 12. Potential Path Traversal in Mod Loading ✅ DONE

**Location:** `internal/mod/mod.go:72-96`
**Status:** Resolved - Added `validateModID()` function that rejects `..` sequences and absolute paths

```go
func Load(id string) (*Mod, error) {
    filename := id + ".yaml"
    // ...
    fullPath := filepath.Join(searchPath, filename)
```

**Issue:** Mod IDs like `../../../etc/passwd` could potentially escape the intended directories.

**Recommendation:** Validate mod IDs:
```go
func Load(id string) (*Mod, error) {
    if strings.Contains(id, "..") {
        return nil, fmt.Errorf("invalid mod id: %s", id)
    }
    // ...
}
```

---

### 13. Dockerfile Path Extraction Fragile

**Location:** `cmd/build.go:232-235` and `cmd/build.go:324`

```go
dockerfileDir := dockerfilePath[:len(dockerfilePath)-len("/Dockerfile")]
if dockerfileDir == "" {
    dockerfileDir = "."
}
```

**Issue:** Manual string slicing instead of using `filepath.Dir()`.

**Recommendation:**
```go
dockerfileDir := filepath.Dir(dockerfilePath)
```

---

### 14. Generator Has Unused Helper ✅ DONE

**Location:** `internal/generator/generator.go:323-325`
**Status:** Resolved - Removed wrapper, now uses `sort.Strings()` directly

```go
func sortStrings(s []string) {
    sort.Strings(s)
}
```

**Issue:** Wrapper function adds no value over `sort.Strings()` directly.

**Recommendation:** Remove wrapper, use `sort.Strings()` directly.

---

## Architecture Observations

### Positive Patterns

1. **Clean package separation** - `cmd/`, `internal/mod/`, `internal/profile/`, etc.
2. **Embedded filesystem for mods** - Good use of `//go:embed`
3. **Layered image architecture** - Base + project images is a solid pattern
4. **Digest tracking** - Good approach for change detection

### Suggestions for AI Maintainability

1. **Add `internal/docker/` package** - Centralize all Docker interactions ✅ Done
2. **Add `internal/naming/` package** - Container/image name generation ✅ Done (merged into `internal/docker/`)
3. **Consider interfaces** - For Docker operations, enables mocking in tests
4. **Add structured logging** - Would help debug AI-generated changes

---

## Recommended Priority Order

For an AI agent to address these issues incrementally:

| Priority | Issue | Effort | Impact | Status |
|----------|-------|--------|--------|--------|
| 1 | Add test infrastructure | Medium | High | ✅ Done |
| 2 | Extract Docker helpers to `internal/docker/` | Low | High | ✅ Done |
| 3 | Consolidate container name generation | Low | Medium | ✅ Done |
| 4 | Fix `go.mod` indirect markers | Trivial | Low | ✅ Done |
| 5 | Remove unused code (`volumeExists`, `sortStrings`, `Generate`) | Trivial | Low | ✅ Done |
| 6 | Add path traversal validation | Low | Medium | ✅ Done |
| 7 | Fix deprecated `strings.Title` | Low | Low | ✅ Done |
| 8 | Standardize error handling | Medium | Medium | ✅ Done |
| 9 | Make env passthrough configurable | Medium | Low | ✅ Done |
| 10 | Centralize color definitions | Low | Low | ✅ Done |

---

## Conclusion

Glovebox is a solid CLI with good architecture. The primary focus should be on:

1. **Adding tests** - Critical for reliable AI-assisted development
2. **Reducing duplication** - DRY improvements in Docker helpers
3. **Small cleanups** - Dead code, deprecated functions

The codebase is well-positioned for incremental improvements. Each change should be small, focused, and independently verifiable.
