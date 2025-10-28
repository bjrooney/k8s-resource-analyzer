# k8s-resource-analyzer Quick Reference

## Quick Start (30 seconds)
```bash
./quickstart.sh
```

## Build & Run
```bash
make build              # Build binary
make run                # Build and run analysis
make run-ai             # Run with AI (needs API key)
make clean              # Remove build artifacts
```

## Direct Usage
```bash
# Basic analysis (auto-names file: clustername-YYYYMMDD.md)
./bin/k8s-analyzer

# With AI (generates resource suggestions for ALL namespaces)
export OPENAI_API_KEY="sk-..."
./bin/k8s-analyzer

# Custom options
./bin/k8s-analyzer \
  -kubeconfig=/path/to/config \
  -output=custom-report.md \
  -ai-provider=azure \
  -ai-endpoint=https://....openai.azure.com/
```

## Command-Line Flags
| Flag | Description | Default |
|------|-------------|---------|
| `-kubeconfig` | Path to kubeconfig | `~/.kube/config` |
| `-output` | Output file path | `<cluster>-YYYYMMDD.md` (auto) |
| `-ai-provider` | AI provider (openai/azure) | `openai` |
| `-ai-endpoint` | Azure OpenAI endpoint | (none) |
| `-ai-model` | AI model to use | `gpt-4o` |

## Output Filename
By default, the tool automatically generates a filename based on:
- **Cluster name** (from kubeconfig context)
- **Date** (YYYYMMDD format)
- Example: `production-aks-eastus-20251027.md`

To use a custom filename, specify `-output`:
```bash
./bin/k8s-analyzer -output=my-custom-report.md
```
## Environment Variables
```bash
OPENAI_API_KEY          # OpenAI API key
AZURE_OPENAI_API_KEY    # Azure OpenAI API key
KUBECONFIG              # Path to kubeconfig
```

## What It Analyzes

✅ **Cluster Health** - Overall status and key metrics  
✅ **Critical Issues** - Top 5 issues with recommendations  
✅ **Resource Gaps** - Missing requests/limits  
✅ **Node Health** - Utilization and OOM events  
✅ **Pod Restarts** - Restart analysis (24h & 7d)  
✅ **RabbitMQ** - Stability and eviction protection  
✅ **Namespaces** - Per-namespace risk assessment  
✅ **AI Insights** - Enhanced recommendations (optional)  

## Output Format
- Markdown report with sections
- Tables, code examples, YAML configs
- Actionable recommendations
- Risk-based prioritization

## Common Workflows

### Analyze Production Cluster
```bash
kubectl config use-context production
./bin/k8s-analyzer -output=prod-$(date +%Y%m%d).md
```

### Compare Multiple Clusters
```bash
for ctx in $(kubectl config get-contexts -o name); do
  kubectl config use-context $ctx
  ./bin/k8s-analyzer -output="report-${ctx}.md"
done
```

### CI/CD Integration
```bash
export OPENAI_API_KEY=$SECRET_KEY
./bin/k8s-analyzer -output=analysis-${BUILD_ID}.md
# Upload as artifact
```

## Troubleshooting

### "Cannot connect to cluster"
- Check: `kubectl cluster-info`
- Verify kubeconfig path
- Test: `kubectl get nodes`

### "No AI API key found"
- Set: `export OPENAI_API_KEY="sk-..."`
- Or: `export AZURE_OPENAI_API_KEY="..."`
- Analysis still works without AI

### Build Fails
```bash
go mod tidy             # Fix dependencies
go clean -cache         # Clear cache
make clean && make build
```

## File Locations
| File | Purpose |
|------|---------|
| `bin/k8s-analyzer` | Binary (46MB) |
| `cluster-analysis-report.md` | Default output |
| `config.example.yaml` | Configuration template |
| `README.md` | Full documentation |
| `PROJECT_SUMMARY.md` | Project overview |

## Key Report Sections
1. Cluster Health Summary
2. Critical Issues (Top 5)
3. Resource Management
4. Node Analysis
5. RabbitMQ Stability
6. Namespace Analysis
7. AI Insights
8. Appendix

## Support
- Check `README.md` for detailed docs
- Run `./examples.sh` for usage examples
- Run `./test.sh` to verify setup
- View `PROJECT_SUMMARY.md` for overview

## One-Liner Examples
```bash
# Quick analysis
./bin/k8s-analyzer

# Different output name
./bin/k8s-analyzer -output=my-report.md

# Specific kubeconfig
./bin/k8s-analyzer -kubeconfig=~/.kube/staging

# With OpenAI
OPENAI_API_KEY=sk-... ./bin/k8s-analyzer

# Help
./bin/k8s-analyzer -help
```

## NEW: Enhanced Features

### 1. Auto-Generated Filenames
Reports are automatically named: `<cluster-name>-YYYYMMDD.md`
- Easy to track reports over time
- Identify which cluster was analyzed
- Example: `production-aks-eastus-20251027.md`

### 2. AI Suggestions for ALL Namespaces
When AI is enabled, the tool now:
- Scans **all namespaces** for missing resource configurations
- Generates suggestions for **every namespace** that needs them
- No longer limited to specific namespaces

```bash
export OPENAI_API_KEY="sk-..."
./bin/k8s-analyzer
```

Output:
```
🎯 Generating AI resource suggestions for all namespaces with missing resources...
   Found 12 namespaces with missing resource configurations
   ✅ Generated suggestions for 8 pods in namespace 'cgh'
   ✅ Generated suggestions for 3 pods in namespace 'velero'
   ✅ Generated suggestions for 15 pods in namespace 'kube-system'
   ✅ Generated suggestions for 5 pods in namespace 'flux-system'
   ...
```

### 3. Cluster Name in Report Header
The report shows the **cluster name** at the top for easy identification when analyzing multiple clusters.

### 4. Pod Resource Appendix with Live Metrics

The report includes **Appendix B** with a complete inventory of all running pods:

### Features
- Every container's CPU/Memory requests and limits
- **Actual CPU and Memory usage** (when metrics-server is available)
- Usage vs configured limits analysis
- Organized by namespace
- Summary statistics showing configuration coverage
- Identifies pods missing resource settings

### Use Cases
- **Resource Audit**: Quickly find misconfigured pods
- **Capacity Planning**: Export complete allocation data
- **Compliance**: Verify all pods meet resource standards
- **Troubleshooting**: Cross-reference configs with issues

### Useful Commands (from Appendix D)
```bash
# Find pods without requests
kubectl get pods -A -o json | jq -r '.items[] | select(.spec.containers[].resources.requests == null) | "\(.metadata.namespace)/\(.metadata.name)"'

# Resource usage per namespace
kubectl top pods -n <namespace>

# View pod resources
kubectl get pod <name> -n <namespace> -o jsonpath='{.spec.containers[*].resources}'
```

---
**TIP**: Run `./quickstart.sh` for an interactive guided setup!
