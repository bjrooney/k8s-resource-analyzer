#!/bin/bash

# Quick start script for k8s-resource-analyzer
# This script helps you get started quickly

set -e

echo "🚀 Kubernetes Resource Analyzer - Quick Start"
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21 or later."
    echo "   Visit: https://golang.org/doc/install"
    exit 1
fi

echo "✅ Go found: $(go version)"

# Check if kubectl is installed and configured
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl is not installed. Please install kubectl."
    echo "   Visit: https://kubernetes.io/docs/tasks/tools/"
    exit 1
fi

echo "✅ kubectl found: $(kubectl version --client --short 2>/dev/null || echo 'kubectl installed')"

# Check cluster connectivity
if ! kubectl cluster-info &> /dev/null; then
    echo "⚠️  Warning: Cannot connect to Kubernetes cluster"
    echo "   Make sure your kubeconfig is properly configured"
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
else
    echo "✅ Connected to cluster: $(kubectl config current-context)"
fi

# Download dependencies
echo ""
echo "📦 Downloading Go dependencies..."
go mod download

# Build the application
echo ""
echo "🔨 Building application..."
go build -o k8s-analyzer .

echo ""
echo "✅ Build complete!"
echo ""

# Check for API key
if [ -z "$OPENAI_API_KEY" ] && [ -z "$AZURE_OPENAI_API_KEY" ]; then
    echo "ℹ️  No AI API key detected. Analysis will run without AI insights."
    echo "   To enable AI analysis, set one of:"
    echo "   - OPENAI_API_KEY=your-key"
    echo "   - AZURE_OPENAI_API_KEY=your-key"
    echo ""
    read -p "Run analysis without AI? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 0
    fi
else
    echo "✅ AI API key detected - AI analysis will be enabled"
fi

# Run the analyzer
echo ""
echo "🔍 Starting cluster analysis..."
echo ""
./k8s-analyzer

echo ""
echo "✨ Analysis complete!"
echo ""
echo "📄 Report saved to: cluster-analysis-report.md"
echo ""
echo "Next steps:"
echo "  1. Review the report: cat cluster-analysis-report.md"
echo "  2. Or open in your editor: code cluster-analysis-report.md"
echo "  3. Address critical issues first"
echo ""
