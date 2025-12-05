package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMemoryLeak simulates extended usage to detect memory leaks.
// It performs many scan cycles and checks that memory doesn't grow unboundedly.
func TestMemoryLeak(t *testing.T) {
	// Silence logs during test
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	err := SetupServer("")
	require.NoError(t, err)

	// Content samples of varying sizes
	smallContent := `package main
const key = "AKIATESTKEYEXAMPLE7A"
func main() {}
`
	mediumContent := smallContent + generateCode(100)
	largeContent := smallContent + generateCode(1000)

	contents := []string{smallContent, mediumContent, largeContent}

	// Force GC and get baseline memory
	runtime.GC()
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	ctx := context.Background()
	iterations := 1000

	// Simulate many document open/scan/close cycles
	for i := 0; i < iterations; i++ {
		content := contents[i%len(contents)]
		uri := "file:///test/file.go"

		// Scan (simulates didOpen/didChange)
		findings, err := globalServer.scanner.ScanContent(ctx, uri, content)
		require.NoError(t, err)

		// Store in cache (simulates normal operation)
		globalServer.cache.Put(content, findings)

		// Every 100 iterations, also test cache retrieval
		if i%100 == 0 {
			globalServer.cache.Get(content)
		}

		// Every 200 iterations, clear cache (simulates config reload)
		if i%200 == 0 {
			globalServer.cache.Clear()
		}
	}

	// Force GC and measure final memory
	runtime.GC()
	var final runtime.MemStats
	runtime.ReadMemStats(&final)

	// Calculate memory growth
	heapGrowth := int64(final.HeapAlloc) - int64(baseline.HeapAlloc)
	heapGrowthMB := float64(heapGrowth) / 1024 / 1024

	t.Logf("Memory stats after %d iterations:", iterations)
	t.Logf("  Baseline heap: %.2f MB", float64(baseline.HeapAlloc)/1024/1024)
	t.Logf("  Final heap:    %.2f MB", float64(final.HeapAlloc)/1024/1024)
	t.Logf("  Heap growth:   %.2f MB", heapGrowthMB)
	t.Logf("  Mallocs:       %d", final.Mallocs-baseline.Mallocs)
	t.Logf("  Frees:         %d", final.Frees-baseline.Frees)

	// Memory should not grow more than 50MB for this test
	// (generous limit to account for GC timing variations)
	assert.Less(t, heapGrowthMB, 50.0, "Memory grew too much, possible leak")

	// Most allocations should be freed
	mallocDiff := final.Mallocs - baseline.Mallocs
	freeDiff := final.Frees - baseline.Frees
	freeRatio := float64(freeDiff) / float64(mallocDiff)
	t.Logf("  Free ratio:    %.2f%%", freeRatio*100)

	// At least 90% of allocations should be freed after GC
	assert.Greater(t, freeRatio, 0.90, "Too many allocations not freed")
}

// TestMemoryLeakWorkspaceScan tests workspace scanning doesn't leak
func TestMemoryLeakWorkspaceScan(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	err := SetupServer("")
	require.NoError(t, err)

	// Create temp workspace with 50 files
	tmpDir := t.TempDir()
	for i := 0; i < 50; i++ {
		content := "package test\nfunc example() {}\n"
		filename := filepath.Join(tmpDir, "file"+string(rune('a'+i%26))+".go")
		require.NoError(t, os.WriteFile(filename, []byte(content), 0644))
	}

	runtime.GC()
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	ctx := context.Background()

	// Run workspace scan multiple times
	for i := 0; i < 10; i++ {
		result, err := globalServer.ScanWorkspace(ctx, tmpDir, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Clear between scans
		globalServer.cache.Clear()
	}

	runtime.GC()
	var final runtime.MemStats
	runtime.ReadMemStats(&final)

	heapGrowthMB := float64(int64(final.HeapAlloc)-int64(baseline.HeapAlloc)) / 1024 / 1024

	t.Logf("Workspace scan memory (10 scans of 50 files):")
	t.Logf("  Heap growth: %.2f MB", heapGrowthMB)

	assert.Less(t, heapGrowthMB, 20.0, "Workspace scan memory grew too much")
}

// generateCode creates filler code of approximately n lines
func generateCode(lines int) string {
	var result string
	for i := 0; i < lines; i++ {
		result += "// This is a comment line for padding\n"
	}
	return result
}
