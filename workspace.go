package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/h2non/filetype"
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// ProgressReporter handles LSP progress notifications
type ProgressReporter struct {
	ctx   *glsp.Context
	token protocol.ProgressToken
}

// NewProgressReporter creates a progress reporter with a unique token
func NewProgressReporter(ctx *glsp.Context, title string) *ProgressReporter {
	token := protocol.ProgressToken{Value: "gitleaks-scan"}

	// Send begin notification
	ctx.Notify(protocol.MethodProgress, protocol.ProgressParams{
		Token: token,
		Value: protocol.WorkDoneProgressBegin{
			Kind:  "begin",
			Title: title,
		},
	})

	return &ProgressReporter{ctx: ctx, token: token}
}

// Report sends a progress update
func (p *ProgressReporter) Report(message string, percentage uint32) {
	p.ctx.Notify(protocol.MethodProgress, protocol.ProgressParams{
		Token: p.token,
		Value: protocol.WorkDoneProgressReport{
			Kind:       "report",
			Message:    &message,
			Percentage: &percentage,
		},
	})
}

// End sends the completion notification
func (p *ProgressReporter) End(message string) {
	p.ctx.Notify(protocol.MethodProgress, protocol.ProgressParams{
		Token: p.token,
		Value: protocol.WorkDoneProgressEnd{
			Kind:    "end",
			Message: &message,
		},
	})
}

// WorkspaceScanResult contains the results of a workspace scan
type WorkspaceScanResult struct {
	TotalFiles    int
	ScannedFiles  int
	SkippedFiles  int
	TotalFindings int
	Findings      map[string][]Finding // URI -> findings
}

// ScanWorkspace scans all files in the workspace
func (s *Server) ScanWorkspace(ctx context.Context, rootPath string, progress *ProgressReporter) (*WorkspaceScanResult, error) {
	if rootPath == "" {
		return nil, nil
	}

	// Collect files to scan
	files, err := collectFiles(rootPath)
	if err != nil {
		return nil, fmt.Errorf("collecting files: %w", err)
	}

	slog.Info("starting workspace scan",
		"rootPath", rootPath,
		"files", len(files))

	result := &WorkspaceScanResult{
		TotalFiles: len(files),
		Findings:   make(map[string][]Finding),
	}

	// Use semaphore to limit concurrent scans
	const maxConcurrent = 10
	sem := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var scanned, skipped int64

	for i, file := range files {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		// Report progress
		if progress != nil && i%10 == 0 {
			pct := uint32(float64(i) / float64(len(files)) * 100)
			progress.Report(fmt.Sprintf("Scanning %d/%d files", i, len(files)), pct)
		}

		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			findings, err := s.scanFile(ctx, filePath)
			if err != nil {
				slog.Debug("error scanning file", "path", filePath, "error", err)
				atomic.AddInt64(&skipped, 1)
				return
			}

			atomic.AddInt64(&scanned, 1)

			if len(findings) > 0 {
				uri := pathToURI(filePath)
				mu.Lock()
				result.Findings[uri] = findings
				mu.Unlock()
			}
		}(file)
	}

	wg.Wait()

	result.ScannedFiles = int(scanned)
	result.SkippedFiles = int(skipped)

	// Count total findings
	for _, findings := range result.Findings {
		result.TotalFindings += len(findings)
	}

	slog.Info("workspace scan complete",
		"scanned", result.ScannedFiles,
		"skipped", result.SkippedFiles,
		"findings", result.TotalFindings)

	return result, nil
}

// scanFile reads and scans a single file
func (s *Server) scanFile(ctx context.Context, filePath string) ([]Finding, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Skip binary files detected by magic bytes
	if isBinaryContent(content) {
		return nil, nil
	}

	return s.scanner.ScanContent(ctx, filePath, string(content))
}

// collectFiles walks the directory tree and collects scannable files
func collectFiles(rootPath string) ([]string, error) {
	var files []string

	// Load gitignore patterns
	gitignore := loadGitignore(rootPath)

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		// Get relative path for pattern matching
		relPath, _ := filepath.Rel(rootPath, path)

		// Skip hidden directories
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			// Skip common non-source directories
			switch d.Name() {
			case "node_modules", "vendor", "__pycache__", "target", "build", "dist":
				return filepath.SkipDir
			}
			// Check gitignore patterns
			if gitignore != nil && gitignore.MatchesPath(relPath) {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		// Skip binary and non-text files by extension
		if isBinaryExtension(d.Name()) {
			return nil
		}

		// Check gitignore patterns
		if gitignore != nil && gitignore.MatchesPath(relPath) {
			return nil
		}

		// Check file size (skip large files)
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Size() > 1_000_000 { // 1MB
			return nil
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

// loadGitignore loads patterns from .gitignore file
func loadGitignore(rootPath string) *ignore.GitIgnore {
	gitignorePath := filepath.Join(rootPath, ".gitignore")
	gitignore, err := ignore.CompileIgnoreFile(gitignorePath)
	if err != nil {
		return nil
	}
	return gitignore
}

var binaryExts = map[string]bool{
	".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".ico": true, ".webp": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".zip": true, ".tar": true, ".gz": true, ".rar": true, ".7z": true,
	".bin": true, ".dat": true, ".db": true, ".sqlite": true,
	".pyc": true, ".pyo": true, ".class": true,
	".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
	".mp3": true, ".mp4": true, ".wav": true, ".avi": true, ".mov": true,
	".o": true, ".a": true, ".lib": true,
}

// isBinaryExtension checks if a file has a known binary extension (fast pre-filter)
func isBinaryExtension(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return binaryExts[ext]
}

// isBinaryContent checks if content is binary using magic bytes detection
func isBinaryContent(content []byte) bool {
	// Only need first 262 bytes for magic number detection
	head := content
	if len(head) > 262 {
		head = head[:262]
	}
	kind, _ := filetype.Match(head)
	// If filetype detects a known type, it's binary (images, archives, etc.)
	// Unknown types (text files) return filetype.Unknown
	return kind != filetype.Unknown
}

// PublishWorkspaceFindings publishes diagnostics for all findings from a workspace scan
func (s *Server) PublishWorkspaceFindings(ctx *glsp.Context, result *WorkspaceScanResult) {
	for uri, findings := range result.Findings {
		diagnostics := FindingsToDiagnostics(findings)
		ctx.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: diagnostics,
		})
	}
}

// ExecuteCommand handles workspace/executeCommand requests
func executeCommand(ctx *glsp.Context, params *protocol.ExecuteCommandParams) (any, error) {
	switch params.Command {
	case "gitleaks.scanWorkspace":
		return handleScanWorkspaceCommand(ctx, params)
	default:
		slog.Warn("unknown command", "command", params.Command)
		return nil, nil
	}
}

// handleScanWorkspaceCommand handles the scanWorkspace command
func handleScanWorkspaceCommand(ctx *glsp.Context, _ *protocol.ExecuteCommandParams) (any, error) {
	if globalServer == nil {
		return nil, nil
	}

	// Get workspace root from config or use current directory
	rootPath := ""
	if globalServer.config != nil {
		rootPath = globalServer.config.rootPath
	}

	// Create progress reporter
	progress := NewProgressReporter(ctx, "Scanning workspace for secrets")

	bgCtx := context.Background()
	result, err := globalServer.ScanWorkspace(bgCtx, rootPath, progress)
	if err != nil {
		progress.End("Scan failed")
		slog.Error("workspace scan failed", "error", err)
		return nil, err
	}

	// End progress with summary
	progress.End(fmt.Sprintf("Found %d secrets in %d files", result.TotalFindings, len(result.Findings)))

	// Publish diagnostics for all findings
	globalServer.PublishWorkspaceFindings(ctx, result)

	// Return summary
	return map[string]any{
		"totalFiles":        result.TotalFiles,
		"scannedFiles":      result.ScannedFiles,
		"skippedFiles":      result.SkippedFiles,
		"totalFindings":     result.TotalFindings,
		"filesWithFindings": len(result.Findings),
	}, nil
}
