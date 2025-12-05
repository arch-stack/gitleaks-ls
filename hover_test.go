package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestPositionInRange(t *testing.T) {
	rng := protocol.Range{
		Start: protocol.Position{Line: 5, Character: 10},
		End:   protocol.Position{Line: 5, Character: 30},
	}

	tests := []struct {
		name     string
		position protocol.Position
		expected bool
	}{
		{
			name:     "position before range",
			position: protocol.Position{Line: 5, Character: 5},
			expected: false,
		},
		{
			name:     "position at start",
			position: protocol.Position{Line: 5, Character: 10},
			expected: true,
		},
		{
			name:     "position in middle",
			position: protocol.Position{Line: 5, Character: 20},
			expected: true,
		},
		{
			name:     "position at end",
			position: protocol.Position{Line: 5, Character: 30},
			expected: true,
		},
		{
			name:     "position after range",
			position: protocol.Position{Line: 5, Character: 35},
			expected: false,
		},
		{
			name:     "position on different line before",
			position: protocol.Position{Line: 4, Character: 20},
			expected: false,
		},
		{
			name:     "position on different line after",
			position: protocol.Position{Line: 6, Character: 20},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := positionInRange(tt.position, rng)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPositionInRange_MultiLine(t *testing.T) {
	rng := protocol.Range{
		Start: protocol.Position{Line: 5, Character: 10},
		End:   protocol.Position{Line: 7, Character: 20},
	}

	tests := []struct {
		name     string
		position protocol.Position
		expected bool
	}{
		{
			name:     "on first line",
			position: protocol.Position{Line: 5, Character: 15},
			expected: true,
		},
		{
			name:     "on middle line",
			position: protocol.Position{Line: 6, Character: 0},
			expected: true,
		},
		{
			name:     "on last line",
			position: protocol.Position{Line: 7, Character: 10},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := positionInRange(tt.position, rng)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatHoverContent(t *testing.T) {
	finding := Finding{
		RuleID:      "aws-access-token",
		Description: "AWS Access Key detected",
		Match:       "AKIATESTKEYEXAMPLE7A",
		Secret:      "AKIATESTKEYEXAMPLE7A",
		StartLine:   10,
		StartColumn: 15,
		EndColumn:   35,
		Entropy:     3.5,
		Fingerprint: "abc123def456",
	}

	content := formatHoverContent(finding)

	// Verify key sections are present
	assert.Contains(t, content, "# 🔐 Secret Detected")
	assert.Contains(t, content, "aws-access-token")
	assert.Contains(t, content, "AWS Access Key detected")
	assert.Contains(t, content, "Line 10")
	assert.Contains(t, content, "Entropy")
	assert.Contains(t, content, "3.50")
	assert.Contains(t, content, "abc123def456")
	assert.Contains(t, content, "AKIATESTKEYEXAMPLE7A")
	assert.Contains(t, content, "gitleaks:allow")
	assert.Contains(t, content, "Recommendations")
}

func TestFormatHoverContent_NoEntropy(t *testing.T) {
	finding := Finding{
		RuleID:      "test-rule",
		Description: "Test pattern",
		Match:       "test-secret",
		StartLine:   1,
		StartColumn: 0,
		EndColumn:   10,
		Entropy:     0, // No entropy
		Fingerprint: "xyz789",
	}

	content := formatHoverContent(finding)

	// Should not mention entropy when it's 0
	assert.NotContains(t, content, "Entropy:")
}

func TestFormatHoverContent_LongMatch(t *testing.T) {
	finding := Finding{
		RuleID:      "long-secret",
		Description: "Very long secret",
		Match:       string(make([]byte, 200)), // 200 byte match
		StartLine:   1,
		StartColumn: 0,
		EndColumn:   200,
		Fingerprint: "long123",
	}

	content := formatHoverContent(finding)

	// Should truncate long matches
	assert.Contains(t, content, "...")
}
