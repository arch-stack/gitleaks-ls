package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectFiles(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create some test files
	files := map[string]string{
		"main.go":           "package main\n",
		"util.go":           "package main\n",
		"src/handler.go":    "package src\n",
		"src/model.go":      "package src\n",
		".hidden":           "hidden file\n",
		".git/config":       "git config\n",
		"node_modules/x.js": "module\n",
		"image.png":         "binary\n",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	}

	// Collect files
	collected, err := collectFiles(tmpDir)
	require.NoError(t, err)

	// Should include .go files but not hidden, binary, or node_modules
	assert.GreaterOrEqual(t, len(collected), 4, "Should collect at least 4 .go files")

	// Check specific exclusions
	for _, f := range collected {
		assert.NotContains(t, f, ".git", "Should not include .git directory")
		assert.NotContains(t, f, "node_modules", "Should not include node_modules")
		assert.NotContains(t, f, ".hidden", "Should not include hidden files")
		assert.NotContains(t, f, ".png", "Should not include binary files")
	}
}

func TestCollectFiles_WithGitignore(t *testing.T) {
	tmpDir := t.TempDir()

	// Create gitignore
	gitignore := "*.log\nbuild/\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(gitignore), 0644))

	// Create files
	files := map[string]string{
		"main.go":      "package main\n",
		"debug.log":    "log file\n",
		"build/out.go": "build output\n",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	}

	collected, err := collectFiles(tmpDir)
	require.NoError(t, err)

	// Should include main.go but not debug.log or build/
	assert.Len(t, collected, 1)
	assert.Contains(t, collected[0], "main.go")
}

func TestIsBinaryExtension(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"main.go", false},
		{"script.py", false},
		{"index.html", false},
		{"image.png", true},
		{"photo.jpg", true},
		{"archive.zip", true},
		{"binary.exe", true},
		{"font.woff", true},
		{"video.mp4", true},
		{"data.db", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBinaryExtension(tt.name)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBinaryContent(t *testing.T) {
	// PNG magic bytes
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	assert.True(t, isBinaryContent(pngBytes))

	// Plain text
	textBytes := []byte("package main\n\nfunc main() {}\n")
	assert.False(t, isBinaryContent(textBytes))

	// Empty content
	assert.False(t, isBinaryContent([]byte{}))
}

func TestPathToURI(t *testing.T) {
	uri := pathToURI("/tmp/test.go")
	assert.True(t, len(uri) > 0)
	assert.Contains(t, uri, "file://")
	assert.Contains(t, uri, "test.go")
}

func TestScanWorkspace(t *testing.T) {
	// Setup server
	rootURI := "file:///tmp/test"
	ctx := context.Background()

	// Initialize server
	err := SetupServer("")
	require.NoError(t, err)

	// Create temp workspace with a secret
	tmpDir := t.TempDir()
	secretFile := filepath.Join(tmpDir, "secret.go")
	secretContent := `package main
const awsKey = "AKIATESTKEYEXAMPLE7A"
`
	require.NoError(t, os.WriteFile(secretFile, []byte(secretContent), 0644))

	cleanFile := filepath.Join(tmpDir, "clean.go")
	cleanContent := `package main
func main() {}
`
	require.NoError(t, os.WriteFile(cleanFile, []byte(cleanContent), 0644))

	// Scan workspace
	result, err := globalServer.ScanWorkspace(ctx, tmpDir, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 2, result.TotalFiles)
	assert.Equal(t, 2, result.ScannedFiles)
	assert.Equal(t, 0, result.SkippedFiles)
	assert.GreaterOrEqual(t, result.TotalFindings, 1, "Should find at least 1 secret")
	assert.GreaterOrEqual(t, len(result.Findings), 1, "Should have findings for at least 1 file")

	// Verify the secret was found
	found := false
	for _, findings := range result.Findings {
		if len(findings) > 0 {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find the AWS key secret")

	_ = rootURI // unused but keeps the pattern consistent
}

func TestScanWorkspace_EmptyPath(t *testing.T) {
	err := SetupServer("")
	require.NoError(t, err)

	result, err := globalServer.ScanWorkspace(context.Background(), "", nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestLoadGitignore(t *testing.T) {
	tmpDir := t.TempDir()

	// Create gitignore
	gitignoreContent := `# Comment
*.log
build/
temp

# Another comment
*.tmp
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(gitignoreContent), 0644))

	gi := loadGitignore(tmpDir)
	require.NotNil(t, gi)

	// Test pattern matching
	assert.True(t, gi.MatchesPath("debug.log"))
	assert.True(t, gi.MatchesPath("build/out.go"))
	assert.True(t, gi.MatchesPath("temp"))
	assert.True(t, gi.MatchesPath("file.tmp"))
	assert.False(t, gi.MatchesPath("main.go"))
}

func TestLoadGitignore_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	gi := loadGitignore(tmpDir)
	assert.Nil(t, gi)
}
