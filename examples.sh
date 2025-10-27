#!/bin/bash

# Example usage scenarios for k8s-resource-analyzer

echo "=== Kubernetes Resource Analyzer - Usage Examples ==="
echo ""

# Example 1: Basic usage
echo "1. Basic Analysis (no AI):"
echo "   ./bin/k8s-analyzer"
echo ""

# Example 2: With OpenAI
echo "2. Analysis with OpenAI:"
echo "   export OPENAI_API_KEY='sk-...'"
echo "   ./bin/k8s-analyzer"
echo ""

# Example 3: With Azure OpenAI
echo "3. Analysis with Azure OpenAI:"
echo "   export AZURE_OPENAI_API_KEY='your-key'"
echo "   ./bin/k8s-analyzer -ai-provider=azure -ai-endpoint='https://your-resource.openai.azure.com/'"
echo ""

# Example 4: Custom kubeconfig
echo "4. Using custom kubeconfig:"
echo "   ./bin/k8s-analyzer -kubeconfig=/path/to/kubeconfig"
echo ""

# Example 5: Custom output
echo "5. Custom output file:"
echo "   ./bin/k8s-analyzer -output=my-cluster-report.md"
echo ""

# Example 6: Analyzing specific cluster context
echo "6. Analyzing specific cluster context:"
echo "   kubectl config use-context production"
echo "   ./bin/k8s-analyzer -output=production-report.md"
echo ""

# Example 7: Quick analysis of all your clusters
echo "7. Analyze all clusters:"
cat << 'EOF'
   for context in $(kubectl config get-contexts -o name); do
       echo "Analyzing $context..."
       ./bin/k8s-analyzer \
           -kubeconfig=$HOME/.kube/config \
           -output="report-${context}.md"
       kubectl config use-context $context
   done
EOF
echo ""

# Example 8: Using Make commands
echo "8. Using Make commands:"
echo "   make build          # Build the binary"
echo "   make run            # Build and run"
echo "   make run-ai         # Run with AI (requires API key)"
echo "   make clean          # Clean build artifacts"
echo "   make help           # Show all commands"
echo ""

# Example 9: Running with Docker (if you create a Dockerfile)
echo "9. Running with Docker (future):"
echo "   docker build -t k8s-analyzer ."
echo "   docker run -v ~/.kube:/root/.kube k8s-analyzer"
echo ""

# Example 10: CI/CD Integration
echo "10. CI/CD Integration example:"
cat << 'EOF'
   # In your pipeline:
   - export OPENAI_API_KEY=$SECRET_OPENAI_KEY
   - ./bin/k8s-analyzer -output=analysis-${CI_COMMIT_SHA}.md
   - Upload report as artifact
EOF
echo ""
