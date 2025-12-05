package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Server holds the language server state
type Server struct {
	scanner   *Scanner
	documents *DocumentStore
	config    *Config
	cache     *Cache
}

// DocumentStore tracks open documents and their diagnostics
type DocumentStore struct {
	mu        sync.RWMutex
	documents map[protocol.DocumentUri]*Document
}

// Document represents an open file
type Document struct {
	URI         protocol.DocumentUri
	Version     int32
	Content     string
	Diagnostics []protocol.Diagnostic
	Findings    []Finding // Store findings for hover support
}

// NewDocumentStore creates a new document store
func NewDocumentStore() *DocumentStore {
	return &DocumentStore{
		documents: make(map[protocol.DocumentUri]*Document),
	}
}

// Set stores or updates a document
func (ds *DocumentStore) Set(uri protocol.DocumentUri, version int32, content string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.documents[uri] = &Document{
		URI:     uri,
		Version: version,
		Content: content,
	}
}

// Get retrieves a document
func (ds *DocumentStore) Get(uri protocol.DocumentUri) (*Document, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	doc, ok := ds.documents[uri]
	return doc, ok
}

// Delete removes a document
func (ds *DocumentStore) Delete(uri protocol.DocumentUri) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	delete(ds.documents, uri)
}

// Global server instance
var globalServer *Server

func SetupServer(rootPath string) error {
	cache := NewCache()

	// Check for .gitleaksignore file
	ignoreFilePath := findIgnoreFile(rootPath)

	cfg, err := NewConfig(rootPath, func() {
		slog.Info("reloading configuration, clearing cache")
		if globalServer != nil {
			if globalServer.config != nil {
				// Recreate scanner with ignore file on reload
				ignoreFile := findIgnoreFile(rootPath)
				newScanner := NewScannerWithIgnore(globalServer.config.GetConfig(), ignoreFile)
				globalServer.scanner = newScanner
			}
			// Clear cache on config reload
			globalServer.cache.Clear()
		}
	})
	if err != nil {
		return err
	}

	scanner := NewScannerWithIgnore(cfg.GetConfig(), ignoreFilePath)

	globalServer = &Server{
		scanner:   scanner,
		documents: NewDocumentStore(),
		config:    cfg,
		cache:     cache,
	}

	// Start watching config file
	go func() {
		if err := cfg.Watch(context.Background()); err != nil {
			slog.Error("failed to watch config", "error", err)
		}
	}()

	// Start watching ignore file if it exists
	if ignoreFilePath != "" {
		go watchIgnoreFile(rootPath, ignoreFilePath)
	}

	return nil
}

// findIgnoreFile looks for .gitleaksignore in workspace root
func findIgnoreFile(rootPath string) string {
	if rootPath == "" {
		return ""
	}
	ignoreFile := filepath.Join(rootPath, ".gitleaksignore")
	if _, err := os.Stat(ignoreFile); err == nil {
		return ignoreFile
	}
	return ""
}

// watchIgnoreFile watches .gitleaksignore for changes
func watchIgnoreFile(rootPath, ignoreFilePath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("failed to create ignore file watcher", "error", err)
		return
	}
	defer watcher.Close()

	// Watch the directory containing the ignore file
	dir := filepath.Dir(ignoreFilePath)
	if err := watcher.Add(dir); err != nil {
		slog.Error("failed to watch directory for ignore file", "error", err)
		return
	}

	slog.Info("watching .gitleaksignore for changes", "path", ignoreFilePath)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// Check if it's the ignore file that changed
			if filepath.Base(event.Name) == ".gitleaksignore" {
				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					slog.Info("reloading .gitleaksignore")
					if globalServer != nil && globalServer.config != nil {
						ignoreFile := findIgnoreFile(rootPath)
						newScanner := NewScannerWithIgnore(globalServer.config.GetConfig(), ignoreFile)
						globalServer.scanner = newScanner
						globalServer.cache.Clear()
					}
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			slog.Error("ignore file watcher error", "error", err)
		}
	}
}

func textDocumentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	uri := params.TextDocument.URI
	content := params.TextDocument.Text
	version := params.TextDocument.Version

	slog.Debug("document opened", "uri", uri)

	// Store document
	globalServer.documents.Set(uri, version, content)

	// Scan and publish diagnostics
	return scanAndPublish(context, uri, content)
}

func textDocumentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	uri := params.TextDocument.URI

	// We use Full sync, so there's only one change with the full content
	if len(params.ContentChanges) == 0 {
		return nil
	}

	var content string
	switch change := params.ContentChanges[0].(type) {
	case protocol.TextDocumentContentChangeEvent:
		content = change.Text
	case protocol.TextDocumentContentChangeEventWhole:
		content = change.Text
	default:
		slog.Error("unexpected content change type", "type", fmt.Sprintf("%T", params.ContentChanges[0]))
		return nil
	}

	version := params.TextDocument.Version

	slog.Debug("document changed", "uri", uri)

	// Update document
	globalServer.documents.Set(uri, version, content)

	// Scan on change to provide immediate feedback
	return scanAndPublish(context, uri, content)
}

func textDocumentDidSave(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	uri := params.TextDocument.URI

	slog.Debug("document saved", "uri", uri)

	// Get content
	var content string
	if params.Text != nil {
		content = *params.Text
	} else {
		// Fallback to stored content
		doc, ok := globalServer.documents.Get(uri)
		if !ok {
			slog.Warn("document not found in store", "uri", uri)
			return nil
		}
		content = doc.Content
	}

	// Scan and publish diagnostics
	return scanAndPublish(context, uri, content)
}

func textDocumentDidClose(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	uri := params.TextDocument.URI

	slog.Debug("document closed", "uri", uri)

	// Remove document from store
	globalServer.documents.Delete(uri)

	// Clear diagnostics
	context.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: []protocol.Diagnostic{},
	})

	return nil
}

// scanAndPublish scans content and publishes diagnostics
func scanAndPublish(glspContext *glsp.Context, uri protocol.DocumentUri, content string) error {
	var findings []Finding
	var err error
	cacheHit := false

	// Check cache first
	if cached, ok := globalServer.cache.Get(content); ok {
		findings = cached
		cacheHit = true
	} else {
		// Scan for secrets
		ctx := context.Background()
		findings, err = globalServer.scanner.ScanContent(ctx, uri, content)
		if err != nil {
			slog.Error("scan failed", "uri", uri, "error", err)
			return err
		}
		// Store in cache
		globalServer.cache.Put(content, findings)
	}

	// Convert to diagnostics
	diagnostics := FindingsToDiagnostics(findings)

	// Store findings with diagnostics for hover support
	doc, ok := globalServer.documents.Get(uri)
	if ok {
		doc.Diagnostics = diagnostics
		doc.Findings = findings
	}

	slog.Debug("scan complete",
		"uri", uri,
		"findings", len(findings),
		"cacheHit", cacheHit)

	// Publish diagnostics
	glspContext.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})

	return nil
}
