package main

import protocol "github.com/tliron/glsp/protocol_3_16"

// Settings holds user-configurable options for the language server
type Settings struct {
	// DiagnosticSeverity controls the severity level for detected secrets
	// Valid values: "error", "warning", "information", "hint"
	// Default: "warning"
	DiagnosticSeverity string `json:"diagnosticSeverity"`
}

// DefaultSettings returns the default configuration
func DefaultSettings() *Settings {
	return &Settings{
		DiagnosticSeverity: "warning",
	}
}

// Global settings instance
var serverSettings = DefaultSettings()

// GetDiagnosticSeverity returns the LSP severity based on settings
func GetDiagnosticSeverity() protocol.DiagnosticSeverity {
	switch serverSettings.DiagnosticSeverity {
	case "error":
		return protocol.DiagnosticSeverityError
	case "warning":
		return protocol.DiagnosticSeverityWarning
	case "information":
		return protocol.DiagnosticSeverityInformation
	case "hint":
		return protocol.DiagnosticSeverityHint
	default:
		return protocol.DiagnosticSeverityWarning
	}
}

// UpdateSettings updates server settings from client configuration
func UpdateSettings(config map[string]interface{}) {
	if config == nil {
		return
	}

	if gitleaks, ok := config["gitleaks"].(map[string]interface{}); ok {
		if severity, ok := gitleaks["diagnosticSeverity"].(string); ok {
			serverSettings.DiagnosticSeverity = severity
		}
	}
}
