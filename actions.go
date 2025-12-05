package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// commentStyle represents comment syntax for a language
type commentStyle struct {
	prefix string
	suffix string // empty if single-line comment
}

func textDocumentCodeAction(context *glsp.Context, params *protocol.CodeActionParams) (any, error) {
	uri := params.TextDocument.URI

	var actions []protocol.CodeAction

	// Check each diagnostic in the range
	for _, diag := range params.Context.Diagnostics {
		if diag.Source != nil && *diag.Source == "gitleaks" {
			// Get document content to determine indentation
			doc, ok := globalServer.documents.Get(uri)
			if !ok {
				continue
			}

			// Add action to ignore this finding
			actions = append(actions, createIgnoreAction(uri, diag, doc.Content))
		}
	}

	return actions, nil
}

// createIgnoreAction creates a code action to add gitleaks:allow comment
func createIgnoreAction(uri protocol.DocumentUri, diag protocol.Diagnostic, content string) protocol.CodeAction {
	// Calculate the line of the diagnostic
	line := diag.Range.Start.Line

	// Get the content of the line
	lines := strings.Split(content, "\n")
	var lineContent string
	if int(line) < len(lines) {
		lineContent = lines[line]
	}

	// Get comment syntax for the file type
	style := getCommentSyntax(uri)

	// Create the comment to append
	var comment string
	if style.suffix == "" {
		// Single-line comment
		comment = fmt.Sprintf(" %s gitleaks:allow", style.prefix)
	} else {
		// Block comment
		comment = fmt.Sprintf(" %s gitleaks:allow %s", style.prefix, style.suffix)
	}

	// Create text edit to append comment at end of line
	edit := protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: line, Character: uint32(len(lineContent))},
			End:   protocol.Position{Line: line, Character: uint32(len(lineContent))},
		},
		NewText: comment,
	}

	changes := make(map[protocol.DocumentUri][]protocol.TextEdit)
	changes[uri] = []protocol.TextEdit{edit}

	title := "Ignore this secret (add gitleaks:allow comment)"

	kind := protocol.CodeActionKindQuickFix

	return protocol.CodeAction{
		Title: title,
		Kind:  &kind,
		Edit: &protocol.WorkspaceEdit{
			Changes: changes,
		},
		Diagnostics: []protocol.Diagnostic{diag},
	}
}

// getCommentSyntax returns the appropriate comment syntax for a file
func getCommentSyntax(filename string) commentStyle {
	// Extract extension
	ext := strings.ToLower(filepath.Ext(filename))

	// Map extensions to comment styles
	switch ext {
	// C-style comments: //
	case ".go", ".rs", ".java", ".js", ".ts", ".jsx", ".tsx", ".c", ".cpp", ".cc", ".h", ".hpp",
		".cs", ".swift", ".kt", ".scala", ".dart", ".php":
		return commentStyle{prefix: "//"}

	// Hash comments: #
	case ".py", ".rb", ".sh", ".bash", ".zsh", ".fish", ".pl", ".yaml", ".yml", ".toml",
		".conf", ".cfg", ".ini", ".env", ".r", ".jl":
		return commentStyle{prefix: "#"}

	// Lua/SQL comments: --
	case ".lua", ".sql", ".hs", ".elm":
		return commentStyle{prefix: "--"}

	// Lisp-style: ;
	case ".lisp", ".el", ".clj", ".scm":
		return commentStyle{prefix: ";"}

	// Vim script: "
	case ".vim":
		return commentStyle{prefix: "\""}

	// HTML/XML comments: <!-- -->
	case ".html", ".htm", ".xml", ".svg":
		return commentStyle{prefix: "<!--", suffix: "-->"}

	// CSS/C block comments: /* */
	case ".css", ".scss", ".sass", ".less":
		return commentStyle{prefix: "/*", suffix: "*/"}

	// Default to // for unknown types
	default:
		return commentStyle{prefix: "//"}
	}
}
