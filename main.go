package main

import (
	"log/slog"
	"os"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"

	_ "github.com/tliron/commonlog/simple"
)

const lsName = "gitleaks-ls"

var (
	version = "0.1.0"
	handler protocol.Handler
)

func main() {
	// Setup logging
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create LSP handler
	handler = protocol.Handler{
		Initialize:  initialize,
		Initialized: initialized,
		Shutdown:    shutdown,
		SetTrace:    setTrace,
		// Register document handlers
		TextDocumentDidOpen:   textDocumentDidOpen,
		TextDocumentDidChange: textDocumentDidChange,
		TextDocumentDidSave:   textDocumentDidSave,
		TextDocumentDidClose:  textDocumentDidClose,
		// Register feature handlers
		TextDocumentHover:      textDocumentHover,
		TextDocumentCodeAction: textDocumentCodeAction,
		// Register command handler
		WorkspaceExecuteCommand: executeCommand,
	}

	// Create LSP server
	glspServer := server.NewServer(&handler, lsName, false)

	// Run server over stdio
	slog.Info("starting gitleaks language server", "version", version)
	if err := glspServer.RunStdio(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	capabilities := handler.CreateServerCapabilities()

	// Set text document sync capabilities
	capabilities.TextDocumentSync = protocol.TextDocumentSyncOptions{
		OpenClose: &[]bool{true}[0],
		Change:    &[]protocol.TextDocumentSyncKind{protocol.TextDocumentSyncKindFull}[0],
		Save: &protocol.SaveOptions{
			IncludeText: &[]bool{true}[0],
		},
	}

	// Enable hover support
	capabilities.HoverProvider = true

	// Enable code actions
	capabilities.CodeActionProvider = true

	// Enable execute command
	capabilities.ExecuteCommandProvider = &protocol.ExecuteCommandOptions{
		Commands: []string{"gitleaks.scanWorkspace"},
	}

	clientName := "unknown"
	clientVersion := "unknown"
	if params.ClientInfo != nil {
		clientName = params.ClientInfo.Name
		if params.ClientInfo.Version != nil {
			clientVersion = *params.ClientInfo.Version
		}
	}

	slog.Info("initialized",
		"clientName", clientName,
		"clientVersion", clientVersion)

	rootPath := ""
	if params.RootPath != nil {
		rootPath = *params.RootPath
	} else if params.RootURI != nil {
		rootPath = uriToPath(*params.RootURI)
	}

	if err := SetupServer(rootPath); err != nil {
		slog.Error("failed to setup server", "error", err)
		return nil, err
	}

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    lsName,
			Version: &version,
		},
	}, nil
}

func initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	slog.Info("client confirmed initialization")
	return nil
}

func shutdown(context *glsp.Context) error {
	slog.Info("shutting down")
	if globalServer != nil && globalServer.cancel != nil {
		globalServer.cancel()
	}
	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

func setTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}
