package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestFindingToDiagnostic(t *testing.T) {
	finding := Finding{
		RuleID:      "test-rule",
		Description: "Test Secret Detected",
		StartLine:   10,
		StartColumn: 5,
		EndLine:     10,
		EndColumn:   20,
		Line:        "    secret123456789", // Line with no tabs
		Entropy:     3.5,
	}

	diag := FindingToDiagnostic(finding)

	// Both LSP and gitleaks use 0-indexed lines and columns
	// With no tabs: gitleaks column 5 (2-indexed for Line > 0) -> LSP position 3 (0-indexed)
	assert.Equal(t, uint32(10), diag.Range.Start.Line)
	assert.Equal(t, uint32(3), diag.Range.Start.Character)
	assert.Equal(t, uint32(10), diag.Range.End.Line)
	assert.Equal(t, uint32(19), diag.Range.End.Character) // EndColumn 20 (1-indexed for Line > 0) -> 19 (0-indexed)

	assert.Equal(t, protocol.DiagnosticSeverityWarning, *diag.Severity)
	assert.Equal(t, "gitleaks", *diag.Source)
	assert.Contains(t, diag.Message, "test-rule")
	assert.Contains(t, diag.Message, "Test Secret Detected")
	assert.Contains(t, diag.Message, "entropy: 3.5")
}

func TestFindingToDiagnostic_MultiLine(t *testing.T) {
	finding := Finding{
		RuleID:      "multiline-secret",
		Description: "Secret spanning multiple lines",
		StartLine:   5,
		StartColumn: 10,
		EndLine:     7,
		EndColumn:   15,
	}

	diag := FindingToDiagnostic(finding)

	assert.Equal(t, uint32(5), diag.Range.Start.Line)
	assert.Equal(t, uint32(7), diag.Range.End.Line)
}

func TestFindingToDiagnostic_NoEntropy(t *testing.T) {
	finding := Finding{
		RuleID:      "regex-match",
		Description: "Pattern matched",
		StartLine:   1,
		StartColumn: 2, // Minimum for Line > 0 is 2
		EndLine:     1,
		EndColumn:   10,
		Entropy:     0,
	}

	diag := FindingToDiagnostic(finding)

	// Should not include entropy in message when it's 0
	assert.NotContains(t, diag.Message, "entropy")
}

func TestFindingsToDiagnostics(t *testing.T) {
	findings := []Finding{
		{
			RuleID:      "rule-1",
			Description: "First finding",
			StartLine:   1,
			StartColumn: 2,
			EndLine:     1,
			EndColumn:   10,
		},
		{
			RuleID:      "rule-2",
			Description: "Second finding",
			StartLine:   5,
			StartColumn: 5,
			EndLine:     5,
			EndColumn:   20,
		},
	}

	diagnostics := FindingsToDiagnostics(findings)

	assert.Len(t, diagnostics, 2)
	assert.Contains(t, diagnostics[0].Message, "rule-1")
	assert.Contains(t, diagnostics[1].Message, "rule-2")
}

func TestFindingsToDiagnostics_Empty(t *testing.T) {
	findings := []Finding{}
	diagnostics := FindingsToDiagnostics(findings)
	assert.Empty(t, diagnostics)
}
