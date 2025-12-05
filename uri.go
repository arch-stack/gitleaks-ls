package main

import (
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
)

// uriToPath converts a file:// URI to a filesystem path.
// Handles platform differences (Windows vs Unix).
func uriToPath(uri string) string {
	if uri == "" {
		return ""
	}

	// Parse the URI
	parsed, err := url.Parse(uri)
	if err != nil {
		// Fallback: simple prefix removal
		return strings.TrimPrefix(uri, "file://")
	}

	if parsed.Scheme != "file" {
		return uri
	}

	path := parsed.Path

	// On Windows, file URIs look like file:///C:/path
	// url.Parse gives us /C:/path, we need to remove the leading slash
	if runtime.GOOS == "windows" && len(path) >= 3 {
		// Check for /C: or /D: pattern
		if path[0] == '/' && path[2] == ':' {
			path = path[1:]
		}
	}

	// Convert to native path separators
	return filepath.FromSlash(path)
}

// pathToURI converts a filesystem path to a file:// URI.
// Handles platform differences (Windows vs Unix).
func pathToURI(path string) string {
	if path == "" {
		return ""
	}

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	// Convert to forward slashes for URI
	uriPath := filepath.ToSlash(absPath)

	// On Windows, paths like C:/foo need an extra leading slash
	// to become file:///C:/foo
	if runtime.GOOS == "windows" && len(uriPath) >= 2 && uriPath[1] == ':' {
		return "file:///" + uriPath
	}

	// Unix paths already start with /, so file:// + /path = file:///path
	return "file://" + uriPath
}
