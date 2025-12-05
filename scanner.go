package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/zricethezav/gitleaks/v8/config"
	"github.com/zricethezav/gitleaks/v8/detect"
	"github.com/zricethezav/gitleaks/v8/report"
)

// Finding represents a detected secret with location information
type Finding struct {
	RuleID      string
	Description string
	Match       string
	Secret      string
	StartLine   int
	EndLine     int
	StartColumn int
	EndColumn   int
	Line        string // The line content where the secret was found
	Entropy     float32
	File        string
	Fingerprint string
}

// Scanner wraps gitleaks detection engine for LSP usage
type Scanner struct {
	detector       *detect.Detector
	config         config.Config
	ignoreFilePath string
	ignoreSet      map[string]struct{} // Set of fingerprints to ignore
}

// NewScanner creates a scanner with the provided config
func NewScanner(cfg config.Config) *Scanner {
	detector := detect.NewDetector(cfg)

	slog.Debug("scanner initialized",
		"config", cfg.Path,
		"rules", len(cfg.Rules))

	return &Scanner{
		detector:  detector,
		config:    cfg,
		ignoreSet: make(map[string]struct{}),
	}
}

// NewScannerWithIgnore creates a scanner with config and ignore file
func NewScannerWithIgnore(cfg config.Config, ignoreFilePath string) *Scanner {
	detector := detect.NewDetector(cfg)
	ignoreSet := make(map[string]struct{})

	if ignoreFilePath != "" {
		var err error
		ignoreSet, err = loadGitleaksIgnore(ignoreFilePath)
		if err != nil {
			slog.Warn("failed to load .gitleaksignore",
				"path", ignoreFilePath,
				"error", err)
		} else {
			slog.Info("loaded .gitleaksignore",
				"path", ignoreFilePath,
				"entries", len(ignoreSet))
		}
	}

	slog.Debug("scanner initialized",
		"config", cfg.Path,
		"rules", len(cfg.Rules),
		"ignorefile", ignoreFilePath)

	return &Scanner{
		detector:       detector,
		config:         cfg,
		ignoreFilePath: ignoreFilePath,
		ignoreSet:      ignoreSet,
	}
}

// loadGitleaksIgnore loads fingerprints from a .gitleaksignore file
func loadGitleaksIgnore(path string) (map[string]struct{}, error) {
	ignoreSet := make(map[string]struct{})

	file, err := os.Open(path)
	if err != nil {
		return ignoreSet, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	replacer := strings.NewReplacer("\\", "/")

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Normalize path separators
		parts := strings.Split(line, ":")
		switch len(parts) {
		case 3:
			// Global fingerprint: file:rule-id:start-line
			parts[0] = replacer.Replace(parts[0])
		case 4:
			// Commit fingerprint: commit:file:rule-id:start-line
			parts[1] = replacer.Replace(parts[1])
		default:
			slog.Warn("invalid .gitleaksignore entry", "line", line)
			continue
		}

		ignoreSet[strings.Join(parts, ":")] = struct{}{}
	}

	return ignoreSet, scanner.Err()
}

// ScanContent scans the provided content and returns findings
// Returns empty slice for files that are too large or have errors
func (s *Scanner) ScanContent(_ context.Context, filename, content string) ([]Finding, error) {
	const maxSize = 1_000_000 // 1MB limit

	if len(content) > maxSize {
		slog.Warn("file too large, skipping scan",
			"filename", filename,
			"size", len(content))
		return nil, nil
	}

	// Create a Fragment with the filename so fingerprints work correctly
	fragment := detect.Fragment{
		Raw:      content,
		FilePath: filename,
	}

	// Detect secrets using gitleaks Detect method
	gitleaksFindings := s.detector.Detect(fragment)

	// Convert gitleaks findings to our Finding type, filtering ignored ones
	findings := make([]Finding, 0, len(gitleaksFindings))
	for _, gf := range gitleaksFindings {
		// Check if this finding should be ignored
		globalFingerprint := fmt.Sprintf("%s:%s:%d", gf.File, gf.RuleID, gf.StartLine)
		if _, ignored := s.ignoreSet[globalFingerprint]; ignored {
			slog.Debug("ignoring finding",
				"fingerprint", globalFingerprint,
				"rule", gf.RuleID)
			continue
		}

		findings = append(findings, convertGitleaksFinding(gf))
	}

	return findings, nil
}

// convertGitleaksFinding converts gitleaks report.Finding to our Finding type
func convertGitleaksFinding(gf report.Finding) Finding {
	// Calculate fingerprint for this finding
	fingerprint := calculateFingerprint(gf)

	return Finding{
		RuleID:      gf.RuleID,
		Description: gf.Description,
		Match:       gf.Match,
		Secret:      gf.Secret,
		StartLine:   gf.StartLine,
		EndLine:     gf.EndLine,
		StartColumn: gf.StartColumn,
		EndColumn:   gf.EndColumn,
		Line:        gf.Line,
		Entropy:     gf.Entropy,
		File:        gf.File,
		Fingerprint: fingerprint,
	}
}

// calculateFingerprint creates a unique identifier for a finding
func calculateFingerprint(gf report.Finding) string {
	// Use file, line, and rule to create fingerprint
	data := fmt.Sprintf("%s:%d:%s", gf.File, gf.StartLine, gf.RuleID)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:8]) // First 8 bytes as hex
}
