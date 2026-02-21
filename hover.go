package main

import (
	"fmt"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func textDocumentHover(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	uri := params.TextDocument.URI
	position := params.Position

	// Get document snapshot
	doc, ok := globalServer.documents.Get(uri)
	if !ok {
		return nil, nil
	}

	// Find diagnostic at cursor position
	var finding *Finding
	for i, diag := range doc.Diagnostics {
		if positionInRange(position, diag.Range) {
			if i < len(doc.Findings) {
				finding = &doc.Findings[i]
				break
			}
		}
	}

	if finding == nil {
		return nil, nil
	}

	// Format hover content
	content := formatHoverContent(*finding)

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: content,
		},
	}, nil
}

// positionInRange checks if a position is within a range
func positionInRange(pos protocol.Position, rng protocol.Range) bool {
	// Check if position is after range start
	if pos.Line < rng.Start.Line {
		return false
	}
	if pos.Line == rng.Start.Line && pos.Character < rng.Start.Character {
		return false
	}

	// Check if position is before range end
	if pos.Line > rng.End.Line {
		return false
	}
	if pos.Line == rng.End.Line && pos.Character > rng.End.Character {
		return false
	}

	return true
}

// formatHoverContent creates markdown documentation for a finding
func formatHoverContent(f Finding) string {
	var sb strings.Builder

	// Title
	sb.WriteString(fmt.Sprintf("# 🔐 Secret Detected: %s\n\n", f.RuleID))

	// Description
	sb.WriteString(fmt.Sprintf("**Description**: %s\n\n", f.Description))

	// Details section
	sb.WriteString("## Details\n\n")
	sb.WriteString(fmt.Sprintf("- **Rule ID**: `%s`\n", f.RuleID))
	sb.WriteString(fmt.Sprintf("- **Location**: Line %d, Column %d-%d\n", f.StartLine, f.StartColumn, f.EndColumn))

	if f.Entropy > 0 {
		sb.WriteString(fmt.Sprintf("- **Entropy**: %.2f (randomness score)\n", f.Entropy))
	}

	sb.WriteString(fmt.Sprintf("- **Fingerprint**: `%s`\n", f.Fingerprint))
	sb.WriteString("\n")

	// Matched content (truncated if too long)
	if len(f.Match) > 0 {
		match := f.Match
		if len(match) > 100 {
			match = match[:100] + "..."
		}
		sb.WriteString("**Matched Content**:\n")
		sb.WriteString(fmt.Sprintf("```\n%s\n```\n\n", match))
	}

	// Recommendations
	sb.WriteString("## 🛡️ Recommendations\n\n")
	sb.WriteString("1. **Remove the secret** from the code\n")
	sb.WriteString("2. **Use environment variables** or secret management tools\n")
	sb.WriteString("3. **Rotate the credential** if already committed\n")
	sb.WriteString("4. **Add to `.gitleaksignore`** if this is a false positive\n\n")

	// How to suppress
	sb.WriteString("## 🔕 How to Ignore\n\n")
	sb.WriteString("If this is a false positive, add a comment on the line above:\n")
	sb.WriteString("```go\n")
	sb.WriteString("// gitleaks:allow\n")
	sb.WriteString("```\n\n")
	sb.WriteString("Or add the fingerprint to `.gitleaksignore`:\n")
	sb.WriteString(fmt.Sprintf("```\n%s\n```\n", f.Fingerprint))

	return sb.String()
}
