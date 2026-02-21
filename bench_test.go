package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func init() {
	// Disable logging for benchmarks
	if strings.Contains(strings.Join(os.Args, " "), "-test.bench") {
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	}
}

// --- Scanner Benchmarks ---

func BenchmarkScanner_SmallFile(b *testing.B) {
	scanner := newTestScanner(b)
	// ~50 lines, typical small file
	content := `package main

import "fmt"

const awsKey = "AKIATESTKEYEXAMPLE7A"

func main() {
	fmt.Println("Hello")
}
`
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scanner.ScanContent(ctx, "test.go", content)
	}
}

func BenchmarkScanner_MediumFile(b *testing.B) {
	scanner := newTestScanner(b)
	// ~1000 lines
	var sb strings.Builder
	sb.WriteString("package main\n\n")
	for i := 0; i < 100; i++ {
		sb.WriteString("func function")
		sb.WriteString(string(rune('A' + i%26)))
		sb.WriteString("() {\n")
		sb.WriteString("\tvar x = 1\n")
		sb.WriteString("\tvar y = 2\n")
		sb.WriteString("\tvar z = x + y\n")
		sb.WriteString("\t_ = z\n")
		sb.WriteString("}\n\n")
	}
	// Add a secret in the middle
	sb.WriteString("const awsKey = \"AKIATESTKEYEXAMPLE7A\"\n")
	content := sb.String()

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scanner.ScanContent(ctx, "test.go", content)
	}
}

func BenchmarkScanner_LargeFile(b *testing.B) {
	scanner := newTestScanner(b)
	// ~500KB, below 1MB limit
	var sb strings.Builder
	sb.WriteString("package main\n\n")
	line := "var line = \"some text content here for testing purposes\"\n"
	for sb.Len() < 500000 {
		sb.WriteString(line)
	}
	sb.WriteString("const awsKey = \"AKIATESTKEYEXAMPLE7A\"\n")
	content := sb.String()

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scanner.ScanContent(ctx, "test.go", content)
	}
}

func BenchmarkScanner_NoSecrets(b *testing.B) {
	scanner := newTestScanner(b)
	// Clean file with no secrets
	content := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scanner.ScanContent(ctx, "test.go", content)
	}
}

func BenchmarkScanner_MultipleSecrets(b *testing.B) {
	scanner := newTestScanner(b)
	content := `package main

const awsKey1 = "AKIATESTKEYEXAMPLE7A"
const awsKey2 = "AKIATESTKEY2XAMPLE7B"
const ghToken = "ghp_1234567890abcdefghijklmnopqrstuvwx"
`
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scanner.ScanContent(ctx, "test.go", content)
	}
}

// --- Cache Benchmarks ---

func BenchmarkCache_Hit(b *testing.B) {
	cache := NewCache()
	content := "package main\nconst key = \"AKIATESTKEYEXAMPLE7A\"\n"
	findings := []Finding{{RuleID: "aws-access-key"}}
	cache.Put(content, findings)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(content)
	}
}

func BenchmarkCache_Miss(b *testing.B) {
	cache := NewCache()
	content := "package main\nconst key = \"AKIATESTKEYEXAMPLE7A\"\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(content)
	}
}

func BenchmarkCache_Put(b *testing.B) {
	cache := NewCache()
	findings := []Finding{{RuleID: "test-rule"}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		content := string(rune('a' + i%26)) // Different content each time
		cache.Put(content, findings)
	}
}

func BenchmarkCache_LargeContent(b *testing.B) {
	cache := NewCache()
	// 100KB content for hash benchmarking
	content := strings.Repeat("x", 100000)
	findings := []Finding{}
	cache.Put(content, findings)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(content)
	}
}

// --- Diagnostics Benchmarks ---

func BenchmarkFindingsToDiagnostics_Single(b *testing.B) {
	findings := []Finding{
		{
			RuleID:      "aws-access-key",
			Description: "AWS Access Key",
			StartLine:   10,
			StartColumn: 5,
			EndLine:     10,
			EndColumn:   25,
			Entropy:     4.2,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FindingsToDiagnostics(findings)
	}
}

func BenchmarkFindingsToDiagnostics_Multiple(b *testing.B) {
	findings := make([]Finding, 10)
	for i := range findings {
		findings[i] = Finding{
			RuleID:      "test-rule",
			Description: "Test Secret",
			StartLine:   i * 10,
			StartColumn: 5,
			EndLine:     i * 10,
			EndColumn:   25,
			Entropy:     4.2,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FindingsToDiagnostics(findings)
	}
}

// --- End-to-End Benchmarks ---

func BenchmarkScanAndPublish_CacheHit(b *testing.B) {
	// Setup server
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	_ = SetupServer("")

	content := "package main\nconst key = \"AKIATESTKEYEXAMPLE7A\"\n"
	uri := protocol.DocumentUri("file:///test/bench.go")

	// Pre-populate cache
	globalServer.documents.Set(uri, 1, content)
	ctx := context.Background()
	findings, _ := globalServer.scanner.ScanContent(ctx, uri, content)
	globalServer.cache.Put(content, findings)

	// Create a no-op notify context
	mockContext := &mockGlspContext{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = scanAndPublishBench(mockContext, uri, content)
	}
}

func BenchmarkScanAndPublish_CacheMiss(b *testing.B) {
	// Setup server
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	_ = SetupServer("")

	uri := protocol.DocumentUri("file:///test/bench.go")

	// Create a no-op notify context
	mockContext := &mockGlspContext{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use different content each iteration to force cache miss
		content := "package main\nvar x = " + string(rune('0'+i%10)) + "\nconst key = \"AKIATESTKEYEXAMPLE7A\"\n"
		globalServer.documents.Set(uri, int32(i), content)
		globalServer.cache.Clear() // Force cache miss
		_ = scanAndPublishBench(mockContext, uri, content)
	}
}

// mockGlspContext provides a no-op context for benchmarking
type mockGlspContext struct{}

func (m *mockGlspContext) Notify(method string, params any) {}

// scanAndPublishBench is a benchmark-friendly version without glsp.Context
func scanAndPublishBench(ctx *mockGlspContext, uri protocol.DocumentUri, content string) error {
	var findings []Finding
	var err error

	if cached, ok := globalServer.cache.Get(content); ok {
		findings = cached
	} else {
		bgCtx := context.Background()
		findings, err = globalServer.getScanner().ScanContent(bgCtx, string(uri), content)
		if err != nil {
			return err
		}
		globalServer.cache.Put(content, findings)
	}

	diagnostics := FindingsToDiagnostics(findings)
	globalServer.documents.SetDiagnostics(uri, diagnostics, findings)

	// Simulate notify (no-op)
	ctx.Notify("textDocument/publishDiagnostics", nil)

	return nil
}
