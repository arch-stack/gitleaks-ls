package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zricethezav/gitleaks/v8/config"
	"github.com/zricethezav/gitleaks/v8/report"
)

func newTestScanner(t testing.TB) *Scanner {
	v := viper.New()
	v.SetConfigType("toml")
	require.NoError(t, v.ReadConfig(strings.NewReader(config.DefaultConfig)))

	var vc config.ViperConfig
	require.NoError(t, v.Unmarshal(&vc))

	cfg, err := vc.Translate()
	require.NoError(t, err)

	return NewScanner(cfg)
}

func TestNewScanner(t *testing.T) {
	scanner := newTestScanner(t)
	assert.NotNil(t, scanner)
	assert.NotNil(t, scanner.detector)
}

func TestScanner_DetectsAWSKey(t *testing.T) {
	scanner := newTestScanner(t)

	// Use valid AWS key format: AKIA + 16 chars from [A-Z2-7]
	content := `
package main

const awsKey = "AKIATESTKEYEXAMPLE7A"
`

	findings, err := scanner.ScanContent(context.Background(), "test.go", content)
	require.NoError(t, err)
	assert.NotEmpty(t, findings, "should detect AWS access key")

	// Verify we found the AWS key
	found := false
	for _, f := range findings {
		if strings.Contains(f.RuleID, "aws") || strings.Contains(strings.ToLower(f.Description), "aws") {
			found = true
			assert.Greater(t, f.StartColumn, 0, "should have valid column number")
			assert.NotEmpty(t, f.Fingerprint, "should have fingerprint")
			assert.Equal(t, "AKIATESTKEYEXAMPLE7A", f.Secret)
			break
		}
	}
	assert.True(t, found, "should detect AWS access key pattern")
}

func TestScanner_DetectsGitHubToken(t *testing.T) {
	scanner := newTestScanner(t)

	// GitHub PAT format: ghp_ + exactly 36 alphanumeric chars
	content := `
package main

// GitHub personal access token
const token = "ghp_1234567890abcdefghijklmnopqrstuvwx"
`

	findings, err := scanner.ScanContent(context.Background(), "test.go", content)
	require.NoError(t, err)
	assert.NotEmpty(t, findings, "should detect GitHub token")

	// Verify the secret was detected (may match github or generic-api-key rule)
	assert.Equal(t, "ghp_1234567890abcdefghijklmnopqrstuvwx", findings[0].Secret)
}

func TestScanner_HandlesLargeFile(t *testing.T) {
	scanner := newTestScanner(t)

	// Create 2MB file (exceeds 1MB limit)
	content := strings.Repeat("x", 2_000_000)

	findings, err := scanner.ScanContent(context.Background(), "large.txt", content)
	require.NoError(t, err)
	assert.Empty(t, findings, "should skip large files gracefully")
}

func TestScanner_HandlesEmptyContent(t *testing.T) {
	scanner := newTestScanner(t)

	findings, err := scanner.ScanContent(context.Background(), "empty.txt", "")
	require.NoError(t, err)
	assert.Empty(t, findings)
}

func TestScanner_NoSecretsInCleanCode(t *testing.T) {
	scanner := newTestScanner(t)

	content := `
package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`

	findings, err := scanner.ScanContent(context.Background(), "main.go", content)
	require.NoError(t, err)
	assert.Empty(t, findings, "clean code should have no findings")
}

func TestCalculateFingerprint(t *testing.T) {
	// Test that same finding produces same fingerprint
	finding1 := Finding{
		File:      "test.go",
		StartLine: 10,
		RuleID:    "aws-access-key",
	}
	finding2 := Finding{
		File:      "test.go",
		StartLine: 10,
		RuleID:    "aws-access-key",
	}

	fp1 := calculateFingerprint(reportFindingFromFinding(finding1))
	fp2 := calculateFingerprint(reportFindingFromFinding(finding2))

	assert.Equal(t, fp1, fp2, "same findings should have same fingerprint")

	// Different line should produce different fingerprint
	finding3 := Finding{
		File:      "test.go",
		StartLine: 11,
		RuleID:    "aws-access-key",
	}
	fp3 := calculateFingerprint(reportFindingFromFinding(finding3))
	assert.NotEqual(t, fp1, fp3, "different line should have different fingerprint")
}

// Helper to convert our Finding to report.Finding for testing
func reportFindingFromFinding(f Finding) report.Finding {
	return report.Finding{
		File:      f.File,
		StartLine: f.StartLine,
		RuleID:    f.RuleID,
	}
}

func TestNewScannerWithIgnore(t *testing.T) {
	v := viper.New()
	v.SetConfigType("toml")
	require.NoError(t, v.ReadConfig(strings.NewReader(config.DefaultConfig)))

	var vc config.ViperConfig
	require.NoError(t, v.Unmarshal(&vc))

	cfg, err := vc.Translate()
	require.NoError(t, err)

	// Test with empty ignore file path
	scanner := NewScannerWithIgnore(cfg, "")
	assert.NotNil(t, scanner)
	assert.Empty(t, scanner.ignoreFilePath)

	// Test with non-existent ignore file (should just log warning)
	scanner2 := NewScannerWithIgnore(cfg, "/nonexistent/.gitleaksignore")
	assert.NotNil(t, scanner2)
}

func TestScannerWithIgnoreFile(t *testing.T) {
	// Create temp dir with ignore file
	tmpDir := t.TempDir()
	ignoreFile := filepath.Join(tmpDir, ".gitleaksignore")

	// Create scanner config
	v := viper.New()
	v.SetConfigType("toml")
	require.NoError(t, v.ReadConfig(strings.NewReader(config.DefaultConfig)))

	var vc config.ViperConfig
	require.NoError(t, v.Unmarshal(&vc))

	cfg, err := vc.Translate()
	require.NoError(t, err)

	// First scan without ignore file - should find secret
	scanner1 := NewScanner(cfg)
	content := `const awsKey = "AKIATESTKEYEXAMPLE7A"`

	findings1, err := scanner1.ScanContent(context.Background(), "test.go", content)
	require.NoError(t, err)
	require.NotEmpty(t, findings1, "Should find secret without ignore file")

	// Get the actual rule ID and line from the finding
	actualRuleID := findings1[0].RuleID
	actualLine := findings1[0].StartLine

	// Format: file:rule-id:start-line
	ignoreEntry := fmt.Sprintf("test.go:%s:%d\n", actualRuleID, actualLine)
	err = os.WriteFile(ignoreFile, []byte(ignoreEntry), 0644)
	require.NoError(t, err)

	// Create scanner with ignore file
	scanner2 := NewScannerWithIgnore(cfg, ignoreFile)

	findings2, err := scanner2.ScanContent(context.Background(), "test.go", content)
	require.NoError(t, err)

	// The finding should be ignored
	assert.Empty(t, findings2, "Secret should be ignored with ignore file")
}

func TestFindIgnoreFile(t *testing.T) {
	// Test with empty root path
	result := findIgnoreFile("")
	assert.Empty(t, result)

	// Test with non-existent directory
	result = findIgnoreFile("/nonexistent/path")
	assert.Empty(t, result)

	// Test with existing ignore file
	tmpDir := t.TempDir()
	ignoreFile := filepath.Join(tmpDir, ".gitleaksignore")
	err := os.WriteFile(ignoreFile, []byte("# ignore file\n"), 0644)
	require.NoError(t, err)

	result = findIgnoreFile(tmpDir)
	assert.Equal(t, ignoreFile, result)
}
