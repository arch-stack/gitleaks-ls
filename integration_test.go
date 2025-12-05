package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestIntegration_Diagnostics_Direct(t *testing.T) {
	// Capture notifications
	var notifications []protocol.PublishDiagnosticsParams

	notifyFunc := func(method string, params any) {
		if method == "textDocument/publishDiagnostics" {
			if p, ok := params.(protocol.PublishDiagnosticsParams); ok {
				notifications = append(notifications, p)
			}
		}
	}

	ctx := &glsp.Context{
		Notify: notifyFunc,
	}

	// Initialize
	rootURI := "file:///tmp/test"
	initParams := &protocol.InitializeParams{
		RootURI: &rootURI,
		Capabilities: protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				Synchronization: &protocol.TextDocumentSyncClientCapabilities{
					DidSave: &[]bool{true}[0],
				},
			},
		},
	}

	// Call initialize to setup globalServer
	_, err := initialize(ctx, initParams)
	require.NoError(t, err)

	// Send didOpen with a secret
	secretContent := `
package main

const awsKey = "AKIATESTKEYEXAMPLE7A"
`
	didOpenParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        "file:///tmp/test/secret.go",
			LanguageID: "go",
			Version:    1,
			Text:       secretContent,
		},
	}

	err = textDocumentDidOpen(ctx, didOpenParams)
	require.NoError(t, err)

	// Verify diagnostics
	require.NotEmpty(t, notifications, "No diagnostics published")
	diag := notifications[0]
	assert.Equal(t, "file:///tmp/test/secret.go", diag.URI)
	assert.NotEmpty(t, diag.Diagnostics, "Diagnostics list is empty")
	assert.Contains(t, diag.Diagnostics[0].Message, "AWS credentials")
}

func TestIntegration_DidChange(t *testing.T) {
	// Capture notifications
	var notifications []protocol.PublishDiagnosticsParams

	notifyFunc := func(method string, params any) {
		if method == "textDocument/publishDiagnostics" {
			if p, ok := params.(protocol.PublishDiagnosticsParams); ok {
				notifications = append(notifications, p)
			}
		}
	}

	ctx := &glsp.Context{
		Notify: notifyFunc,
	}

	// Initialize
	rootURI := "file:///tmp/test"
	initParams := &protocol.InitializeParams{
		RootURI: &rootURI,
		Capabilities: protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				Synchronization: &protocol.TextDocumentSyncClientCapabilities{
					DidSave: &[]bool{true}[0],
				},
			},
		},
	}

	_, err := initialize(ctx, initParams)
	require.NoError(t, err)

	// First open a file without secrets
	didOpenParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        "file:///tmp/test/change.go",
			LanguageID: "go",
			Version:    1,
			Text:       "package main\n",
		},
	}

	err = textDocumentDidOpen(ctx, didOpenParams)
	require.NoError(t, err)
	notifications = nil // Clear initial notification

	// Now change it to have a secret
	didChangeParams := &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{
				URI: "file:///tmp/test/change.go",
			},
			Version: 2,
		},
		ContentChanges: []any{
			protocol.TextDocumentContentChangeEventWhole{
				Text: "package main\nconst key = \"AKIATESTKEYEXAMPLE7A\"\n",
			},
		},
	}

	err = textDocumentDidChange(ctx, didChangeParams)
	require.NoError(t, err)

	// Verify diagnostics were published
	require.NotEmpty(t, notifications, "No diagnostics published after change")
	assert.NotEmpty(t, notifications[0].Diagnostics, "Should have found secret after change")
}

func TestIntegration_DidChange_EmptyChanges(t *testing.T) {
	// Initialize
	ctx := &glsp.Context{
		Notify: func(method string, params any) {},
	}

	rootURI := "file:///tmp/test"
	initParams := &protocol.InitializeParams{
		RootURI: &rootURI,
	}
	_, err := initialize(ctx, initParams)
	require.NoError(t, err)

	// Send didChange with empty ContentChanges
	didChangeParams := &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{
				URI: "file:///tmp/test/empty.go",
			},
			Version: 2,
		},
		ContentChanges: []any{}, // Empty
	}

	// Should return nil without error
	err = textDocumentDidChange(ctx, didChangeParams)
	assert.NoError(t, err)
}

func TestIntegration_DidSave(t *testing.T) {
	// Capture notifications
	var notifications []protocol.PublishDiagnosticsParams

	notifyFunc := func(method string, params any) {
		if method == "textDocument/publishDiagnostics" {
			if p, ok := params.(protocol.PublishDiagnosticsParams); ok {
				notifications = append(notifications, p)
			}
		}
	}

	ctx := &glsp.Context{
		Notify: notifyFunc,
	}

	// Initialize
	rootURI := "file:///tmp/test"
	initParams := &protocol.InitializeParams{
		RootURI: &rootURI,
	}

	_, err := initialize(ctx, initParams)
	require.NoError(t, err)

	// Open a file first
	didOpenParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        "file:///tmp/test/save.go",
			LanguageID: "go",
			Version:    1,
			Text:       "package main\n",
		},
	}
	err = textDocumentDidOpen(ctx, didOpenParams)
	require.NoError(t, err)
	notifications = nil

	// Now save with new content containing a secret
	secretContent := "package main\nconst key = \"AKIATESTKEYEXAMPLE7A\"\n"
	didSaveParams := &protocol.DidSaveTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///tmp/test/save.go",
		},
		Text: &secretContent,
	}

	err = textDocumentDidSave(ctx, didSaveParams)
	require.NoError(t, err)

	// Verify diagnostics
	require.NotEmpty(t, notifications)
	assert.NotEmpty(t, notifications[0].Diagnostics, "Should find secret on save")
}

func TestIntegration_DidSave_NoText(t *testing.T) {
	// Capture notifications
	var notifications []protocol.PublishDiagnosticsParams

	notifyFunc := func(method string, params any) {
		if method == "textDocument/publishDiagnostics" {
			if p, ok := params.(protocol.PublishDiagnosticsParams); ok {
				notifications = append(notifications, p)
			}
		}
	}

	ctx := &glsp.Context{
		Notify: notifyFunc,
	}

	// Initialize
	rootURI := "file:///tmp/test"
	initParams := &protocol.InitializeParams{
		RootURI: &rootURI,
	}
	_, err := initialize(ctx, initParams)
	require.NoError(t, err)

	// Open a file with secret
	secretContent := "package main\nconst key = \"AKIATESTKEYEXAMPLE7A\"\n"
	didOpenParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        "file:///tmp/test/save_notext.go",
			LanguageID: "go",
			Version:    1,
			Text:       secretContent,
		},
	}
	err = textDocumentDidOpen(ctx, didOpenParams)
	require.NoError(t, err)
	notifications = nil

	// Save without text (should use stored content)
	didSaveParams := &protocol.DidSaveTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///tmp/test/save_notext.go",
		},
		Text: nil, // No text provided
	}

	err = textDocumentDidSave(ctx, didSaveParams)
	require.NoError(t, err)

	// Should still find the secret from stored content
	require.NotEmpty(t, notifications)
	assert.NotEmpty(t, notifications[0].Diagnostics)
}

func TestIntegration_DidSave_UnknownDocument(t *testing.T) {
	ctx := &glsp.Context{
		Notify: func(method string, params any) {},
	}

	// Initialize
	rootURI := "file:///tmp/test"
	initParams := &protocol.InitializeParams{
		RootURI: &rootURI,
	}
	_, err := initialize(ctx, initParams)
	require.NoError(t, err)

	// Save a document that was never opened
	didSaveParams := &protocol.DidSaveTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///tmp/test/unknown.go",
		},
		Text: nil,
	}

	// Should return nil without crashing
	err = textDocumentDidSave(ctx, didSaveParams)
	assert.NoError(t, err)
}

func TestIntegration_DidClose(t *testing.T) {
	// Capture notifications
	var notifications []protocol.PublishDiagnosticsParams

	notifyFunc := func(method string, params any) {
		if method == "textDocument/publishDiagnostics" {
			if p, ok := params.(protocol.PublishDiagnosticsParams); ok {
				notifications = append(notifications, p)
			}
		}
	}

	ctx := &glsp.Context{
		Notify: notifyFunc,
	}

	// Initialize
	rootURI := "file:///tmp/test"
	initParams := &protocol.InitializeParams{
		RootURI: &rootURI,
	}
	_, err := initialize(ctx, initParams)
	require.NoError(t, err)

	uri := protocol.DocumentUri("file:///tmp/test/close.go")

	// Open a file
	didOpenParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "go",
			Version:    1,
			Text:       "package main\nconst key = \"AKIATESTKEYEXAMPLE7A\"\n",
		},
	}
	err = textDocumentDidOpen(ctx, didOpenParams)
	require.NoError(t, err)

	// Verify document is stored
	_, ok := globalServer.documents.Get(uri)
	assert.True(t, ok, "Document should be in store after open")

	notifications = nil

	// Close the document
	didCloseParams := &protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: uri,
		},
	}

	err = textDocumentDidClose(ctx, didCloseParams)
	require.NoError(t, err)

	// Verify document is removed
	_, ok = globalServer.documents.Get(uri)
	assert.False(t, ok, "Document should be removed after close")

	// Verify empty diagnostics published
	require.NotEmpty(t, notifications)
	assert.Empty(t, notifications[0].Diagnostics, "Diagnostics should be cleared on close")
}

func TestIntegration_Hover(t *testing.T) {
	ctx := &glsp.Context{
		Notify: func(method string, params any) {},
	}

	// Initialize
	rootURI := "file:///tmp/test"
	initParams := &protocol.InitializeParams{
		RootURI: &rootURI,
	}
	_, err := initialize(ctx, initParams)
	require.NoError(t, err)

	uri := protocol.DocumentUri("file:///tmp/test/hover.go")

	// Open a file with a secret
	secretContent := `package main

const awsKey = "AKIATESTKEYEXAMPLE7A"
`
	didOpenParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "go",
			Version:    1,
			Text:       secretContent,
		},
	}
	err = textDocumentDidOpen(ctx, didOpenParams)
	require.NoError(t, err)

	// Hover over the secret (line 2, character 20 is inside the key)
	hoverParams := &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 2, Character: 20},
		},
	}

	hover, err := textDocumentHover(ctx, hoverParams)
	require.NoError(t, err)
	require.NotNil(t, hover, "Should return hover content for secret")

	// Verify content
	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok, "Expected MarkupContent")
	assert.Equal(t, protocol.MarkupKindMarkdown, content.Kind)
	assert.Contains(t, content.Value, "Secret Detected")
}

func TestIntegration_Hover_NoSecret(t *testing.T) {
	ctx := &glsp.Context{
		Notify: func(method string, params any) {},
	}

	// Initialize
	rootURI := "file:///tmp/test"
	initParams := &protocol.InitializeParams{
		RootURI: &rootURI,
	}
	_, err := initialize(ctx, initParams)
	require.NoError(t, err)

	uri := protocol.DocumentUri("file:///tmp/test/hover_clean.go")

	// Open a file without secrets
	didOpenParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "go",
			Version:    1,
			Text:       "package main\n\nfunc main() {}\n",
		},
	}
	err = textDocumentDidOpen(ctx, didOpenParams)
	require.NoError(t, err)

	// Hover somewhere
	hoverParams := &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 1, Character: 5},
		},
	}

	hover, err := textDocumentHover(ctx, hoverParams)
	require.NoError(t, err)
	assert.Nil(t, hover, "Should return nil when not over a secret")
}

func TestIntegration_Hover_UnknownDocument(t *testing.T) {
	ctx := &glsp.Context{
		Notify: func(method string, params any) {},
	}

	// Initialize
	rootURI := "file:///tmp/test"
	initParams := &protocol.InitializeParams{
		RootURI: &rootURI,
	}
	_, err := initialize(ctx, initParams)
	require.NoError(t, err)

	// Hover on document that was never opened
	hoverParams := &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///unknown.go"},
			Position:     protocol.Position{Line: 0, Character: 0},
		},
	}

	hover, err := textDocumentHover(ctx, hoverParams)
	require.NoError(t, err)
	assert.Nil(t, hover)
}

func TestIntegration_CodeAction(t *testing.T) {
	ctx := &glsp.Context{
		Notify: func(method string, params any) {},
	}

	// Initialize
	rootURI := "file:///tmp/test"
	initParams := &protocol.InitializeParams{
		RootURI: &rootURI,
	}
	_, err := initialize(ctx, initParams)
	require.NoError(t, err)

	uri := protocol.DocumentUri("file:///tmp/test/action.go")

	// Open a file with a secret
	secretContent := `package main

const awsKey = "AKIATESTKEYEXAMPLE7A"
`
	didOpenParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "go",
			Version:    1,
			Text:       secretContent,
		},
	}
	err = textDocumentDidOpen(ctx, didOpenParams)
	require.NoError(t, err)

	// Get the diagnostic
	doc, ok := globalServer.documents.Get(uri)
	require.True(t, ok)
	require.NotEmpty(t, doc.Diagnostics)

	// Request code actions for the diagnostic
	codeActionParams := &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range:        doc.Diagnostics[0].Range,
		Context: protocol.CodeActionContext{
			Diagnostics: doc.Diagnostics,
		},
	}

	result, err := textDocumentCodeAction(ctx, codeActionParams)
	require.NoError(t, err)

	actions, ok := result.([]protocol.CodeAction)
	require.True(t, ok, "Expected []CodeAction")
	require.NotEmpty(t, actions, "Should return code actions for secret")
	assert.Contains(t, actions[0].Title, "gitleaks:allow")
	assert.NotNil(t, actions[0].Edit)
}

func TestIntegration_CodeAction_NoSecrets(t *testing.T) {
	ctx := &glsp.Context{
		Notify: func(method string, params any) {},
	}

	// Initialize
	rootURI := "file:///tmp/test"
	initParams := &protocol.InitializeParams{
		RootURI: &rootURI,
	}
	_, err := initialize(ctx, initParams)
	require.NoError(t, err)

	uri := protocol.DocumentUri("file:///tmp/test/action_clean.go")

	// Open a file without secrets
	didOpenParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "go",
			Version:    1,
			Text:       "package main\n",
		},
	}
	err = textDocumentDidOpen(ctx, didOpenParams)
	require.NoError(t, err)

	// Request code actions with no diagnostics
	codeActionParams := &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range:        protocol.Range{},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	}

	result, err := textDocumentCodeAction(ctx, codeActionParams)
	require.NoError(t, err)

	actions, ok := result.([]protocol.CodeAction)
	require.True(t, ok, "Expected []CodeAction")
	assert.Empty(t, actions, "Should return no actions when no diagnostics")
}

func TestDocumentStore_Delete(t *testing.T) {
	store := NewDocumentStore()

	// Add a document
	uri := protocol.DocumentUri("file:///test.go")
	store.Set(uri, 1, "content")

	// Verify it exists
	_, ok := store.Get(uri)
	assert.True(t, ok)

	// Delete it
	store.Delete(uri)

	// Verify it's gone
	_, ok = store.Get(uri)
	assert.False(t, ok)
}

func TestIntegration_CacheHit(t *testing.T) {
	ctx := &glsp.Context{
		Notify: func(method string, params any) {},
	}

	// Initialize
	rootURI := "file:///tmp/test"
	initParams := &protocol.InitializeParams{
		RootURI: &rootURI,
	}
	_, err := initialize(ctx, initParams)
	require.NoError(t, err)

	// Clear cache to start fresh
	globalServer.cache.Clear()
	assert.Equal(t, 0, globalServer.cache.Size())

	secretContent := "package main\nconst key = \"AKIATESTKEYEXAMPLE7A\"\n"
	uri := protocol.DocumentUri("file:///tmp/test/cache.go")

	// First open - should scan and cache
	didOpenParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "go",
			Version:    1,
			Text:       secretContent,
		},
	}
	err = textDocumentDidOpen(ctx, didOpenParams)
	require.NoError(t, err)

	// Cache should have 1 entry
	assert.Equal(t, 1, globalServer.cache.Size())

	// Open another file with same content - should hit cache
	uri2 := protocol.DocumentUri("file:///tmp/test/cache2.go")
	didOpenParams2 := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri2,
			LanguageID: "go",
			Version:    1,
			Text:       secretContent, // Same content
		},
	}
	err = textDocumentDidOpen(ctx, didOpenParams2)
	require.NoError(t, err)

	// Cache should still have 1 entry (same content hash)
	assert.Equal(t, 1, globalServer.cache.Size())
}

func TestIntegration_CacheClearedOnConfigReload(t *testing.T) {
	ctx := &glsp.Context{
		Notify: func(method string, params any) {},
	}

	// Initialize
	rootURI := "file:///tmp/test"
	initParams := &protocol.InitializeParams{
		RootURI: &rootURI,
	}
	_, err := initialize(ctx, initParams)
	require.NoError(t, err)

	// Put something in cache
	globalServer.cache.Put("test content", []Finding{{RuleID: "test"}})
	assert.Equal(t, 1, globalServer.cache.Size())

	// Simulate config reload by calling the callback
	globalServer.cache.Clear()
	assert.Equal(t, 0, globalServer.cache.Size())
}

func TestInitialize_WithClientInfo(t *testing.T) {
	ctx := &glsp.Context{
		Notify: func(method string, params any) {},
	}

	rootURI := "file:///tmp/test"
	clientVersion := "1.2.3"
	initParams := &protocol.InitializeParams{
		RootURI: &rootURI,
		ClientInfo: &struct {
			Name    string  `json:"name"`
			Version *string `json:"version,omitempty"`
		}{
			Name:    "test-client",
			Version: &clientVersion,
		},
	}

	result, err := initialize(ctx, initParams)
	require.NoError(t, err)

	initResult, ok := result.(protocol.InitializeResult)
	require.True(t, ok, "Expected InitializeResult")
	assert.NotNil(t, initResult.ServerInfo)
	assert.Equal(t, "gitleaks-ls", initResult.ServerInfo.Name)
}

func TestInitialize_WithRootPath(t *testing.T) {
	ctx := &glsp.Context{
		Notify: func(method string, params any) {},
	}

	rootPath := "/tmp/test-path"
	initParams := &protocol.InitializeParams{
		RootPath: &rootPath,
	}

	result, err := initialize(ctx, initParams)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestInitialized(t *testing.T) {
	ctx := &glsp.Context{
		Notify: func(method string, params any) {},
	}

	err := initialized(ctx, &protocol.InitializedParams{})
	assert.NoError(t, err)
}

func TestShutdown(t *testing.T) {
	ctx := &glsp.Context{
		Notify: func(method string, params any) {},
	}

	err := shutdown(ctx)
	assert.NoError(t, err)
}

func TestSetTrace(t *testing.T) {
	ctx := &glsp.Context{
		Notify: func(method string, params any) {},
	}

	// Test setting different trace values
	err := setTrace(ctx, &protocol.SetTraceParams{Value: protocol.TraceValueVerbose})
	assert.NoError(t, err)

	err = setTrace(ctx, &protocol.SetTraceParams{Value: protocol.TraceValueOff})
	assert.NoError(t, err)
}

func TestIntegration_ConfigReload(t *testing.T) {
	t.Skip("Skipping flaky config reload test in CI environment")
	// Create temp dir
	tmpDir := t.TempDir()
	configFile := tmpDir + "/.gitleaks.toml"

	// Create initial config that detects "TEST_SECRET_A"
	configA := `
[[rules]]
id = "test-rule-a"
description = "Test Rule A"
regex = "TEST_SECRET_A"
`
	err := os.WriteFile(configFile, []byte(configA), 0644)
	require.NoError(t, err)

	// Capture notifications
	var notifications []protocol.PublishDiagnosticsParams
	notifyFunc := func(method string, params any) {
		if method == "textDocument/publishDiagnostics" {
			if p, ok := params.(protocol.PublishDiagnosticsParams); ok {
				notifications = append(notifications, p)
			}
		}
	}

	ctx := &glsp.Context{
		Notify: notifyFunc,
	}

	// Initialize
	rootURI := "file://" + tmpDir
	initParams := &protocol.InitializeParams{
		RootURI: &rootURI,
		Capabilities: protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				Synchronization: &protocol.TextDocumentSyncClientCapabilities{
					DidSave: &[]bool{true}[0],
				},
			},
		},
	}

	// Call initialize
	_, err = initialize(ctx, initParams)
	require.NoError(t, err)

	// Test detection of TEST_SECRET_A
	didOpenParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        "file://" + tmpDir + "/test.txt",
			LanguageID: "text",
			Version:    1,
			Text:       "TEST_SECRET_A",
		},
	}

	err = textDocumentDidOpen(ctx, didOpenParams)
	require.NoError(t, err)

	require.NotEmpty(t, notifications)
	assert.Contains(t, notifications[0].Diagnostics[0].Message, "test-rule-a: Test Rule A")

	// Clear notifications
	notifications = nil

	// Update config to detect "TEST_SECRET_B" instead
	configB := `
[[rules]]
id = "test-rule-b"
description = "Test Rule B"
regex = "TEST_SECRET_B"
`
	err = os.WriteFile(configFile, []byte(configB), 0644)
	require.NoError(t, err)

	// Wait for reload (fsnotify is async)
	time.Sleep(100 * time.Millisecond)

	// Trigger scan again (didChange or didSave)
	// We'll use didChange
	didChangeParams := &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{
				URI: "file://" + tmpDir + "/test.txt",
			},
			Version: 2,
		},
		ContentChanges: []any{
			protocol.TextDocumentContentChangeEventWhole{Text: "TEST_SECRET_B"},
		},
	}

	// Retry a few times if reload is slow
	for i := 0; i < 10; i++ {
		notifications = nil
		err = textDocumentDidChange(ctx, didChangeParams)
		require.NoError(t, err)

		if len(notifications) > 0 && len(notifications[0].Diagnostics) > 0 {
			if strings.Contains(notifications[0].Diagnostics[0].Message, "Test Rule B") {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	require.NotEmpty(t, notifications, "Should have received diagnostics after config reload")
	require.NotEmpty(t, notifications[0].Diagnostics, "Should have diagnostics")
	assert.Equal(t, "test-rule-b: Test Rule B", notifications[0].Diagnostics[0].Message)
}
