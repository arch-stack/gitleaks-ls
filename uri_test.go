package main

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUriToPath(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		wantUnix string
		wantWin  string
	}{
		{
			name:     "empty",
			uri:      "",
			wantUnix: "",
			wantWin:  "",
		},
		{
			name:     "unix path",
			uri:      "file:///home/user/project/file.go",
			wantUnix: "/home/user/project/file.go",
			wantWin:  "/home/user/project/file.go", // Not a valid Windows path anyway
		},
		{
			name:     "windows path",
			uri:      "file:///C:/Users/user/project/file.go",
			wantUnix: "/C:/Users/user/project/file.go",
			wantWin:  "C:/Users/user/project/file.go",
		},
		{
			name:     "spaces in path",
			uri:      "file:///home/user/my%20project/file.go",
			wantUnix: "/home/user/my project/file.go",
			wantWin:  "/home/user/my project/file.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uriToPath(tt.uri)
			if runtime.GOOS == "windows" {
				// Normalize for Windows backslashes
				assert.Contains(t, []string{tt.wantWin, toBackslash(tt.wantWin)}, result)
			} else {
				assert.Equal(t, tt.wantUnix, result)
			}
		})
	}
}

func TestPathToUri(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Run("windows absolute path", func(t *testing.T) {
			result := pathToURI("C:\\Users\\user\\file.go")
			assert.Equal(t, "file:///C:/Users/user/file.go", result)
		})
	} else {
		t.Run("unix absolute path", func(t *testing.T) {
			result := pathToURI("/home/user/file.go")
			assert.Equal(t, "file:///home/user/file.go", result)
		})
	}

	t.Run("empty path", func(t *testing.T) {
		result := pathToURI("")
		assert.Equal(t, "", result)
	})
}

func toBackslash(s string) string {
	result := ""
	for _, c := range s {
		if c == '/' {
			result += "\\"
		} else {
			result += string(c)
		}
	}
	return result
}
