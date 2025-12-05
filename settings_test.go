package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestDefaultSettings(t *testing.T) {
	settings := DefaultSettings()
	assert.Equal(t, "warning", settings.DiagnosticSeverity)
}

func TestGetDiagnosticSeverity(t *testing.T) {
	tests := []struct {
		setting  string
		expected protocol.DiagnosticSeverity
	}{
		{"error", protocol.DiagnosticSeverityError},
		{"warning", protocol.DiagnosticSeverityWarning},
		{"information", protocol.DiagnosticSeverityInformation},
		{"hint", protocol.DiagnosticSeverityHint},
		{"invalid", protocol.DiagnosticSeverityWarning}, // default
		{"", protocol.DiagnosticSeverityWarning},        // default
	}

	for _, tt := range tests {
		t.Run(tt.setting, func(t *testing.T) {
			serverSettings.DiagnosticSeverity = tt.setting
			result := GetDiagnosticSeverity()
			assert.Equal(t, tt.expected, result)
		})
	}

	// Reset to default
	serverSettings = DefaultSettings()
}

func TestUpdateSettings(t *testing.T) {
	// Reset before test
	serverSettings = DefaultSettings()

	// Test updating with valid config
	config := map[string]interface{}{
		"gitleaks": map[string]interface{}{
			"diagnosticSeverity": "error",
		},
	}
	UpdateSettings(config)
	assert.Equal(t, "error", serverSettings.DiagnosticSeverity)

	// Test with nil config
	UpdateSettings(nil)
	assert.Equal(t, "error", serverSettings.DiagnosticSeverity) // unchanged

	// Test with missing gitleaks key
	UpdateSettings(map[string]interface{}{"other": "value"})
	assert.Equal(t, "error", serverSettings.DiagnosticSeverity) // unchanged

	// Reset
	serverSettings = DefaultSettings()
}
