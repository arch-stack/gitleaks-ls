package main

import (
	"fmt"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// adjustColumn converts gitleaks column to LSP character position
// Gitleaks has an off-by-one error for lines > 0 (counting newline as a column)
// Line 0: StartColumn is 1-indexed, EndColumn is 0-indexed (exclusive)
// Line >0: StartColumn is 2-indexed, EndColumn is 1-indexed (exclusive)
func adjustColumn(col int, lineNum int, isEndColumn bool) uint32 {
	if col <= 0 {
		return 0
	}

	if lineNum == 0 {
		if isEndColumn {
			return uint32(col)
		}
		return uint32(max(0, col-1))
	}

	// For lines > 0
	if isEndColumn {
		return uint32(max(0, col-1))
	}
	return uint32(max(0, col-2))
}

// FindingsToDiagnostics converts scanner findings to LSP diagnostics
func FindingsToDiagnostics(findings []Finding) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0, len(findings))
	for _, f := range findings {
		diagnostics = append(diagnostics, FindingToDiagnostic(f))
	}
	return diagnostics
}

// FindingToDiagnostic converts a single finding to an LSP diagnostic
func FindingToDiagnostic(f Finding) protocol.Diagnostic {
	severity := GetDiagnosticSeverity()
	source := "gitleaks"
	code := protocol.IntegerOrString{Value: f.RuleID}

	// Gitleaks has inconsistent column numbering between first line and subsequent lines
	// We adjust for this to get correct 0-indexed byte positions for LSP
	startChar := adjustColumn(f.StartColumn, f.StartLine, false)
	endChar := adjustColumn(f.EndColumn, f.StartLine, true)

	return protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(f.StartLine),
				Character: startChar,
			},
			End: protocol.Position{
				Line:      uint32(f.EndLine),
				Character: endChar,
			},
		},
		Severity: &severity,
		Source:   &source,
		Message:  formatDiagnosticMessage(f),
		Code:     &code,
	}
}

// formatDiagnosticMessage creates a human-readable diagnostic message
func formatDiagnosticMessage(f Finding) string {
	msg := fmt.Sprintf("%s: %s", f.RuleID, f.Description)

	if f.Entropy > 0 {
		msg += fmt.Sprintf(" (entropy: %.1f)", f.Entropy)
	}

	return msg
}
