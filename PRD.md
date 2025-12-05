# Product Requirements Document: Gitleaks Language Server

## 1. Executive Summary

### 1.1 Product Vision
Create a Language Server Protocol (LSP) implementation for Gitleaks that provides real-time secret detection capabilities directly within code editors and IDEs, enabling developers to identify and prevent secret leaks before committing code.

### 1.2 Problem Statement
Developers often accidentally commit sensitive information (API keys, passwords, tokens, credentials) to version control systems. Current solutions like pre-commit hooks and CI/CD scanning catch secrets too late in the development workflow, after code has been written and staged. This leads to:
- Security incidents requiring secret rotation
- Delayed development cycles when secrets are caught in CI
- Increased cognitive load from context switching
- Potential exposure of secrets in git history

### 1.3 Solution
A gitleaks-powered language server that integrates seamlessly with any LSP-compatible editor (VS Code, Neovim, Emacs, Sublime Text, etc.) to provide:
- Real-time secret detection as developers type
- Inline diagnostics and warnings
- Quick fixes and remediation suggestions
- Context-aware intelligence about detected secrets

## 2. Goals and Objectives

### 2.1 Primary Goals
1. **Shift-Left Security**: Catch secrets at write-time, not commit-time
2. **Developer Experience**: Provide frictionless, non-intrusive security feedback
3. **Universal Compatibility**: Support all LSP-compatible editors and IDEs
4. **Performance**: Maintain sub-100ms response time for real-time feedback
5. **Accuracy**: Leverage gitleaks' proven detection engine with minimal false positives

### 2.2 Success Metrics
**Performance is the primary success metric for this internal/prototype phase:**
- **Response Time**: <100ms for 95th percentile diagnostic requests
- **Scan Time**: <50ms for typical files (<1000 lines)
- **Memory Usage**: <50MB for typical workspaces (10-100 files)
- **CPU Usage**: <5% average CPU during idle, <20% during active scanning
- **Startup Time**: <500ms language server initialization
- **Throughput**: Handle 100+ file scans per second for workspace scanning

### 2.3 Non-Goals (Out of Scope for Prototype)
- Secret management or rotation capabilities
- Automated secret remediation
- Cloud-based secret scanning services
- Git history scanning (use gitleaks CLI instead)
- Custom rule creation UI (use gitleaks config files)
- Editor extensions/plugins (raw LSP only)
- Multi-user or enterprise features
- Public distribution or marketplace publishing
- User documentation or marketing materials

## 3. User Personas

### 3.1 Primary Persona: Security-Conscious Developer
- **Background**: Software engineer working on cloud-native applications
- **Pain Points**: Accidentally committed secrets, slow CI feedback loops
- **Needs**: Immediate feedback, minimal disruption to workflow
- **Technical Level**: Comfortable with CLI tools and editor configuration

### 3.2 Secondary Persona: Security/DevSecOps Engineer
- **Background**: Responsible for implementing security controls across teams
- **Pain Points**: Difficulty enforcing security practices, lack of visibility
- **Needs**: Centralized configuration, metrics, policy enforcement
- **Technical Level**: Advanced technical knowledge, infrastructure experience

### 3.3 Tertiary Persona: Junior Developer
- **Background**: New to secure coding practices
- **Pain Points**: Unaware of security risks, unclear about what constitutes a secret
- **Needs**: Educational feedback, clear guidance on remediation
- **Technical Level**: Basic development skills, learning security concepts

## 4. Core Features

### 4.1 Real-Time Secret Detection (P0)
**Description**: Scan file contents as the user types and provide immediate feedback.

**Requirements**:
- Scan on file open, edit, and save events
- Support all file types and languages
- Use gitleaks detection engine and default rule set
- Debounce scanning to avoid performance issues (300ms default)
- Cache scan results to minimize redundant work

**User Stories**:
- As a developer, I want to see warnings when I type a secret so I can correct it immediately
- As a developer, I want scanning to feel instant so my workflow isn't disrupted

### 4.2 Inline Diagnostics (P0)
**Description**: Display detected secrets as editor diagnostics with severity levels.

**Requirements**:
- Show diagnostics as warnings or errors (configurable)
- Underline the exact secret location in the editor
- Support diagnostic ranges that span multiple lines if needed
- Clear diagnostics when secrets are removed
- Respect `.gitleaksignore` files and `gitleaks:allow` comments

**Diagnostic Information**:
- Rule ID and description
- Secret type (e.g., "AWS Access Key", "GitHub Token")
- Line and column numbers
- Entropy score (if applicable)
- Confidence level

**User Stories**:
- As a developer, I want to see exactly where the secret is so I can fix it quickly
- As a developer, I want to understand what type of secret was detected

### 4.3 Hover Documentation (P0)
**Description**: Provide detailed information when hovering over a detected secret.

**Requirements**:
- Show full rule details including pattern
- Display remediation guidance
- Include links to documentation
- Show how to suppress false positives
- Provide entropy analysis details

**User Stories**:
- As a developer, I want to understand why something was flagged as I hover over it
- As a developer, I want quick access to documentation about the detected secret type

### 4.4 Code Actions / Quick Fixes (P1)
**Description**: Offer actionable fixes for detected secrets.

**Requirements**:
- "Ignore this occurrence" - add `gitleaks:allow` comment
- "Add to .gitleaksignore" - append fingerprint to ignore file
- "View rule details" - open documentation
- "Copy fingerprint" - copy to clipboard for allowlisting

**User Stories**:
- As a developer, I want to quickly suppress false positives without leaving my editor
- As a developer, I want to easily allowlist test/example secrets

### 4.5 Configuration Support (P0)
**Description**: Support gitleaks configuration files and custom rules.

**Requirements**:
- Auto-detect `.gitleaks.toml` in workspace root
- Support custom config path via LSP settings
- Hot-reload configuration on file changes
- Fallback to gitleaks default config if none found
- Validate configuration and show errors

**Configuration Options**:
```json
{
  "gitleaks-ls.configPath": "",
  "gitleaks-ls.enabled": true,
  "gitleaks-ls.severity": "warning",
  "gitleaks-ls.scanMode": "onChange",
  "gitleaks-ls.debounceMs": 300,
  "gitleaks-ls.maxFileSizeMB": 1,
  "gitleaks-ls.enabledRules": [],
  "gitleaks-ls.disabledRules": [],
  "gitleaks-ls.logLevel": "info"
}
```

**User Stories**:
- As a security engineer, I want to enforce custom detection rules across my team
- As a developer, I want the language server to respect my project's gitleaks config

### 4.6 Workspace Scanning (P1)
**Description**: Scan entire workspace/project on demand.

**Requirements**:
- Command to scan all files in workspace
- Progress reporting for large workspaces
- Generate summary report of findings
- Option to scan only open files vs. all files
- Exclude files based on `.gitignore` patterns

**User Stories**:
- As a developer, I want to scan my entire project before committing
- As a security engineer, I want to audit a new codebase for existing secrets

### 4.7 Ignore File Support (P0)
**Description**: Respect `.gitleaksignore` files for suppressing false positives.

**Requirements**:
- Auto-detect `.gitleaksignore` in workspace root
- Support fingerprint-based ignoring
- Support path-based ignoring
- Hot-reload on ignore file changes
- Provide command to add current finding to ignore file

**User Stories**:
- As a developer, I want to maintain a list of known false positives
- As a team, we want to share approved exceptions via version control

## 5. Technical Architecture

### 5.1 Technology Stack
**Core Principle**: Use modern, popular, well-maintained libraries. Keep code simple.

- **Language**: Go 1.21+ (for consistency with gitleaks, excellent performance)
- **LSP Library**: `github.com/tliron/glsp` (most popular, actively maintained, simple API)
  - Alternative: `gopls` libraries if needed for specific features
- **JSON-RPC**: Built into glsp (handles LSP transport layer)
- **Detection Engine**: `github.com/gitleaks/gitleaks/v8` (import as library)
- **Configuration**: Use gitleaks' native TOML parser (`github.com/pelletier/go-toml/v2`)
- **File Watching**: `github.com/fsnotify/fsnotify` (standard library for file events)
- **Logging**: `log/slog` (Go 1.21+ standard library, structured logging)
- **Testing**: Go standard testing + `github.com/stretchr/testify` (assertions)
- **Benchmarking**: Go standard `testing.B` (built-in benchmarking)

### 5.2 System Components

**Design Principle**: Keep it simple. Avoid over-engineering. Use standard patterns.

```
┌─────────────────────────────────────────────────────┐
│                 Neovim LSP Client                    │
└───────────────────┬─────────────────────────────────┘
                    │ LSP Protocol (JSON-RPC over stdio)
                    │
┌───────────────────▼─────────────────────────────────┐
│            Gitleaks Language Server                  │
│                                                       │
│  main.go - Entry point, glsp server setup            │
│                                                       │
│  handlers.go - LSP request handlers                  │
│    • didOpen/didChange/didSave → trigger scan       │
│    • hover → return diagnostic details               │
│    • codeAction → return ignore actions              │
│                                                       │
│  scanner.go - Gitleaks integration (simple wrapper)  │
│    • ScanFile(content) → []Finding                   │
│    • LoadConfig() → use gitleaks defaults            │
│                                                       │
│  cache.go - Simple in-memory cache                   │
│    • map[fileURI][]Diagnostic (that's it)            │
│                                                       │
│  config.go - Configuration loader                    │
│    • Find .gitleaks.toml in workspace                │
│    • Watch for changes with fsnotify                 │
│                                                       │
└─────────────────────────────────────────────────────┘
```

**File Structure** (keep it flat, no deep nesting):
```
gitleaks-ls/
├── main.go           # Entry point, server initialization
├── handlers.go       # LSP message handlers
├── scanner.go        # Gitleaks wrapper
├── cache.go          # Simple caching
├── config.go         # Config loading/watching
├── diagnostics.go    # Convert findings to LSP diagnostics
├── go.mod            # Dependencies
├── go.sum
├── README.md
└── *_test.go         # Tests alongside source files
```

### 5.3 Performance Considerations
**Keep It Simple First, Optimize Later**

**Phase 1 - Simple approach**:
- Scan entire file on every change (with 300ms debounce)
- Simple map-based cache: `map[uri][]Diagnostic`
- No fancy algorithms - just fast regex matching via gitleaks

**Phase 2 - Add caching**:
- Cache by content hash: `hash(fileContent) → []Finding`
- Skip scan if hash hasn't changed

**Phase 3 - Optimize if needed**:
- Parallel workspace scanning (simple `sync.WaitGroup`)
- File size limits (skip if >1MB)
- Consider incremental scanning only if performance targets not met

**Avoid Premature Optimization**:
- No complex data structures
- No custom scheduling/queuing
- No sophisticated debouncing (use `time.AfterFunc`)
- Rely on gitleaks' optimized regex engine

### 5.4 Supported LSP Features

| Feature | Priority | Status |
|---------|----------|--------|
| textDocument/publishDiagnostics | P0 | Required |
| textDocument/didOpen | P0 | Required |
| textDocument/didChange | P0 | Required |
| textDocument/didSave | P0 | Required |
| textDocument/didClose | P1 | Nice-to-have |
| textDocument/hover | P0 | Required |
| textDocument/codeAction | P1 | Required |
| workspace/didChangeConfiguration | P1 | Required |
| workspace/executeCommand | P1 | Optional |
| $/cancelRequest | P2 | Optional |

## 6. User Experience

### 6.1 Installation & Setup
1. Build language server binary: `go build -o gitleaks-ls`
2. Add to PATH or note binary location
3. Configure Neovim LSP client in `init.lua`:
```lua
vim.lsp.start({
  name = 'gitleaks-ls',
  cmd = {'/path/to/gitleaks-ls'},
  root_dir = vim.fs.dirname(vim.fs.find({'.git'}, { upward = true })[1]),
})
```
4. Optional: Add `.gitleaks.toml` to project root for custom rules

### 6.2 Typical Workflow
1. Developer opens file in editor
2. Language server scans file and shows diagnostics
3. Developer hovers over warning to see details
4. Developer either:
   - Removes the secret
   - Replaces with environment variable
   - Adds `gitleaks:allow` comment for false positives
   - Adds to `.gitleaksignore` for persistent exceptions
5. Diagnostics clear when issue is resolved

### 6.3 Example Diagnostic Output
```
Warning: Detected AWS Access Key [gitleaks:aws-access-key]
Line 42: aws_key = "AKIAIOSFODNN7EXAMPLE"
           ^^^^^^^^^^^^^^^^^^^^^^^^
Entropy: 4.2 | Confidence: High

🛠️ Quick Fixes:
  • Ignore this occurrence (add gitleaks:allow comment)
  • Add to .gitleaksignore
  • Learn more about AWS secret management

💡 Recommendation: Store credentials in environment variables or use AWS IAM roles.
```

## 7. Editor Integration

### 7.1 Neovim (Raw LSP Only)
**Target**: Neovim with native LSP client (`:h lsp`)

**Setup Method**: Manual configuration via `vim.lsp.start()` or `lspconfig`
- No custom plugin required
- Use Neovim's built-in LSP client
- Standard LSP protocol only
- Configuration via `init.lua`

**Example Configuration**:
```lua
vim.lsp.start({
  name = 'gitleaks-ls',
  cmd = {'gitleaks-ls'},
  root_dir = vim.fs.dirname(vim.fs.find({'.git'}, { upward = true })[1]),
})
```

### 7.2 Future Editors (Post-Prototype)
- VS Code extension
- Emacs (lsp-mode)
- Other LSP-compatible editors
- Custom plugins and enhanced integrations

## 8. Configuration Examples

### 8.1 Basic Editor Configuration (VS Code)
```json
{
  "gitleaks-ls.enabled": true,
  "gitleaks-ls.severity": "warning",
  "gitleaks-ls.scanMode": "onChange"
}
```

### 8.2 Custom Rules Configuration (`.gitleaks.toml`)
```toml
[extend]
useDefault = true

[[rules]]
id = "custom-api-key"
description = "Custom API Key Pattern"
regex = '''(?i)custom[_-]?api[_-]?key[:\s=]+['"]?([a-zA-Z0-9]{32})['"]?'''
keywords = ["custom_api_key"]
```

### 8.3 Ignore Configuration (`.gitleaksignore`)
```
# Example test credentials
abc123def456:src/tests/fixtures/credentials.txt:test-api-key:10

# False positive in documentation
**/docs/examples/**
```

## 9. Security & Privacy

### 9.1 Data Handling
- **No External Communication**: All scanning happens locally
- **No Telemetry by Default**: Optional, opt-in anonymous usage stats
- **No Secret Transmission**: Detected secrets never leave the user's machine
- **Secure Logging**: Redact secrets from logs by default

### 9.2 Performance & Resource Usage
- **Memory**: Target <100MB for typical workspaces
- **CPU**: Background scanning with low priority threads
- **Disk**: Minimal disk I/O, read-only access to workspace files
- **Network**: None (except for optional update checks)

## 10. Testing Strategy

### 10.1 Unit Tests
- LSP message handling (request/response parsing)
- Configuration parsing
- Ignore file handling
- Cache invalidation logic
- Diagnostic creation and formatting

### 10.2 Integration Tests
- Basic LSP protocol compliance (initialize, shutdown)
- File scanning with gitleaks library
- Configuration hot-reload
- Memory leak detection (long-running sessions)

### 10.3 Performance Tests (CRITICAL)
- Benchmark suite for all performance metrics
- Load testing (100+ files)
- Stress testing (large files, rapid edits)
- Memory profiling
- CPU profiling
- Response time percentiles (p50, p95, p99)

### 10.4 Test Coverage Goals
- LSP handlers: >80%
- Core scanning logic: >75%
- Configuration: >70%
- Overall: >75% (focus on correctness over coverage)

## 11. Development Plan

### 11.1 Phase 1: Core LSP (v0.1.0) - Week 1-2
**Focus**: Simplest possible working implementation

**Files to Create** (~500 lines total):
- `main.go`: glsp server setup, stdio transport (50 lines)
- `handlers.go`: didOpen, didChange, didSave handlers (100 lines)
- `scanner.go`: Call gitleaks library, return findings (50 lines)
- `diagnostics.go`: Convert gitleaks findings to LSP diagnostics (50 lines)
- `config.go`: Load .gitleaks.toml if exists, else defaults (50 lines)

**Key Implementation Details**:
- Use glsp's `NewServer()` with stdio protocol
- Scan on didOpen/didSave immediately (no debouncing yet)
- Store diagnostics in simple `map[string][]protocol.Diagnostic`
- Publish diagnostics with `server.PublishDiagnostics()`

**Success Criteria**: Can open file in Neovim and see secret warnings

### 11.2 Phase 2: Enhanced Features (v0.2.0) - Week 3-4
**Focus**: Essential UX improvements

**Add** (~300 lines):
- `hover.go`: Return finding details on hover (50 lines)
- `actions.go`: Code actions for adding `gitleaks:allow` comment (80 lines)
- `cache.go`: Simple hash-based cache `map[hash][]Finding` (50 lines)
- `debounce.go`: Simple debouncing with `time.AfterFunc` (30 lines)
- `.gitleaksignore` support in scanner.go (50 lines)

**Keep It Simple**:
- Hover: just format the finding as markdown
- Code actions: single action to insert `// gitleaks:allow` on previous line
- Cache: hash file content, store results, check before scanning
- Debounce: cancel previous timer, start new one

**Success Criteria**: Productive workflow for handling secrets and false positives

### 11.3 Phase 3: Performance Optimization (v0.3.0) - Week 5-6
**Focus**: Measure first, optimize what matters

**Add** (~200 lines):
- `bench_test.go`: Comprehensive benchmarks (100 lines)
- `workspace.go`: Parallel workspace scanning with WaitGroup (50 lines)
- Profile-guided optimizations based on benchmark results
- File size limits and early returns

**Methodology**:
1. Write benchmarks for all operations (scan, cache, publish)
2. Run `go test -bench` and `pprof` to find bottlenecks
3. Optimize ONLY the slow paths
4. Keep code simple - no clever tricks without proven benefit

**Simple Optimizations**:
- Add file size check before scanning
- Use `sync.Map` instead of mutex if contention detected
- Reuse gitleaks detector instance instead of recreating

**Success Criteria**: All performance metrics met consistently

### 11.4 Phase 4: Polish & Stability (v0.4.0) - Week 7-8
**Focus**: Make it robust and debuggable

**Add** (~300 lines):
- Proper error handling with context (wrap errors)
- Structured logging with `slog` (info, debug levels)
- `workspace/executeCommand` for manual scan
- Integration tests using real LSP messages (150 lines)
- README with setup instructions

**Error Handling Pattern**:
```go
if err != nil {
    slog.Error("failed to scan file", "uri", uri, "error", err)
    return // fail gracefully, don't crash
}
```

**Testing Approach**:
- Unit tests for each package
- Integration test: send didOpen, verify diagnostics received
- Benchmark tests for performance validation
- Manual testing in Neovim

**Success Criteria**: Stable for daily development use

## 12. Documentation Requirements (Minimal for Prototype)

### 12.1 Developer Documentation (Essential)
- Architecture overview
- Build and run instructions
- Neovim setup example (`init.lua` config)
- Performance benchmarking guide
- Debugging guide

### 12.2 Code Documentation
- Inline code comments for complex logic
- Package-level documentation
- API documentation for key interfaces

### 12.3 Testing Documentation
- How to run tests
- Performance test suite usage
- Test coverage expectations

## 13. Success Criteria (Prototype Phase)

### 13.1 Performance Benchmarks (PRIMARY)
**Must achieve consistently:**
- ✅ <100ms response time for 95th percentile diagnostic requests
- ✅ <50ms scan time for files <1000 lines
- ✅ <500ms language server startup time
- ✅ <50MB memory usage for typical workspace (10-100 files)
- ✅ <5% CPU usage when idle
- ✅ <20% CPU usage during active scanning
- ✅ 100+ files/second throughput for workspace scanning

### 13.2 Functional Success
- ✅ Detects secrets in real-time (on didChange/didSave)
- ✅ Shows accurate diagnostics with proper ranges
- ✅ Hover provides useful information
- ✅ Code actions work for ignoring false positives
- ✅ Respects `.gitleaks.toml` and `.gitleaksignore`
- ✅ Works reliably in Neovim via native LSP client

### 13.3 Stability Success
- ✅ Zero crashes during normal operation
- ✅ No memory leaks over 8+ hour sessions
- ✅ Handles large files (>10k lines) gracefully
- ✅ Recovers from configuration errors
- ✅ Clean error messages and logging

## 14. Risks & Mitigations

### 14.1 Technical Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Performance degrades on large files | High | Medium | Implement file size limits, incremental scanning |
| High false positive rate | High | Medium | Extensive testing, community feedback loop |
| LSP compatibility issues | Medium | Low | Follow LSP spec strictly, test with multiple editors |
| Memory leaks in long-running sessions | Medium | Low | Regular profiling, leak detection tests |
| Gitleaks library API changes | Low | Medium | Pin to stable version, abstract integration layer |

### 14.2 Development Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Performance targets not achievable | High | Medium | Early prototyping, profiling, alternative algorithms |
| Gitleaks library integration issues | Medium | Medium | Test early, consider vendoring or forking |
| LSP protocol complexity | Medium | Low | Use well-tested LSP libraries, reference implementations |
| Neovim LSP client quirks | Low | Medium | Test thoroughly, consult Neovim docs/community |

## 15. Dependencies

### 15.1 External Dependencies (Minimal, Modern, Popular)

**Runtime Dependencies**:
```go
require (
    github.com/gitleaks/gitleaks/v8 v8.18.0+   // Detection engine
    github.com/tliron/glsp v0.2.0+              // LSP server (5k+ stars)
    github.com/fsnotify/fsnotify v1.7.0+        // File watching (9k+ stars)
    github.com/pelletier/go-toml/v2 v2.1.0+     // TOML parsing (1.5k+ stars)
)
```

**Development Dependencies**:
```go
require (
    github.com/stretchr/testify v1.8.4+         // Test assertions (22k+ stars)
)
```

**Rationale for Each Library**:
- **glsp**: Most complete Go LSP library, active maintenance, simple API
- **gitleaks/v8**: Core requirement, battle-tested, comprehensive rules
- **fsnotify**: De facto standard for file watching in Go
- **go-toml/v2**: Fast, compliant, used by gitleaks itself
- **testify**: Industry standard for Go testing

**No External Tools Required**:
- Use Go standard library for HTTP, JSON, logging (slog)
- No build tools beyond `go build`
- No package managers beyond Go modules

### 15.2 Development Dependencies
- **Testing**: Go standard library + testify
- **Benchmarking**: Go standard `testing.B`
- **Profiling**: Go standard `pprof`
- **CI/CD**: GitHub Actions (simple Go workflow)
- **Linting**: `golangci-lint` (standard Go linter aggregator)

## 16. Implementation Decisions (KISS Principle)

### Resolved for Simplicity:

1. **Caching Strategy**: Content hash only. Simpler, no path management needed.
   ```go
   hash := sha256.Sum256([]byte(content))
   if cached, ok := cache[hash]; ok { return cached }
   ```

2. **Scan Trigger**: On save only for Phase 1. Add debounced onChange in Phase 2.
   - Simpler to start, sufficient for most use cases
   - Add onChange if users request it

3. **Large Files**: Hard limit 1MB. Return early with info diagnostic.
   ```go
   if len(content) > 1_000_000 {
       return // skip silently or show "file too large" diagnostic
   }
   ```

4. **Configuration Hot Reload**: Automatic with fsnotify. Simple to implement.
   - Watch .gitleaks.toml file
   - Reload and clear cache on change

5. **Logging**: Default INFO level, log to stderr (Neovim captures it).
   ```go
   slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
       Level: slog.LevelInfo,
   })))
   ```

6. **Debouncing**: Simple timer pattern, no libraries needed:
   ```go
   var timer *time.Timer
   if timer != nil { timer.Stop() }
   timer = time.AfterFunc(300*time.Millisecond, func() { scan() })
   ```

### Open Questions for Phase 3+:
- Should we scan git-ignored files? (default: no)
- Incremental scanning worth the complexity? (decide after benchmarks)

## 17. Appendix

### 17.1 Competitive Analysis

| Tool | Type | Pros | Cons |
|------|------|------|------|
| Talisman | Pre-commit hook | Mature, local scanning | Requires git setup, no IDE integration |
| GitGuardian | Cloud service | High accuracy, dashboard | Requires cloud service, privacy concerns |
| detect-secrets | CLI/Pre-commit | Baseline support | Python dependency, no real-time feedback |
| SecretLint | Pre-commit | Rule-based | Node.js dependency, limited rules |

### 17.2 References
- [LSP Specification](https://microsoft.github.io/language-server-protocol/)
- [Gitleaks GitHub](https://github.com/gitleaks/gitleaks)
- [VS Code Extension API](https://code.visualstudio.com/api)
- [Neovim LSP Documentation](https://neovim.io/doc/user/lsp.html)

### 17.3 Glossary
- **LSP**: Language Server Protocol - standardized protocol for editor/IDE integrations
- **Diagnostic**: Editor annotation showing warnings/errors
- **Code Action**: Quick fix or refactoring option
- **Fingerprint**: Unique identifier for a specific secret finding
- **Entropy**: Measure of randomness in a string (used for secret detection)
- **False Positive**: Non-secret incorrectly flagged as a secret

---

**Document Version**: 2.1
**Last Updated**: 2025-12-05

---

## 18. Next Steps: From PRD to Implementation

### 18.1 Recommended Development Sequence

**Step 1: Project Initialization** (30 minutes)
```bash
# Initialize Go module
go mod init github.com/yourusername/gitleaks-ls

# Add dependencies
go get github.com/tliron/glsp@latest
go get github.com/gitleaks/gitleaks/v8@latest
go get github.com/fsnotify/fsnotify@latest
go get github.com/pelletier/go-toml/v2@latest
go get -d github.com/stretchr/testify@latest

# Create project structure
mkdir -p cmd/gitleaks-ls
touch main.go handlers.go scanner.go diagnostics.go config.go
touch README.md .gitignore
```

**Step 2: Create Technical Design Document** (2-3 hours)
- Document LSP message flows (sequence diagrams)
- Define Go interfaces and types
- Specify gitleaks library integration points
- Design cache data structures
- Map out error handling strategy

**Step 3: Implement Phase 1 - Core LSP** (1-2 weeks)
Follow the order that minimizes dependencies:
1. `main.go` - Basic glsp server setup, just accept connections
2. `scanner.go` - Gitleaks wrapper (can test standalone)
3. `diagnostics.go` - Convert findings to LSP format (can test standalone)
4. `handlers.go` - Wire up didOpen/didSave handlers
5. `config.go` - Load .gitleaks.toml
6. Integration testing with real Neovim

**Step 4: Iterate on Phases 2-4** (4-6 weeks)
- Add one feature at a time
- Test each feature before moving to next
- Benchmark continuously

### 18.2 Documents to Create Next

#### A. Technical Design Document (TDD)
**Purpose**: Bridge PRD and code. Answers "how" not "what".

**Contents**:
- LSP protocol flow diagrams
- Go package/type design
- Interface definitions
- Data structure specifications
- Concurrency model
- Error handling strategy

**For Copilot CLI**: Create TDD by having CLI expand on technical sections:
```
"Create a technical design document for the LSP handlers. 
Include Go type definitions for all LSP message handlers, 
the diagnostic cache structure, and scanner interface."
```

#### B. Implementation Tasks (GitHub Issues/TODOs)
**Purpose**: Break work into discrete, testable chunks.

**Example Task Breakdown**:
```markdown
## Phase 1 Tasks

### Task 1.1: Basic LSP Server Setup
- [ ] Create main.go with glsp server initialization
- [ ] Accept stdio connections
- [ ] Handle initialize/shutdown requests
- [ ] Test with Neovim LSP client
- Estimated: 2 hours

### Task 1.2: Gitleaks Scanner Wrapper
- [ ] Create scanner.go with ScanFile function
- [ ] Import gitleaks detector
- [ ] Convert gitleaks.Finding to internal type
- [ ] Write unit tests with known secrets
- Estimated: 4 hours

### Task 1.3: Diagnostic Conversion
- [ ] Create diagnostics.go
- [ ] Convert findings to protocol.Diagnostic
- [ ] Map severity levels
- [ ] Format diagnostic messages
- [ ] Unit tests
- Estimated: 3 hours
...
```

**For Copilot CLI**: Generate issues from PRD:
```
"Generate GitHub issues for Phase 1 implementation tasks. 
Each issue should be 2-4 hours of work, include acceptance 
criteria, and reference the PRD sections."
```

#### C. API/Interface Documentation
**Purpose**: Define contracts before implementation.

**Example**:
```go
// pkg/scanner/scanner.go

// Scanner detects secrets in source code using gitleaks
type Scanner interface {
    // ScanContent scans the provided content and returns findings
    ScanContent(ctx context.Context, content string) ([]Finding, error)
    
    // LoadConfig loads gitleaks configuration from path
    LoadConfig(path string) error
}

// Finding represents a detected secret
type Finding struct {
    RuleID      string
    Description string
    StartLine   int
    StartColumn int
    EndLine     int
    EndColumn   int
    Secret      string // redacted
    Entropy     float64
}
```

**For Copilot CLI**: 
```
"Define Go interfaces for the scanner, cache, and config 
packages based on the PRD architecture section."
```

### 18.3 Optimal Copilot CLI Workflow

**Iterative Development Pattern**:

```bash
# 1. Start with architecture
$ @workspace "Create pkg/scanner/scanner.go with the Scanner 
interface and Finding type based on the PRD section 5.2"

# 2. Implement piece by piece
$ @workspace "Implement the Scanner interface using gitleaks/v8. 
Keep it simple, just wrap the gitleaks detector."

# 3. Add tests as you go
$ @workspace "Write unit tests for scanner.go using testify. 
Test with a sample secret string from AWS access keys."

# 4. Build up incrementally
$ @workspace "Create handlers.go with didOpen handler that 
calls scanner and publishes diagnostics."

# 5. Test integration points
$ @workspace "Create an integration test that sends a 
textDocument/didOpen LSP message and verifies diagnostic output."

# 6. Iterate on features
$ @workspace "Add hover support to handlers.go. Show the 
finding details formatted as markdown."

# 7. Optimize when needed
$ @workspace "Add benchmark tests for ScanContent. We need 
to scan 100 files in under 1 second."
```

**Key Principles for Copilot CLI**:
1. **Incremental**: One file/function at a time
2. **Contextual**: Reference PRD sections explicitly
3. **Testable**: Ask for tests with each implementation
4. **Concrete**: Provide specific examples of input/output
5. **Measurable**: Request benchmarks for performance-critical code

### 18.4 Development Environment Setup

**Prerequisites**:
```bash
# Required
- Go 1.21+ installed
- Neovim 0.8+ with native LSP
- git

# Recommended
- golangci-lint for linting
- delve for debugging
- pprof for profiling
```

**Neovim Test Configuration** (`test-init.lua`):
```lua
-- Minimal config for testing gitleaks-ls
vim.lsp.start({
  name = 'gitleaks-ls',
  cmd = {'./gitleaks-ls'},
  root_dir = vim.fn.getcwd(),
  on_attach = function(client, bufnr)
    print('gitleaks-ls attached')
  end,
})

-- Enable LSP logging
vim.lsp.set_log_level('debug')
```

**Test with**:
```bash
nvim -u test-init.lua test-file-with-secrets.go
```

### 18.5 First Implementation Session Checklist

**Session 1: Project Setup (2-3 hours)**
- [ ] Initialize Go module
- [ ] Add all dependencies
- [ ] Create file structure
- [ ] Write README with build instructions
- [ ] Create .gitignore
- [ ] Set up basic CI (GitHub Actions for `go test`)
- [ ] Create first commit

**Session 2: Core Scanner (3-4 hours)**
- [ ] Implement scanner.go
- [ ] Write unit tests
- [ ] Test with real secrets
- [ ] Verify gitleaks integration works

**Session 3: LSP Basics (4-6 hours)**
- [ ] Implement main.go with glsp setup
- [ ] Add initialize/shutdown handlers
- [ ] Test connection with Neovim
- [ ] Implement didOpen handler
- [ ] End-to-end test: see diagnostics in Neovim

**Deliverable**: Working prototype that shows warnings in Neovim

### 18.6 Suggested File Creation Order

**Order optimized for testing and incremental progress**:

1. `go.mod`, `go.sum` - Dependencies
2. `README.md` - Project documentation
3. `scanner.go` + `scanner_test.go` - Core logic (testable standalone)
4. `diagnostics.go` + `diagnostics_test.go` - Conversion logic (testable standalone)
5. `config.go` + `config_test.go` - Config loading (testable standalone)
6. `main.go` - Wiring everything together
7. `handlers.go` - LSP handlers
8. `integration_test.go` - End-to-end testing

**Rationale**: Build and test independent components first, then integrate.

### 18.7 Quality Gates Per Phase

**Phase 1 Checklist**:
- [ ] `go build` succeeds
- [ ] All tests pass (`go test ./...`)
- [ ] Can connect from Neovim
- [ ] Shows diagnostic for AWS key in test file
- [ ] No crashes on invalid input
- [ ] Basic error messages in logs

**Phase 2 Checklist**:
- [ ] All Phase 1 checks pass
- [ ] Hover shows finding details
- [ ] Code action adds gitleaks:allow comment
- [ ] Cache speeds up second scan (benchmark proves it)
- [ ] Respects .gitleaksignore file

**Phase 3 Checklist**:
- [ ] All Phase 2 checks pass
- [ ] All performance benchmarks meet targets
- [ ] `pprof` shows no memory leaks
- [ ] Can scan 100+ file workspace
- [ ] CPU usage acceptable

**Phase 4 Checklist**:
- [ ] All Phase 3 checks pass
- [ ] Test coverage >75%
- [ ] No panics in stress test
- [ ] README has complete setup instructions
- [ ] Handles configuration errors gracefully

### 18.8 Example First Copilot CLI Commands

**To kick off implementation**:

```bash
# 1. Initialize project structure
$ "Initialize a Go project for gitleaks-ls. Create go.mod with 
module github.com/user/gitleaks-ls, add dependencies from PRD 
section 15.1, create basic file structure from section 5.2"

# 2. Start with scanner
$ "Create pkg/scanner/scanner.go implementing a Scanner interface 
that wraps gitleaks/v8. Include a ScanContent method that takes 
a string and returns findings. Keep it under 100 lines."

# 3. Add tests
$ "Create pkg/scanner/scanner_test.go with testify. Test that 
ScanContent detects an AWS access key AKIAIOSFODNN7EXAMPLE"

# 4. Build diagnostics converter
$ "Create pkg/lsp/diagnostics.go that converts scanner.Finding 
to protocol.Diagnostic from glsp. Map line/column numbers and 
severity levels."

# 5. Wire up main server
$ "Create cmd/gitleaks-ls/main.go that initializes a glsp server 
with stdio transport. Just handle initialize/shutdown for now."

# 6. Add first handler
$ "Add textDocument/didOpen handler to handlers.go. When a file 
opens, scan it with scanner, convert to diagnostics, and publish."

# 7. Test it
$ "Create an integration test that simulates opening a file with 
a secret and verifies we receive a diagnostic message."
```

### 18.9 Common Pitfalls to Avoid

1. **Don't build everything at once** - Build incrementally, test constantly
2. **Don't optimize prematurely** - Get it working first, then optimize
3. **Don't skip tests** - Tests are documentation and safety net
4. **Don't hardcode paths** - Use workspace root detection
5. **Don't ignore errors** - Log all errors with context
6. **Don't block the main goroutine** - Run scans in background
7. **Don't cache without invalidation** - Stale cache is worse than no cache

### 18.10 Success Criteria for "Real Project" Status

**Minimum Viable Project** (End of Phase 1):
- ✅ Builds without errors
- ✅ Connects to Neovim LSP client
- ✅ Detects and displays at least one type of secret
- ✅ Has basic tests
- ✅ README explains how to build and use

**Production-Ready Prototype** (End of Phase 4):
- ✅ All features from PRD implemented
- ✅ Meets all performance targets
- ✅ >75% test coverage
- ✅ Used daily by author without issues
- ✅ Complete documentation

---

## 19. Quick Start Guide for Development

### For the Very First Implementation Step:

```bash
# 1. Create the project
mkdir -p gitleaks-ls
cd gitleaks-ls

# 2. Ask Copilot CLI:
$ @workspace "I'm starting to implement the gitleaks-ls project 
from PRD.md. First, initialize the Go project structure with go.mod, 
basic file stubs (main.go, scanner.go, handlers.go, diagnostics.go, 
config.go), and add all dependencies from section 15.1 of the PRD."

# 3. Then ask:
$ @workspace "Now implement scanner.go based on PRD section 5.2. 
It should wrap gitleaks/v8 and provide a simple ScanContent function. 
Keep it under 100 lines and include inline comments explaining 
how it integrates with gitleaks."

# 4. Test it:
$ @workspace "Create scanner_test.go with a test that verifies we 
can detect the AWS access key pattern. Use testify for assertions."

# 5. Build up from there...
```

**That's it!** The PRD provides the "what" and "why". The TDD (to be created) 
provides the "how". Copilot CLI executes the implementation step by step.
