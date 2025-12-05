#!/bin/bash
# Quick test script for gitleaks-ls

cd "$(dirname "$0")"

echo "🔍 Gitleaks Language Server - Quick Test"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check if binary exists
if [ ! -f gitleaks-ls ]; then
    echo "❌ gitleaks-ls binary not found. Building..."
    go build -o gitleaks-ls . || exit 1
    echo "✅ Build complete"
    echo ""
fi

# Check if test file exists
if [ ! -f examples/test_file.go ]; then
    echo "❌ Test file not found: examples/test_file.go"
    exit 1
fi

echo "📋 Test file contains 2 secrets:"
echo "   Line 9:  AWS Access Key"
echo "   Line 12: GitHub Personal Access Token"
echo ""
echo "🎯 Once Neovim opens:"
echo "   • Move to line 9 or 12"
echo "   • Press K for hover documentation"
echo "   • Press <leader>ca for code actions"
echo "   • Use ]d and [d to navigate diagnostics"
echo ""
echo "Press Enter to continue..."
read

# Launch Neovim with test config
exec nvim -u test-lsp.lua examples/test_file.go
