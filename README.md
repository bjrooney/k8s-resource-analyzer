# Kubernetes Resource Analyzer with AI

A comprehensive Go-based tool for analyzing Kubernetes cluster health, resource allocation, and stability. This tool integrates with OpenAI/Azure OpenAI to provide intelligent insights and recommendations for cluster optimization.

## Features

- 🔍 **Comprehensive Cluster Analysis**: Scans all pods, nodes, and events
- 🤖 **AI-Powered Insights**: Uses OpenAI/Azure OpenAI to supplement findings with intelligent recommendations
- 📊 **Resource Gap Detection**: Identifies pods missing resource requests/limits
- 🔴 **OOM Event Tracking**: Monitors and reports OOMKilled events
- 🏥 **Node Health Analysis**: Detects poorly balanced nodes and resource pressure
- 🐰 **RabbitMQ Stability**: Special analysis for RabbitMQ workload protection
- 📦 **Namespace-by-Namespace Analysis**: Risk-based namespace evaluation
- 📝 **Detailed Markdown Reports**: Generate comprehensive analysis reports

## Prerequisites

- Go 1.21 or later
- Access to a Kubernetes cluster
- kubectl configured with cluster access
- (Optional) OpenAI API key or Azure OpenAI credentials for AI-enhanced analysis

## Installation

### 1. Clone or navigate to the project directory

```bash
cd /home/brendan/projects/sita/scripts/k8s-resource-analyzer
```

### 2. Download dependencies

```bash
go mod download
```

### 3. Build the binary

```bash
go build -o k8s-analyzer .
```

## Configuration

### Kubernetes Access

The tool uses your kubectl configuration by default (`~/.kube/config`). You can specify a different kubeconfig:

```bash
./k8s-analyzer -kubeconfig=/path/to/kubeconfig
```

### AI Integration (Optional)

Set one of these environment variables to enable AI-enhanced analysis:

**For OpenAI:**
```bash
export OPENAI_API_KEY="sk-..."
```

**For Azure OpenAI:**
```bash
export AZURE_OPENAI_API_KEY="your-key"
```

## Usage

### Basic Usage

```bash
./k8s-analyzer
```

This will:
1. Connect to your Kubernetes cluster
2. Collect pod, node, and event data
3. Perform comprehensive analysis
4. Generate a report: `cluster-analysis-report.md`

### Advanced Options

```bash
./k8s-analyzer \
  -kubeconfig=/path/to/kubeconfig \
  -output=my-report.md \
  -ai-provider=openai
```

#### Command-Line Flags

- `-kubeconfig`: Path to kubeconfig file (default: `~/.kube/config`)
- `-output`: Output file path (default: `cluster-analysis-report.md`)
- `-ai-provider`: AI provider to use: `openai` or `azure` (default: `openai`)
- `-ai-endpoint`: Azure OpenAI endpoint URL (required if using Azure)
- `-ai-model`: AI model to use (default: `gpt-4o`)
  - Available: `gpt-4o`, `gpt-4o-mini`, `gpt-4-turbo`, `gpt-3.5-turbo`

### Examples

**Basic analysis without AI:**
```bash
./k8s-analyzer
```

**With OpenAI (default GPT-4o):**
```bash
export OPENAI_API_KEY="sk-..."
./k8s-analyzer
```

**With GPT-4o-mini (faster/cheaper):**
```bash
export OPENAI_API_KEY="sk-..."
./k8s-analyzer -ai-model=gpt-4o-mini
```

**With Azure OpenAI:**
```bash
export AZURE_OPENAI_API_KEY="your-key"
./k8s-analyzer \
  -ai-provider=azure \
  -ai-endpoint=https://your-instance.openai.azure.com/ \
  -output=analysis.md
```

## Report Sections

The generated report includes:

1. **Cluster Health Summary**: High-level overview of cluster status
2. **Critical Issues**: Top 3-5 most critical problems with actionable recommendations
3. **Resource Management**: Analysis of missing requests/limits and their impact
4. **Node Analysis**: Node utilization, OOM events, and autoscaling recommendations
5. **Pod Restart Analysis**: Pods with restarts in last 24 hours and 7 days
6. **RabbitMQ Stability**: Specific recommendations for RabbitMQ workload protection
7. **Namespace Analysis**: Detailed per-namespace breakdown with risk levels
8. **AI Insights** (if enabled): AI-generated recommendations and strategic insights

## What the Tool Analyzes

### Resource Management
- Pods missing CPU/memory requests
- Pods missing CPU/memory limits
- Impact on Velero backups
- Impact on system pod stability
- Short-lived job patterns

### Node Health
- High CPU/memory utilization
- Poorly balanced node pools
- Resource pressure indicators
- Autoscaling bottlenecks

### Application Stability
- OOMKilled events
- Pod eviction history
- Priority class usage
- PodDisruptionBudget status

### Special Workload Analysis
- RabbitMQ stability and priority
- Job completion patterns
- Critical service protection

## AI Analysis Features

When AI integration is enabled, the tool provides:

- **Enhanced Recommendations**: Context-aware suggestions based on your specific cluster configuration
- **Risk Assessment**: Intelligent prioritization of issues based on impact
- **Automation Suggestions**: Recommendations for policies, quotas, and preventive measures
- **Strategic Insights**: Long-term optimization strategies

## Example Output

```markdown
# Kubernetes Cluster Analysis Report

**Generated:** 2025-10-27T10:30:00Z

---

## 1. Cluster Health Summary

🟡 **Overall Health**: DEGRADED

### Key Metrics

| Metric | Value |
|--------|-------|
| Total Pods | 234 |
| Total Nodes | 12 |
| Pods Missing Resources | 87 |
| OOM Events (Recent) | 5 |
...
```

## Development

### Project Structure

```
k8s-resource-analyzer/
├── main.go         # Entry point and CLI setup
├── analyzer.go     # Core analysis logic
├── ai.go           # AI integration (OpenAI/Azure)
├── report.go       # Markdown report generation
├── go.mod          # Go module dependencies
└── README.md       # This file
```

### Testing

```bash
# Run against a test cluster
./k8s-analyzer -kubeconfig=~/.kube/test-config

# Dry run without AI
unset OPENAI_API_KEY
./k8s-analyzer
```

## Troubleshooting

### "Error building kubeconfig"
- Ensure kubectl is properly configured
- Verify you have access to the cluster: `kubectl get nodes`

### "Error creating kubernetes client"
- Check your kubeconfig file permissions
- Verify cluster connectivity

### "Warning: Could not initialize AI client"
- Check that your API key is set correctly
- For Azure, ensure the endpoint URL is correct

### "AI analysis failed"
- Verify API key is valid
- Check network connectivity to OpenAI/Azure endpoints
- Review any rate limiting or quota issues

## Best Practices

1. **Run regularly**: Schedule weekly or bi-weekly cluster scans
2. **Compare reports**: Track improvements over time
3. **Start with critical namespaces**: Focus on high-risk areas first
4. **Test resource changes**: Use non-production environments to validate recommendations
5. **Monitor after changes**: Watch cluster metrics after implementing suggestions

## Security Considerations

- The tool requires **read-only** access to the Kubernetes API
- API keys are only used for AI analysis and not stored
- Reports may contain sensitive cluster information - treat them as confidential
- Consider using Kubernetes RBAC to limit tool permissions

## Contributing

Suggestions and improvements are welcome! Areas for enhancement:

- Additional workload-specific analysis (databases, web servers, etc.)
- Historical trend analysis
- Integration with metrics servers (Prometheus, etc.)
- Custom policy definition and validation
- HTML report generation

## License

MIT License - feel free to use and modify as needed.

## Support

For issues or questions:
1. Check the troubleshooting section
2. Review Kubernetes documentation
3. Consult with your SRE team

## Credits

Built with:
- [client-go](https://github.com/kubernetes/client-go) - Kubernetes Go client
- [go-openai](https://github.com/sashabaranov/go-openai) - OpenAI Go SDK
