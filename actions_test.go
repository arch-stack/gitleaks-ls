package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestCreateIgnoreAction_Append(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		line         uint32
		expectedText string
	}{
		{
			name:         "Simple",
			content:      "secret := \"...\"",
			line:         0,
			expectedText: " // gitleaks:allow",
		},
		{
			name:         "Indented",
			content:      "\tsecret := \"...\"",
			line:         0,
			expectedText: " // gitleaks:allow",
		},
		{
			name: "Middle Line",
			content: `func main() {
	secret := "..."
}`,
			line:         1,
			expectedText: " // gitleaks:allow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diag := protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: tt.line, Character: 0},
				},
			}

			action := createIgnoreAction("file:///test.go", diag, tt.content)

			// Check the edit text
			changes := action.Edit.Changes["file:///test.go"]
			assert.Len(t, changes, 1)

			newText := changes[0].NewText
			assert.Equal(t, tt.expectedText, newText)

			// Check position (should be end of line)
			lines := strings.Split(tt.content, "\n")
			lineLen := len(lines[tt.line])
			assert.Equal(t, uint32(lineLen), changes[0].Range.Start.Character)
		})
	}
}

func TestGetCommentSyntax(t *testing.T) {
	tests := []struct {
		filename       string
		expectedPrefix string
		expectedSuffix string
	}{
		// Single-line comments
		{"test.go", "//", ""},
		{"test.rs", "//", ""},
		{"test.java", "//", ""},
		{"test.js", "//", ""},
		{"test.ts", "//", ""},
		{"test.jsx", "//", ""},
		{"test.tsx", "//", ""},
		{"test.c", "//", ""},
		{"test.cpp", "//", ""},
		{"test.cs", "//", ""},
		{"test.swift", "//", ""},
		{"test.kt", "//", ""},
		{"test.php", "//", ""},

		// Hash comments
		{"test.py", "#", ""},
		{"test.rb", "#", ""},
		{"test.sh", "#", ""},
		{"test.bash", "#", ""},
		{"test.yaml", "#", ""},
		{"test.yml", "#", ""},
		{"test.toml", "#", ""},
		{"test.env", "#", ""},
		{"test.r", "#", ""},

		// Double dash
		{"test.lua", "--", ""},
		{"test.sql", "--", ""},
		{"test.hs", "--", ""},

		// Semicolon
		{"test.lisp", ";", ""},
		{"test.clj", ";", ""},

		// Quote
		{"test.vim", "\"", ""},

		// Block comments
		{"test.html", "<!--", "-->"},
		{"test.htm", "<!--", "-->"},
		{"test.xml", "<!--", "-->"},
		{"test.svg", "<!--", "-->"},
		{"test.css", "/*", "*/"},
		{"test.scss", "/*", "*/"},
		{"test.sass", "/*", "*/"},

		// Unknown defaults to //
		{"test.unknown", "//", ""},
		{"README.md", "//", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			style := getCommentSyntax(tt.filename)
			assert.Equal(t, tt.expectedPrefix, style.prefix, "prefix mismatch")
			assert.Equal(t, tt.expectedSuffix, style.suffix, "suffix mismatch")
		})
	}
}

func TestGetCommentSyntax_CaseInsensitive(t *testing.T) {
	// Test that extension matching is case-insensitive
	tests := []struct {
		filename string
		expected string
	}{
		{"Test.GO", "//"},
		{"Test.PY", "#"},
		{"Test.HTML", "<!--"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			style := getCommentSyntax(tt.filename)
			assert.Equal(t, tt.expected, style.prefix)
		})
	}
}

func TestGetCommentSyntax_FullPaths(t *testing.T) {
	// Test with full file paths
	tests := []struct {
		path     string
		expected string
	}{
		{"/path/to/file.go", "//"},
		{"/home/user/project/main.py", "#"},
		{"C:\\Users\\test\\file.js", "//"},
		{"./relative/path/index.html", "<!--"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			style := getCommentSyntax(tt.path)
			assert.Equal(t, tt.expected, style.prefix)
		})
	}
}
