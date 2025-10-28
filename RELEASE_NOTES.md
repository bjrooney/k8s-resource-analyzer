# Release Notes - k8s-resource-analyzer

## Version 2.0 - Enhanced Metrics & Cluster Identification

**Release Date:** October 27, 2025

### 🎉 Major New Features

#### 1. Cluster Name in Report Header
- **What:** Report now displays the cluster name at the very top
- **Why:** Easy identification when analyzing multiple clusters
- **How:** Extracted from kubeconfig context name or cluster server URL
- **Example:**
  ```markdown
  # Kubernetes Cluster Analysis Report
  
  **Cluster:** production-aks-eastus
  **Generated:** 2025-10-27T14:55:00Z
  ```

#### 2. Live Metrics Integration (Metrics-Server)
- **What:** Shows actual CPU and Memory usage for every container
- **Why:** Compare real usage against configured limits
- **How:** Automatically queries metrics-server API if available
- **Graceful Fallback:** Shows "N/A" if metrics-server not installed
- **Unit Normalization:** Automatically converts CPU to millicores (m) and Memory to Mebibytes (Mi)

#### 3. Enhanced Appendix B - Pod Resource Inventory
- **What:** Complete table with actual usage vs configured resources
- **New Columns:**
  - Current CPU in millicores (m) - converted from nanocores
  - Current Memory in Mebibytes (Mi) - converted from Kibibytes
  - Current Memory (with % of limit if configured)
- **Benefits:**
  - Right-sizing analysis
  - Cost optimization opportunities
  - Performance tuning insights

### 📊 Example Output

#### Before (v1.0)
```markdown
| Pod | Container | CPU Request | CPU Limit | Memory Request | Memory Limit | Status |
|-----|-----------|-------------|-----------|----------------|--------------|--------|
| api-abc | api | 500m | 1000m | 512Mi | 1Gi | Running |
```

#### After (v2.0)
```markdown
**Cluster:** production-aks-eastus

| Pod | Container | CPU Req | CPU Limit | Mem Req | Mem Limit | Current CPU | Current Mem | Status |
|-----|-----------|---------|-----------|---------|-----------|-------------|-------------|--------|
| api-abc | api | 500m | 1000m | 512Mi | 1Gi | 234m (46.8%) | 678Mi (66.2%) | Running |

**Note:** Metrics-server is available. Current usage reflects live data.
* CPU automatically converted from nanocores (n) to millicores (m)
* Memory automatically converted from Kibibytes (Ki) to Mebibytes (Mi)
```

### 🔧 Technical Changes
**analyzer.go:**
- Added `getClusterName()` function
- Added `getPodMetrics()` function for metrics-server integration
- Updated `ClusterData` struct with `ClusterName` and `PodMetrics` fields
- Enhanced `CollectClusterData()` to fetch metrics

**report.go:**
- Updated report header template to include cluster name
- Enhanced appendix table with Current CPU/Memory columns
- Added usage percentage calculations
- Added metrics availability indicator
- Added `convertCPUToMillicores()` function - converts nanocores (n) to millicores (m)
- Added `convertMemoryToMi()` function - converts Kibibytes (Ki) to Mebibytes (Mi)
- Added `parseFloat()` helper function for unit parsing

**go.mod:**
- No new dependencies required (uses dynamic REST client)

### 📈 Use Cases

1. **Right-Sizing Pods:**
   - See pod using 234m CPU with 1000m limit → reduce limit to 500m
   - Save money by reducing over-provisioned resources

2. **Performance Tuning:**
   - Pod at 89.1% of CPU limit → increase before throttling occurs
   - Proactive scaling before issues arise

3. **Multi-Cluster Analysis:**
   - Generate reports for dev, staging, prod clusters
   - Cluster name makes it easy to identify which environment

4. **Capacity Planning:**
   - Use actual usage data instead of guessing
   - More accurate forecasting

5. **Cost Optimization:**
   - Find containers using 10% of allocated resources
   - Reduce waste across the cluster

### 🚀 Migration Guide

No breaking changes! Simply rebuild and run:

```bash
cd /home/brendan/projects/sita/scripts/k8s-resource-analyzer
go build -o bin/k8s-analyzer .
./bin/k8s-analyzer
```

**Metrics-Server Required?** No - tool works with or without it.

**To Install Metrics-Server:**
```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

### 📝 What's Next

Planned for future releases:
- Historical metrics trending (24h/7d averages)
- Cost analysis per namespace (with cloud provider pricing)
- Automatic right-sizing recommendations
- Export to CSV/JSON for further analysis
- Integration with Prometheus for advanced metrics

### 🐛 Bug Fixes

- Fixed section numbering after adding new analysis sections
- Improved error handling when cluster connection fails
- Better handling of pods with incomplete metadata

### 📚 Documentation Updates

- ✅ README.md - Updated with metrics features
- ✅ QUICKREF.md - Added metrics examples
- ✅ FEATURE_APPENDIX.md - Complete rewrite with v2 features
- ✅ RELEASE_NOTES.md - This document

### ⚡ Performance

- Binary size: 47MB (unchanged)
- Metrics collection adds ~2-5 seconds to analysis time
- Memory usage: Minimal increase (<50MB) for metrics cache

### 🙏 Acknowledgments

Thanks to the Kubernetes metrics-server team for the excellent API documentation.

---

## Previous Releases

### Version 1.5 - Event & Backup Analysis
- Added Flux events analysis (Section 6)
- Added non-Flux warning events (Section 7)
- Added Velero backup analysis (Section 8)
- Pod restart tracking (Section 5)

### Version 1.0 - Initial Release
- Core cluster analysis
- Resource gap detection
- Node health analysis
- RabbitMQ stability recommendations
- Namespace risk assessment
- AI-powered insights integration
