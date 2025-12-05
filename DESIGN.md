# Technical Design Document: Gitleaks Language Server

**Version**: 1.4
**Last Updated**: 2025-12-05
**References**: PRD.md

---

## 1. Overview

This document bridges the PRD (what/why) and implementation (code). It defines interfaces, types, message flows, and technical decisions for the gitleaks language server.

### 1.1 Architecture Summary

```
Neovim (LSP Client) <--stdio/JSON-RPC--> gitleaks-ls (LSP Server)
                                              |
                                              +--> Scanner (gitleaks wrapper)
                                              +--> Cache (findings cache)
                                              +--> Config (watch .gitleaks.toml)
```

### 1.2 Key Design Principles

1. **Simplicity**: Flat structure, no unnecessary abstractions
2. **Testability**: Each component testable in isolation
3. **Performance**: Cache aggressively, fail fast
4. **Reliability**: Never crash, log errors, degrade gracefully

---

## 2. Package Structure

```
gitleaks-ls/
├── main.go              # Entry point, server initialization
├── handlers.go          # LSP message handlers
├── scanner.go           # Gitleaks integration
├── diagnostics.go       # Finding → Diagnostic conversion
├── config.go            # Configuration management
├── cache.go             # Result caching (content hash)
├── hover.go             # Hover provider
├── actions.go           # Code actions (40+ languages)
├── workspace.go         # Workspace scanning
├── go.mod / go.sum      # Dependencies
├── *_test.go            # Tests alongside source
└── README.md            # Usage documentation
```

**Flat structure**: All source files in root to minimize complexity.

---

## 3. Core Types and Interfaces

### 3.1 Scanner Interface

```go
// scanner.go

import (
    "context"
    "github.com/gitleaks/gitleaks/v8/detect"
    "github.com/gitleaks/gitleaks/v8/report"
)

// Scanner wraps gitleaks detection engine
type Scanner struct {
    detector *detect.Detector
    config   *detect.Config
}

// NewScanner creates a scanner with default or custom config
func NewScanner(configPath string) (*Scanner, error)

// ScanContent scans the provided content and returns findings
func (s *Scanner) ScanContent(ctx context.Context, filename, content string) ([]Finding, error)

// Finding represents a detected secret
type Finding struct {
    RuleID      string  // e.g. "aws-access-key"
    Description string  // e.g. "AWS Access Key"
    Match       string  // The matched secret (may be redacted)
    Secret      string  // Extracted secret value (redacted)
    StartLine   int     // 1-indexed
    EndLine     int     // 1-indexed
    StartColumn int     // 1-indexed
    EndColumn   int     // 1-indexed
    Entropy     float64 // Shannon entropy score
    File        string  // File path/name
    Commit      string  // Empty for LSP usage
    Fingerprint string  // Unique identifier for this finding
}
```

**Implementation Notes**:
- Wrap `gitleaks/v8/detect.Detector`
- Convert `report.Finding` to our `Finding` type
- Use gitleaks default config if no `.gitleaks.toml` found
- For LSP, we scan strings not files, so use `detector.DetectString()`

### 3.2 Diagnostic Conversion

```go
// diagnostics.go

import (
    "github.com/tliron/glsp"
    protocol "github.com/tliron/glsp/protocol_3_16"
)

// FindingsToDiagnostics converts scanner findings to LSP diagnostics
func FindingsToDiagnostics(findings []Finding) []protocol.Diagnostic

// FindingToDiagnostic converts a single finding
func FindingToDiagnostic(f Finding) protocol.Diagnostic {
    return protocol.Diagnostic{
        Range: protocol.Range{
            Start: protocol.Position{
                Line:      uint32(f.StartLine - 1), // LSP is 0-indexed
                Character: uint32(f.StartColumn - 1),
            },
            End: protocol.Position{
                Line:      uint32(f.EndLine - 1),
                Character: uint32(f.EndColumn),
            },
        },
        Severity: protocol.DiagnosticSeverityWarning, // Configurable in Phase 2
        Source:   "gitleaks",
        Message:  formatMessage(f),
        Code:     f.RuleID,
    }
}

// formatMessage creates a human-readable diagnostic message
func formatMessage(f Finding) string {
    // Example: "Detected AWS Access Key (entropy: 4.2)"
}
```

**Key Decisions**:
- LSP uses 0-indexed lines, gitleaks uses 1-indexed
- Default severity: Warning (not Error, to avoid being too noisy)
- Include entropy in message for educational value
- Store RuleID in diagnostic.Code for reference

### 3.3 Document State Management

```go
// handlers.go

// DocumentStore tracks open documents and their diagnostics
type DocumentStore struct {
    mu          sync.RWMutex
    documents   map[string]*Document // URI -> Document
}

// Document represents an open file
type Document struct {
    URI         string
    Version     int32
    Content     string
    Diagnostics []protocol.Diagnostic
}

// UpdateDocument updates document content and version
func (ds *DocumentStore) UpdateDocument(uri string, version int32, content string)

// GetDocument retrieves a document by URI
func (ds *DocumentStore) GetDocument(uri string) (*Document, bool)
```

**Design Decision**: Simple in-memory map, no persistence needed.

### 3.4 Server State

```go
// main.go

// Server holds the language server state
type Server struct {
    glspServer *glsp.Server
    scanner    *Scanner
    documents  *DocumentStore
    config     *Config
    logger     *slog.Logger
}

// NewServer creates and initializes the language server
func NewServer() (*Server, error)

// Start begins serving LSP requests over stdio
func (s *Server) Start() error
```

---

## 4. LSP Message Flows

### 4.1 Initialize Sequence

```
Neovim                          gitleaks-ls
  |                                  |
  |-- initialize request ----------->|
  |                                  |-- Load config (.gitleaks.toml)
  |                                  |-- Initialize scanner
  |                                  |-- Setup document store
  |                                  |
  |<- initialize result -------------|
  |   (capabilities)                 |
  |                                  |
  |-- initialized notification ----->|
  |                                  |-- Ready to serve
```

**Server Capabilities** (Phase 1):
```go
capabilities := protocol.ServerCapabilities{
    TextDocumentSync: protocol.TextDocumentSyncOptions{
        OpenClose: true,
        Change:    protocol.TextDocumentSyncKindFull,
        Save:      &protocol.SaveOptions{IncludeText: true},
    },
}
```

### 4.2 Document Open Flow

```
Neovim                          gitleaks-ls
  |                                  |
  |-- textDocument/didOpen --------->|
  |   {uri, text, version}           |
  |                                  |-- Store document
  |                                  |-- Scan content
  |                                  |-- Convert findings to diagnostics
  |                                  |
  |<- textDocument/publishDiagnostics|
  |   {uri, diagnostics[]}           |
```

**Handler Implementation**:
```go
func (s *Server) handleDidOpen(ctx *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
    uri := params.TextDocument.URI
    content := params.TextDocument.Text
    version := params.TextDocument.Version
    
    // Store document
    s.documents.UpdateDocument(uri, version, content)
    
    // Scan for secrets
    findings, err := s.scanner.ScanContent(ctx, uri, content)
    if err != nil {
        s.logger.Error("scan failed", "uri", uri, "error", err)
        return nil // Don't propagate error to client
    }
    
    // Convert to diagnostics
    diagnostics := FindingsToDiagnostics(findings)
    
    // Publish
    s.glspServer.PublishDiagnostics(uri, diagnostics)
    
    return nil
}
```

### 4.3 Document Change Flow (Phase 1: Simple)

```
Neovim                          gitleaks-ls
  |                                  |
  |-- textDocument/didChange ------->|
  |   {uri, text, version}           |
  |                                  |-- Update document
  |                                  |-- (No scan - wait for save)
```

**Phase 1 Decision**: Only scan on save, not on change. Simpler, sufficient.

### 4.4 Document Save Flow

```
Neovim                          gitleaks-ls
  |                                  |
  |-- textDocument/didSave --------->|
  |   {uri, text}                    |
  |                                  |-- Scan content
  |                                  |-- Publish diagnostics
```

### 4.5 Hover Flow (Phase 2)

```
Neovim                          gitleaks-ls
  |                                  |
  |-- textDocument/hover ----------->|
  |   {uri, position}                |
  |                                  |-- Find diagnostic at position
  |                                  |-- Format finding details
  |                                  |
  |<- hover result ------------------|
  |   {markdown content}             |
```

**Hover Content Example**:
```markdown
### Detected Secret: AWS Access Key

**Rule ID**: `aws-access-key`
**Entropy**: 4.2
**Confidence**: High

This pattern matches AWS access keys. 

**Recommendation**: 
- Store credentials in environment variables
- Use AWS IAM roles instead of hardcoded keys
- Add to `.gitleaksignore` if this is a false positive

**To ignore**: Add `// gitleaks:allow` on the line above
```

---

## 5. Configuration Management

### 5.1 Config Type

```go
// config.go

import (
    "github.com/fsnotify/fsnotify"
    "github.com/gitleaks/gitleaks/v8/config"
)

// Config manages gitleaks configuration
type Config struct {
    path       string
    config     *config.ViperConfig
    watcher    *fsnotify.Watcher
    onReload   func() // Callback when config changes
}

// NewConfig loads config from path or uses defaults
func NewConfig(workspaceRoot string) (*Config, error) {
    // Look for .gitleaks.toml in workspace root
    // If not found, use gitleaks default config
}

// Watch starts watching the config file for changes
func (c *Config) Watch(ctx context.Context) error

// GetConfig returns the current gitleaks config
func (c *Config) GetConfig() *config.ViperConfig
```

**Implementation**:
1. On startup, search for `.gitleaks.toml` in workspace root
2. If found, load it; otherwise use `config.DefaultConfig()`
3. Start fsnotify watcher on the config file
4. On file change event, reload config and clear cache

### 5.2 Config File Search Order

1. `.gitleaks.toml` in workspace root
2. Fall back to gitleaks default config

**No cascading search**: Keep it simple, just check workspace root.

---

## 6. Caching Strategy (Phase 2)

### 6.1 Cache Type

```go
// cache.go

import (
    "crypto/sha256"
    "sync"
)

// Cache stores scan results keyed by content hash
type Cache struct {
    mu      sync.RWMutex
    entries map[[32]byte][]Finding // hash -> findings
}

// NewCache creates a new result cache
func NewCache() *Cache

// Get retrieves cached findings for content
func (c *Cache) Get(content string) ([]Finding, bool) {
    hash := sha256.Sum256([]byte(content))
    c.mu.RLock()
    defer c.mu.RUnlock()
    findings, ok := c.entries[hash]
    return findings, ok
}

// Put stores findings for content
func (c *Cache) Put(content string, findings []Finding) {
    hash := sha256.Sum256([]byte(content))
    c.mu.Lock()
    defer c.mu.Unlock()
    c.entries[hash] = findings
}

// Clear empties the cache (e.g., on config reload)
func (c *Cache) Clear()
```

**Cache Invalidation**:
- On config file change: clear entire cache
- No TTL: content hash is sufficient
- No size limit in Phase 2 (assume reasonable workspace size)

**Performance Target**: Cache hit should be <1ms vs ~50ms for full scan.

---

## 7. Error Handling Strategy

### 7.1 Error Categories

1. **Initialization Errors** (fatal): Can't start server
   - Invalid config file
   - Can't initialize gitleaks detector
   - **Action**: Log and exit with error code

2. **Scan Errors** (recoverable): Problem scanning a file
   - File too large
   - Invalid content
   - Gitleaks detector error
   - **Action**: Log error, publish empty diagnostics, continue

3. **LSP Protocol Errors** (recoverable): Malformed request
   - Invalid parameters
   - Unknown document URI
   - **Action**: Log warning, return error to client

### 7.2 Error Logging Pattern

```go
// All errors logged with structured logging
slog.Error("scan failed",
    "uri", uri,
    "error", err,
    "duration_ms", elapsed.Milliseconds())

// Non-critical issues as warnings
slog.Warn("config file not found, using defaults",
    "path", configPath)

// Important events as info
slog.Info("scanner initialized",
    "config", configPath,
    "rules", len(rules))
```

### 7.3 Graceful Degradation

```go
// Example: Handle large files gracefully
func (s *Scanner) ScanContent(ctx context.Context, filename, content string) ([]Finding, error) {
    const maxSize = 1_000_000 // 1MB
    
    if len(content) > maxSize {
        slog.Warn("file too large, skipping scan",
            "filename", filename,
            "size", len(content))
        return nil, nil // Return empty findings, not error
    }
    
    // ... continue with scan
}
```

---

## 8. Concurrency Model

### 8.1 Threading Model (Phase 1)

**Simple approach**: Handle each LSP request synchronously
- glsp handles concurrent requests via goroutines
- Our handlers run sequentially per document
- No explicit goroutine management needed

**Why**: Scanning is fast enough (<50ms target), no need for async.

### 8.2 Synchronization Points

1. **DocumentStore**: RWMutex for concurrent access
   ```go
   type DocumentStore struct {
       mu        sync.RWMutex
       documents map[string]*Document
   }
   ```

2. **Cache** (Phase 2): RWMutex for concurrent reads
   ```go
   type Cache struct {
       mu      sync.RWMutex
       entries map[[32]byte][]Finding
   }
   ```

3. **Config Watcher**: Single goroutine, uses channel for events
   ```go
   go func() {
       for {
           select {
           case event := <-watcher.Events:
               handleConfigChange(event)
           case <-ctx.Done():
               return
           }
       }
   }()
   ```

### 8.3 Concurrency (Phase 3: Workspace Scan)

```go
// Scan multiple files in parallel
func (s *Server) ScanWorkspace(ctx context.Context, files []string) error {
    var wg sync.WaitGroup
    sem := make(chan struct{}, 10) // Limit to 10 concurrent scans
    
    for _, file := range files {
        wg.Add(1)
        go func(f string) {
            defer wg.Done()
            sem <- struct{}{} // Acquire
            defer func() { <-sem }() // Release
            
            s.scanFile(ctx, f)
        }(file)
    }
    
    wg.Wait()
    return nil
}
```

---

## 9. Testing Strategy

### 9.1 Unit Tests

**scanner_test.go**:
```go
func TestScanner_DetectsAWSKey(t *testing.T) {
    scanner, err := NewScanner("")
    require.NoError(t, err)
    
    content := `
        package main
        const key = "AKIAIOSFODNN7EXAMPLE"
    `
    
    findings, err := scanner.ScanContent(context.Background(), "test.go", content)
    require.NoError(t, err)
    assert.Len(t, findings, 1)
    assert.Equal(t, "aws-access-key", findings[0].RuleID)
}

func TestScanner_HandlesLargeFile(t *testing.T) {
    scanner, err := NewScanner("")
    require.NoError(t, err)
    
    // 2MB file
    content := strings.Repeat("x", 2_000_000)
    
    findings, err := scanner.ScanContent(context.Background(), "large.txt", content)
    require.NoError(t, err)
    assert.Empty(t, findings) // Should skip, not error
}
```

**diagnostics_test.go**:
```go
func TestFindingToDiagnostic(t *testing.T) {
    finding := Finding{
        RuleID:      "test-rule",
        Description: "Test Secret",
        StartLine:   10,
        StartColumn: 5,
        EndLine:     10,
        EndColumn:   20,
    }
    
    diag := FindingToDiagnostic(finding)
    
    assert.Equal(t, uint32(9), diag.Range.Start.Line) // 0-indexed
    assert.Equal(t, uint32(4), diag.Range.Start.Character)
    assert.Equal(t, "gitleaks", diag.Source)
}
```

### 9.2 Integration Tests

**integration_test.go**:
```go
func TestLSPIntegration(t *testing.T) {
    // Start server
    server := startTestServer(t)
    defer server.Stop()
    
    // Send initialize
    resp := server.SendRequest("initialize", initParams)
    assert.NotNil(t, resp.Result)
    
    // Open document with secret
    server.SendNotification("textDocument/didOpen", didOpenParams)
    
    // Expect publishDiagnostics
    diag := waitForDiagnostics(t, server, 1*time.Second)
    assert.Len(t, diag.Diagnostics, 1)
    assert.Contains(t, diag.Diagnostics[0].Message, "AWS")
}
```

### 9.3 Benchmark Tests

**scanner_bench_test.go**:
```go
func BenchmarkScanner_SmallFile(b *testing.B) {
    scanner, _ := NewScanner("")
    content := readTestFile("small.go") // ~100 lines
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        scanner.ScanContent(context.Background(), "test.go", content)
    }
}

func BenchmarkScanner_MediumFile(b *testing.B) {
    scanner, _ := NewScanner("")
    content := readTestFile("medium.go") // ~1000 lines
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        scanner.ScanContent(context.Background(), "test.go", content)
    }
}

func BenchmarkCache_Hit(b *testing.B) {
    cache := NewCache()
    content := "test content"
    cache.Put(content, []Finding{})
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        cache.Get(content)
    }
}
```

**Performance Targets**:
- `BenchmarkScanner_SmallFile`: <10ms per operation
- `BenchmarkScanner_MediumFile`: <50ms per operation
- `BenchmarkCache_Hit`: <1µs per operation

---

## 10. Dependencies and Their Usage

### 10.1 glsp (LSP Server)

```go
import (
    "github.com/tliron/glsp"
    protocol "github.com/tliron/glsp/protocol_3_16"
    "github.com/tliron/glsp/server"
)

// Create server
handler := protocol.Handler{
    Initialize:            handleInitialize,
    TextDocumentDidOpen:   handleDidOpen,
    TextDocumentDidChange: handleDidChange,
    TextDocumentDidSave:   handleDidSave,
}

glspServer := server.NewServer(&handler, "gitleaks-ls", false)
glspServer.RunStdio()
```

**Key APIs Used**:
- `server.NewServer()` - Create LSP server
- `server.RunStdio()` - Start serving over stdio
- `server.PublishDiagnostics()` - Send diagnostics to client

### 10.2 gitleaks/v8 (Detection Engine)

```go
import (
    "github.com/gitleaks/gitleaks/v8/config"
    "github.com/gitleaks/gitleaks/v8/detect"
    "github.com/gitleaks/gitleaks/v8/report"
)

// Load config
cfg, err := config.NewConfig("path/to/.gitleaks.toml")
if err != nil {
    cfg = config.DefaultConfig() // Use defaults
}

// Create detector
detector := detect.NewDetector(cfg)

// Scan content
fragment := detect.Fragment{
    Raw: content,
    FilePath: filename,
}

findings := detector.DetectString(fragment)
```

**Key Types**:
- `config.ViperConfig` - Configuration
- `detect.Detector` - Detection engine
- `report.Finding` - Detection result

### 10.3 fsnotify (File Watching)

```go
import "github.com/fsnotify/fsnotify"

watcher, err := fsnotify.NewWatcher()
watcher.Add(configPath)

go func() {
    for {
        select {
        case event := <-watcher.Events:
            if event.Op&fsnotify.Write == fsnotify.Write {
                reloadConfig()
            }
        case err := <-watcher.Errors:
            log.Error("watcher error", err)
        }
    }
}()
```

### 10.4 slog (Logging)

```go
import "log/slog"

// Setup in main()
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))
slog.SetDefault(logger)

// Use throughout code
slog.Info("server started", "version", version)
slog.Error("scan failed", "uri", uri, "error", err)
slog.Debug("cache hit", "hash", hash)
```

---

## 11. Build and Deployment

### 11.1 Build Configuration

**Makefile**:
```makefile
.PHONY: build test bench clean

build:
	go build -o gitleaks-ls main.go

test:
	go test -v ./...

bench:
	go test -bench=. -benchmem ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	golangci-lint run

clean:
	rm -f gitleaks-ls coverage.out
```

### 11.2 Installation

**For Development**:
```bash
go build -o gitleaks-ls
sudo mv gitleaks-ls /usr/local/bin/
```

**For Users** (Phase 4):
```bash
go install github.com/user/gitleaks-ls@latest
```

---

## 12. Performance Budgets

### 12.1 Latency Targets

| Operation | Target | Measurement |
|-----------|--------|-------------|
| Scan small file (<100 lines) | <10ms | p95 |
| Scan medium file (<1000 lines) | <50ms | p95 |
| Scan large file (<10k lines) | <200ms | p95 |
| Cache hit | <1ms | p95 |
| Diagnostic publish | <5ms | p95 |
| Server startup | <500ms | max |

### 12.2 Memory Targets

| Component | Target | Max |
|-----------|--------|-----|
| Server baseline | <10MB | 20MB |
| Per document | <1MB | 5MB |
| Cache (100 files) | <20MB | 50MB |
| Total (typical workspace) | <50MB | 100MB |

### 12.3 CPU Targets

| Scenario | Target | Max |
|----------|--------|-----|
| Idle | <1% | 2% |
| Active scanning | <20% | 50% |
| Workspace scan (100 files) | <5 seconds | 10 seconds |

---

## 13. Open Technical Questions

### 13.1 Resolved for Phase 1

✅ **Q**: Use glsp or custom LSP implementation?  
**A**: Use glsp - mature, well-documented, actively maintained.

✅ **Q**: Store documents in memory or read from disk?  
**A**: In memory - LSP provides content via didOpen/didChange.

✅ **Q**: Scan on every keystroke or only on save?  
**A**: Only on save for Phase 1. Add debounced onChange in Phase 2 if needed.

✅ **Q**: How to handle gitleaks configuration?  
**A**: Auto-detect .gitleaks.toml in workspace root, fall back to defaults.

### 13.2 Resolved in Phase 2-3

✅ **Q**: Should cache be persisted to disk?  
**A**: No, in-memory content-hash cache is sufficient and fast.

✅ **Q**: Should we support custom ignore patterns beyond .gitleaksignore?  
**A**: Yes, workspace scanning respects .gitignore patterns.

### 13.3 To Be Decided in Phase 4+

❓ **Q**: How to handle multiple workspace folders?  
**Proposal**: LSP provides rootUri, use that. Multi-folder support if needed.

---

## 14. Implementation Checklist

### Phase 1: Core LSP
- [x] Initialize go.mod with dependencies
- [x] Create file structure (main.go, scanner.go, etc.)
- [x] Implement scanner.go (gitleaks wrapper)
- [x] Implement diagnostics.go with line/column conversion
- [x] Implement main.go (glsp initialization)
- [x] Add didOpen/didChange/didSave handlers
- [x] Implement config.go with .gitleaks.toml support

### Phase 2: Enhanced Features
- [x] Hover provider with finding details
- [x] Code actions for 40+ languages
- [x] Content-hash caching
- [x] .gitleaksignore support

### Phase 3: Performance
- [x] Benchmark suite
- [x] Workspace scanning with parallel execution
- [x] .gitignore support for workspace scan
- [x] File size limits and binary file detection

### Phase 4: Polish & Stability
- [x] Integration tests
- [x] Memory leak testing (automated stress test)
- [x] Enhanced error handling with context
- [x] Structured logging improvements
- [x] Final documentation review

### Phase 5: CI/CD & Enhancements
- [x] GitHub Actions CI (test matrix, lint, build)
- [x] Release workflow with cross-platform binaries
- [x] Coverage threshold (70% minimum)
- [x] Benchmark CI job
- [x] Progress reporting for workspace scans
- [x] Configurable diagnostic severity
- [x] golangci-lint configuration

---

**Document Version**: 1.4  
**Last Updated**: 2025-12-05
