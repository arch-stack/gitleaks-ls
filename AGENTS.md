# AGENTS.md - AI Agent Guidelines

This document provides guidance for AI agents working on the gitleaks-ls codebase.

## Project Overview

gitleaks-ls is a Language Server Protocol (LSP) implementation for [Gitleaks](https://github.com/gitleaks/gitleaks) that provides real-time secret detection in code editors. Written in Go, it uses stdio for LSP communication.

**Key Features:**
- Real-time scanning on file open/change/save
- Content-hash caching for performance
- `.gitleaks.toml` and `.gitleaksignore` support with file watching
- Hover documentation with remediation guidance
- Code actions for adding `gitleaks:allow` comments
- Workspace-wide scanning command with progress reporting
- Configurable diagnostic severity

## Quick Start

```bash
# Build
go build -o gitleaks-ls

# Test
go test ./...

# Lint
golangci-lint run

# Test manually with Neovim
./test.sh
```

## Architecture

Flat package structure - all source files in root (single `main` package):

| File | Purpose |
|------|---------|
| `main.go` | Entry point, LSP server setup, `initialize` handler |
| `handlers.go` | Document handlers, `Server` struct, `DocumentStore`, `scanAndPublish()` |
| `scanner.go` | Gitleaks wrapper, `Finding` type, `.gitleaksignore` loading |
| `diagnostics.go` | `Finding` → LSP `Diagnostic` conversion, column adjustment |
| `config.go` | `.gitleaks.toml` loading via Viper, file watching |
| `cache.go` | SHA256 content-hash → findings cache |
| `hover.go` | Markdown hover documentation for findings |
| `actions.go` | Code actions, comment syntax for 40+ languages |
| `workspace.go` | `gitleaks.scanWorkspace` command, parallel file scanning |
| `settings.go` | LSP settings (`diagnosticSeverity`) |
| `uri.go` | Cross-platform `file://` URI ↔ filesystem path |

**Global State:** `globalServer *Server` holds scanner, documents, config, and cache.

## Critical Gotchas

### Gitleaks Import Path

```go
// CORRECT - use zricethezav, not gitleaks
import "github.com/zricethezav/gitleaks/v8/config"
import "github.com/zricethezav/gitleaks/v8/detect"
```

The module redirects from `gitleaks/gitleaks` but declares itself as `zricethezav/gitleaks`.

### Gitleaks Config Loading (No Constructor)

```go
// There is NO config.NewConfig() function
v := viper.New()
v.SetConfigType("toml")
v.SetConfigFile(path)
v.ReadInConfig()
var vc config.ViperConfig
v.Unmarshal(&vc)
cfg, _ := vc.Translate()  // This creates config.Config
```

### LSP Types Require Pointers

```go
severity := protocol.DiagnosticSeverityWarning
diag.Severity = &severity  // Must be pointer

source := "gitleaks"
diag.Source = &source      // Must be pointer
```

### LSP Indexing & Column Quirks

| Source | Lines | Columns |
|--------|-------|---------|
| LSP | 0-indexed | 0-indexed |
| Gitleaks | 1-indexed | **inconsistent** |

**Gitleaks column numbering is quirky:**
- Line 0: `StartColumn` is 1-indexed, `EndColumn` is 0-indexed (exclusive)
- Line >0: `StartColumn` is 2-indexed, `EndColumn` is 1-indexed (exclusive)

The `adjustColumn()` function in `diagnostics.go` handles this. Don't try to simplify it.

### Cross-Platform URIs

Windows: `file:///C:/path` → `C:\path`
Unix: `file:///path` → `/path`

Use `uri.go` functions (`uriToPath`, `pathToURI`), not string manipulation.

### File Size Limit

Files >1MB are silently skipped (returns empty findings, no error). See `ScanContent()` in `scanner.go`.

## Valid Test Secrets

Use these for tests - invalid secrets won't be detected:

**AWS Access Key:** `AKIATESTKEYEXAMPLE7A`
- Must be: `AKIA` + 16 chars from `[A-Z2-7]` (Base32 alphabet)
- **Invalid:** `AKIAIOSFODNN7EXAMPLE` (contains O, I - not in Base32)

**GitHub PAT:** `ghp_1234567890abcdefghijklmnopqrstuvwx`
- Must be: `ghp_` + exactly 36 alphanumeric chars

Validate with CLI:
```bash
echo 'key = "AKIATESTKEYEXAMPLE7A"' | gitleaks detect --no-git --source=/dev/stdin
```

## Testing Patterns

**Create a test scanner:**
```go
func newTestScanner(t testing.TB) *Scanner {
    v := viper.New()
    v.SetConfigType("toml")
    require.NoError(t, v.ReadConfig(strings.NewReader(config.DefaultConfig)))
    
    var vc config.ViperConfig
    require.NoError(t, v.Unmarshal(&vc))
    
    cfg, err := vc.Translate()
    require.NoError(t, err)
    
    return NewScanner(cfg)
}
```

**Mock LSP context for integration tests:**
```go
var notifications []protocol.PublishDiagnosticsParams
ctx := &glsp.Context{
    Notify: func(method string, params any) {
        if method == "textDocument/publishDiagnostics" {
            if p, ok := params.(protocol.PublishDiagnosticsParams); ok {
                notifications = append(notifications, p)
            }
        }
    },
}
```

## Code Quality Requirements

- **Linting:** `golangci-lint run` (config in `.golangci.yml`)
- **Tests:** `go test ./...` (maintain 70%+ coverage)
- **Formatting:** `go fmt ./...`

## Performance Targets

| Operation | Target |
|-----------|--------|
| Scan small file (<100 lines) | <10ms |
| Scan medium file (~1K lines) | <50ms |
| Scan large file (~500KB) | <200ms |
| Cache hit | <1µs |
| Server startup | <500ms |

Run benchmarks: `go test -bench=. -benchmem ./...`

## Common Mistakes to Avoid

1. **Invalid test secrets** - Always validate with `gitleaks detect` CLI first
2. **Wrong import path** - Use `zricethezav`, not `gitleaks`
3. **Missing pointers** - LSP types need `&value` for optional fields
4. **Unchecked type assertions** - Use `val, ok := x.(Type)` pattern
5. **Platform-specific paths** - Use `uri.go` functions
6. **Suppress deprecated warnings** - `detect.Fragment` is deprecated (v8), handled in `.golangci.yml`
7. **Simplifying `adjustColumn()`** - The gitleaks column quirks require that exact logic
8. **Using `config.NewConfig()`** - This function doesn't exist; use Viper pattern
9. **Forgetting cache invalidation** - Config/ignore file changes must clear the cache

## Workspace Scanning

The `gitleaks.scanWorkspace` command:
- Scans with 10 concurrent goroutines (`maxConcurrent = 10`)
- Respects `.gitignore` patterns
- Skips: hidden files/dirs, `node_modules`, `vendor`, `__pycache__`, `target`, `build`, `dist`
- Skips binary files (by extension and magic bytes via `filetype` library)
- Reports progress via LSP `$/progress` notifications

## Key Dependencies

- **[glsp](https://github.com/tliron/glsp)** - LSP server framework (protocol_3_16)
- **[gitleaks/v8](https://github.com/zricethezav/gitleaks)** - Secret detection engine
- **[fsnotify](https://github.com/fsnotify/fsnotify)** - File watching for config/ignore reload
- **[viper](https://github.com/spf13/viper)** - Config loading (required by gitleaks)
- **[go-gitignore](https://github.com/sabhiram/go-gitignore)** - .gitignore pattern matching
- **[filetype](https://github.com/h2non/filetype)** - Binary file detection via magic bytes
- **[testify](https://github.com/stretchr/testify)** - Testing (assert, require)

## CI/CD

GitHub Actions workflows in `.github/workflows/`:
- `ci.yml` - Test (Linux/macOS/Windows, Go 1.25), lint, benchmark, build
- `release.yml` - Cross-platform binary releases on tag push

**Coverage requirement:** 70% minimum (enforced in CI)

## Documentation

- [README.md](./README.md) - Usage and setup

## LSP Capabilities

The server advertises these capabilities in `initialize`:
- `textDocumentSync`: Full sync with open/close/save
- `hoverProvider`: true
- `codeActionProvider`: true
- `executeCommandProvider`: `["gitleaks.scanWorkspace"]`

## Design Principles

1. **Simplicity**: Flat structure, no unnecessary abstractions
2. **Testability**: Each component testable in isolation
3. **Performance**: Cache aggressively, fail fast
4. **Reliability**: Never crash, log errors, degrade gracefully

## Error Handling

- **Initialization errors** (fatal): Invalid config, can't init detector → log and exit
- **Scan errors** (recoverable): File too large, detector error → log, return empty findings
- **LSP errors** (recoverable): Invalid params, unknown URI → log warning, return error

Use `slog` for structured logging to stderr.

## Memory & CPU Targets

| Resource | Target | Max |
|----------|--------|-----|
| Server baseline | <10MB | 20MB |
| Per document | <1MB | 5MB |
| Cache (100 files) | <20MB | 50MB |
| Idle CPU | <1% | 2% |
| Active scanning CPU | <20% | 50% |

## Non-Goals

These are explicitly out of scope:
- Secret management or rotation
- Git history scanning (use gitleaks CLI)
- Custom rule creation UI
- Editor extensions/plugins (raw LSP only)
- Multi-workspace folder support (uses single rootUri)
