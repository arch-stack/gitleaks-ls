# AI Assistant Context - Gitleaks Language Server

**Purpose:** Prevent repeated mistakes and reduce AI exploration paths. Use this before making changes.

---

## Critical: Library Gotchas

### Gitleaks v8 Import Path
```go
// CORRECT - use zricethezav, not gitleaks
import "github.com/zricethezav/gitleaks/v8/config"
import "github.com/zricethezav/gitleaks/v8/detect"
```
The module redirects from `gitleaks/gitleaks` but declares itself as `zricethezav/gitleaks`.

### Gitleaks Config Loading (Viper Required)
```go
v := viper.New()
v.SetConfigType("toml")
v.SetConfigFile(path)        // or v.ReadConfig(strings.NewReader(config.DefaultConfig))
v.ReadInConfig()
var vc config.ViperConfig
v.Unmarshal(&vc)
cfg, _ := vc.Translate()     // This is the only way to get config.Config
```
**There is no `config.NewConfig()` function.**

### Gitleaks Scanning
```go
fragment := detect.Fragment{Raw: content, FilePath: filename}
findings := detector.Detect(fragment)
```
Note: `detect.Fragment` is deprecated (v8), will be `sources.Fragment` in v9. Suppress with golangci-lint.

### glsp LSP Types - Pointers Required
```go
severity := protocol.DiagnosticSeverityWarning
diag.Severity = &severity  // Must be pointer

source := "gitleaks"
diag.Source = &source      // Must be pointer

// WorkDoneProgressBegin.Cancellable is *bool - omit if false
```

---

## Valid Test Secrets

**AWS Access Key:** `AKIATESTKEYEXAMPLE7A`
- Must be: `AKIA` + 16 chars from `[A-Z2-7]` (Base32)
- Invalid: `AKIAIOSFODNN7EXAMPLE` (contains O, I)

**GitHub PAT:** `ghp_1234567890abcdefghijklmnopqrstuvwx`
- Must be: `ghp_` + exactly 36 alphanumeric chars

**Validate with CLI:**
```bash
echo 'key = "AKIATESTKEYEXAMPLE7A"' | gitleaks detect --no-git --source=/dev/stdin
```

---

## LSP Indexing

| Source | Lines | Columns |
|--------|-------|---------|
| LSP | 0-indexed | 0-indexed |
| Gitleaks | 1-indexed | varies by line |

Conversion in `diagnostics.go:adjustColumn()` handles the off-by-one differences.

---

## Cross-Platform URIs

```go
// Windows: file:///C:/path → C:\path
// Unix: file:///path → /path
// See uri.go for implementation
```

---

## Code Quality Checklist

Before committing:
```bash
go fmt ./...
golangci-lint run    # Config in .golangci.yml
go test ./...        # Must maintain 70%+ coverage
```

Common lint fixes:
- Type assertions: `x, ok := val.(Type)` not `x := val.(Type)`
- Unused params: use `_` placeholder
- String conversions: don't convert `string` to `string`

---

## Architecture Quick Reference

| File | Purpose |
|------|---------|
| `scanner.go` | Gitleaks wrapper, returns `[]Finding` |
| `diagnostics.go` | `Finding` → LSP `Diagnostic` |
| `handlers.go` | LSP lifecycle (didOpen, didSave, etc.) |
| `workspace.go` | Parallel scanning + progress reporting |
| `settings.go` | `diagnosticSeverity` config |
| `uri.go` | Cross-platform `file://` handling |
| `cache.go` | SHA-256 content hash → findings |

---

## Performance Targets (All Met)

| Operation | Target | Actual |
|-----------|--------|--------|
| Small file | <10ms | ~127µs |
| Large file (500KB) | <200ms | ~198ms |
| Cache hit | <1µs | ~108ns |

---

## Don't Repeat These Mistakes

1. **Invalid test secrets** - Always validate with `gitleaks detect` CLI first
2. **Wrong import path** - Use `zricethezav`, not `gitleaks`
3. **Missing pointers** - LSP types need `&value` for optional fields
4. **Unchecked type assertions** - Use `val, ok := x.(Type)` pattern
5. **Platform-specific paths** - Use `uri.go` functions, not string manipulation

---

## Supported Platforms

- Linux, macOS, Windows
- Go versions: 1.21, 1.22, 1.23

**Last Updated:** 2025-12-05
