# Gitleaks Language Server

A Language Server Protocol (LSP) implementation for [Gitleaks](https://github.com/gitleaks/gitleaks) that provides real-time secret detection in your code editor.

## Features

- Real-time secret scanning with gitleaks detection engine
- Configurable via `.gitleaks.toml`
- LSP diagnostics for detected secrets
- **Hover documentation** - Rich markdown tooltips with recommendations
- **Code actions** - Quick fixes to ignore false positives (40+ languages)
- **Content-hash caching** - Skip redundant scans for unchanged content
- **`.gitleaksignore` support** - Suppress false positives with fingerprints
- **Workspace scanning** - Scan entire project on demand with progress reporting
- **Configurable severity** - Set diagnostic level (error/warning/info/hint)

## Configuration

The language server automatically detects a `.gitleaks.toml` file in your workspace root. If found, it uses the rules defined in that file. If not found, it falls back to the default Gitleaks configuration.

### Ignore File

Create a `.gitleaksignore` file in your workspace root to suppress false positives:

```
# Format: file:rule-id:start-line
config/example.go:aws-access-token:10
tests/fixtures/secrets.txt:generic-api-key:5
```

The ignore file is automatically watched for changes and reloaded.

### Custom Rules

To use custom rules, create a `.gitleaks.toml` file in your project root:

```toml
[extend]
useDefault = true

[[rules]]
id = "custom-api-key"
description = "Custom API Key Pattern"
regex = '''(?i)custom[_-]?api[_-]?key[:\s=]+['"]?([a-zA-Z0-9]{32})['"]?'''
keywords = ["custom_api_key"]
```

The server watches this file for changes and automatically reloads the configuration.

### LSP Settings

Configure via your editor's LSP settings:

```json
{
  "gitleaks": {
    "diagnosticSeverity": "warning"
  }
}
```

| Setting | Values | Default | Description |
|---------|--------|---------|-------------|
| `diagnosticSeverity` | `error`, `warning`, `information`, `hint` | `warning` | Severity level for detected secrets |

## Requirements

- Go 1.24+
- Any LSP-compatible editor (tested with Neovim)

### Supported Platforms

- Linux (x86_64, arm64)
- macOS (x86_64, arm64)
- Windows (x86_64)

## Installation

```bash
# Clone the repository
git clone https://github.com/arch-stack/gitleaks-ls.git
cd gitleaks-ls

# Build the binary
go build -o gitleaks-ls

# Optional: Install to PATH
sudo mv gitleaks-ls /usr/local/bin/
# Or for local user
mv gitleaks-ls ~/.local/bin/
```

## Quick Start

1. Build the server:
   ```bash
   go build -o gitleaks-ls
   ```

2. Test with the included Neovim config:
   ```bash
   ./test.sh
   # Or manually:
   nvim -u test-lsp.lua examples/test_file.go
   ```

3. Add to your Neovim config (see [Editor Setup](#editor-setup) below)

## Developer Commands

### Building

```bash
# Build binary
go build -o gitleaks-ls

# Build with race detector (for development)
go build -race -o gitleaks-ls
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run specific test
go test -v -run TestScanner ./...

# Run memory leak tests
go test -v -run TestMemoryLeak ./...
```

### Benchmarks

```bash
# Run all benchmarks
go test -bench=. ./...

# Run benchmarks with memory stats
go test -bench=. -benchmem ./...

# Run specific benchmark
go test -bench=BenchmarkScanner ./...
```

### Code Quality

```bash
# Format code
go fmt ./...

# Vet code
go vet ./...

# Run golangci-lint (if installed)
golangci-lint run
```

### Manual Testing with Neovim

```bash
# Quick test with included script
./test.sh

# Or manually:
nvim -u test-lsp.lua examples/test_file.go
```

**Keybindings in test config:**
- `K` - Hover documentation
- `<leader>ca` - Code actions
- `]d` / `[d` - Navigate diagnostics
- `<leader>q` - Show all diagnostics in quickfix

**Debug commands:**
- `:LspInfo` - Check LSP status
- `:LspClients` - List active clients
- `:LspLog` - Open LSP log file
- `:DiagShow` - Show diagnostics in current buffer

### Viewing Logs

```bash
# Neovim LSP logs
tail -f ~/.local/state/nvim/lsp.log

# Or from within Neovim
:LspLog
```

## Editor Setup

### Neovim (Native LSP)

Add to your `init.lua`:

```lua
vim.api.nvim_create_autocmd('FileType', {
  pattern = '*',
  callback = function()
    vim.lsp.start({
      name = 'gitleaks-ls',
      cmd = { 'gitleaks-ls' },  -- or full path to binary
      root_dir = vim.fs.dirname(vim.fs.find({'.git'}, { upward = true })[1]),
    })
  end,
})
```

Or with [nvim-lspconfig](https://github.com/neovim/nvim-lspconfig):

```lua
-- Custom server setup (gitleaks-ls not in lspconfig defaults)
local lspconfig = require('lspconfig')
local configs = require('lspconfig.configs')

configs.gitleaks_ls = {
  default_config = {
    cmd = { 'gitleaks-ls' },
    filetypes = { '*' },
    root_dir = lspconfig.util.root_pattern('.git'),
  },
}

lspconfig.gitleaks_ls.setup({})
```

## Architecture

```
gitleaks-ls/
├── main.go           # Entry point, LSP server setup
├── handlers.go       # LSP message handlers (didOpen, didChange, etc.)
├── scanner.go        # Gitleaks library wrapper
├── diagnostics.go    # Finding → LSP Diagnostic conversion
├── config.go         # Configuration loading and watching
├── cache.go          # Content-hash result caching
├── hover.go          # Hover documentation provider
├── actions.go        # Code actions (40+ languages)
├── workspace.go      # Workspace scanning with progress reporting
├── settings.go       # LSP settings (diagnostic severity)
├── uri.go            # Cross-platform URI/path conversion
├── .github/workflows # CI/CD (test, lint, build, release)
└── *_test.go         # Tests for each component
```

## Performance

| Operation | Time | Target |
|-----------|------|--------|
| Scan small file | ~127µs | <10ms |
| Scan medium file (~1K lines) | ~2.7ms | <50ms |
| Scan large file (~500KB) | ~198ms | <200ms |
| Cache hit | ~108ns | <1µs |
| End-to-end (cache hit) | ~558ns | <1ms |
| End-to-end (cache miss) | ~67µs | <100ms |

## Documentation

- [AGENTS.md](./AGENTS.md) - AI agent guidelines and technical reference

## Contributing

```bash
# 1. Make changes

# 2. Format and lint
go fmt ./...
golangci-lint run

# 3. Run tests (must pass with 70%+ coverage)
go test -cover ./...

# 4. Run benchmarks to check performance
go test -bench=. ./...
```

## CI/CD

The project uses GitHub Actions for:
- **Test**: Matrix testing on Linux/macOS/Windows with Go 1.24-1.25
- **Lint**: golangci-lint with custom configuration
- **Benchmark**: Performance regression detection
- **Build**: Cross-platform binary artifacts
- **Release**: Automated releases on tag push

## License

MIT
