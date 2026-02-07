---
description: Upgrade the gitleaks package to a newer version
globs: go.mod, go.sum, scanner.go, config.go
alwaysApply: false
---

# Skill: Upgrade Gitleaks Package

Use this procedure when upgrading the `github.com/zricethezav/gitleaks/v8` dependency.

## Step 1: Check Available Versions

```bash
go list -m -versions github.com/zricethezav/gitleaks/v8
```

## Step 2: Upgrade

```bash
# Latest
go get github.com/zricethezav/gitleaks/v8@latest
go mod tidy

# Or specific version
go get github.com/zricethezav/gitleaks/v8@v8.X.Y
go mod tidy
```

## Step 3: Check for Breaking Changes

Review https://github.com/gitleaks/gitleaks/releases for the new version.

**Watch for:**
- `detect.Fragment` removal (deprecated in v8, may be `sources.Fragment` in v9)
- `config.ViperConfig` or `Translate()` API changes
- `report.Finding` field changes

## Step 4: Files That Import Gitleaks

| File | Imports |
|------|---------|
| `config.go` | `config` |
| `scanner.go` | `config`, `detect`, `report` |
| `scanner_test.go` | `config`, `report` |

## Step 5: Run Tests

```bash
go test ./...
```

**If tests fail:**

1. **Secret detection tests** - Rule patterns may have changed. Validate:
   ```bash
   echo 'key = "AKIATESTKEYEXAMPLE7A"' | gitleaks detect --no-git --source=/dev/stdin
   ```

2. **Column/line tests** - If `diagnostics_test.go` fails, gitleaks may have changed column indexing. Check `adjustColumn()` logic.

3. **Finding struct changes** - Check if `report.Finding` fields were renamed/removed.

## Step 6: Run Linter

```bash
golangci-lint run
```

Add suppressions to `.golangci.yml` if new deprecation warnings appear for APIs that still work.

## Step 7: Run Benchmarks

```bash
go test -bench=. -benchmem ./...
```

Check for performance regressions.

## Step 8: Manual Test

```bash
./test.sh
```

Verify in Neovim:
- Diagnostics appear correctly
- Hover works
- Code actions work

## Step 9: Commit

```bash
git add go.mod go.sum
git commit -m "chore: upgrade gitleaks to vX.Y.Z"
```

## Rollback

```bash
go get github.com/zricethezav/gitleaks/v8@vPREVIOUS
go mod tidy
```
