#!/bin/bash

# Test script to verify k8s-analyzer works correctly

set -e

echo "🧪 Testing k8s-resource-analyzer"
echo "================================="
echo ""

# Check if binary exists
if [ ! -f "./bin/k8s-analyzer" ]; then
    echo "❌ Binary not found. Building..."
    make build
fi

echo "✅ Binary found"
echo ""

# Check cluster connectivity
echo "🔍 Checking cluster connectivity..."
if ! kubectl cluster-info &> /dev/null; then
    echo "❌ Cannot connect to cluster"
    echo "Please ensure kubectl is configured and you have cluster access"
    exit 1
fi

CURRENT_CONTEXT=$(kubectl config current-context)
echo "✅ Connected to: $CURRENT_CONTEXT"
echo ""

# Get basic cluster info
echo "📊 Cluster Information:"
echo "  Nodes: $(kubectl get nodes --no-headers 2>/dev/null | wc -l)"
echo "  Namespaces: $(kubectl get namespaces --no-headers 2>/dev/null | wc -l)"
echo "  Pods: $(kubectl get pods --all-namespaces --no-headers 2>/dev/null | wc -l)"
echo ""

# Run analyzer
echo "🚀 Running analyzer..."
OUTPUT_FILE="test-report-$(date +%Y%m%d-%H%M%S).md"

if [ -n "$OPENAI_API_KEY" ] || [ -n "$AZURE_OPENAI_API_KEY" ]; then
    echo "   AI analysis enabled"
    ./bin/k8s-analyzer -output="$OUTPUT_FILE"
else
    echo "   AI analysis disabled (no API key found)"
    ./bin/k8s-analyzer -output="$OUTPUT_FILE"
fi

echo ""

# Check if report was created
if [ -f "$OUTPUT_FILE" ]; then
    echo "✅ Report generated successfully: $OUTPUT_FILE"
    echo ""
    
    # Show report summary
    echo "📄 Report Summary:"
    echo "  File size: $(du -h "$OUTPUT_FILE" | cut -f1)"
    echo "  Lines: $(wc -l < "$OUTPUT_FILE")"
    echo ""
    
    # Show first few lines
    echo "📋 Report Preview (first 30 lines):"
    echo "-----------------------------------"
    head -n 30 "$OUTPUT_FILE"
    echo "-----------------------------------"
    echo ""
    echo "To view the full report:"
    echo "  cat $OUTPUT_FILE"
    echo "  # or"
    echo "  code $OUTPUT_FILE"
else
    echo "❌ Report was not generated"
    exit 1
fi

echo ""
echo "✨ Test completed successfully!"
