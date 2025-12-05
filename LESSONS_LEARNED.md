# AI Assistant Context - Gitleaks Language Server

**Purpose:** Prevent repeated mistakes and reduce AI exploration paths.

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

## Common Mistakes to Avoid

1. **Invalid test secrets** - Always validate with `gitleaks detect` CLI first
2. **Wrong import path** - Use `zricethezav`, not `gitleaks`
3. **Missing pointers** - LSP types need `&value` for optional fields
4. **Unchecked type assertions** - Use `val, ok := x.(Type)` pattern
5. **Platform-specific paths** - Use `uri.go` functions, not string manipulation
